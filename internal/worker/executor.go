package worker

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/platform/detect"
	dockerpkg "github.com/VatsalP117/hostbox/internal/platform/docker"
	"github.com/VatsalP117/hostbox/internal/platform/sanitize"
	"github.com/VatsalP117/hostbox/internal/repository"
)

// PostBuildHook is called after a build completes.
type PostBuildHook interface {
	OnBuildSuccess(ctx context.Context, project *models.Project, deployment *models.Deployment) error
	OnBuildFailure(ctx context.Context, project *models.Project, deployment *models.Deployment, buildErr error) error
}

// PostBuildCancellationHook removes side effects created by a successful build
// when another workflow cancels the deployment while its success hook runs.
// It is optional so integrations without success side effects need no cleanup.
type PostBuildCancellationHook interface {
	OnBuildCancelled(ctx context.Context, project *models.Project, deployment *models.Deployment) error
}

type noopPostBuildHook struct{}

var errReadinessHook = errors.New("readiness hook failed")

func (n *noopPostBuildHook) OnBuildSuccess(_ context.Context, _ *models.Project, _ *models.Deployment) error {
	return nil
}
func (n *noopPostBuildHook) OnBuildFailure(_ context.Context, _ *models.Project, _ *models.Deployment, _ error) error {
	return nil
}

type InstallationTokenProvider interface {
	GetInstallationToken(installationID int64) (string, error)
}

// LifecycleReporter publishes deployment state to an external integration.
// Reporting is best-effort and never changes the build's primary outcome.
type LifecycleReporter interface {
	Report(context.Context, *models.Project, *models.Deployment) error
}

// BuildExecutor runs the 6-step build pipeline for a single deployment.
type BuildExecutor struct {
	cfg            *config.BuildConfig
	encryptionKey  string
	docker         dockerpkg.DockerClient
	deploymentRepo *repository.DeploymentRepository
	projectRepo    *repository.ProjectRepository
	envVarRepo     *repository.EnvVarRepository
	sseHub         *SSEHub
	readinessHook  PostBuildHook
	postBuild      PostBuildHook
	reporter       LifecycleReporter
	platformDomain string
	tokenProvider  InstallationTokenProvider

	mu        sync.Mutex
	cancelFns map[string]context.CancelFunc
}

// NewBuildExecutor creates an executor with all required dependencies.
func NewBuildExecutor(
	cfg *config.BuildConfig,
	encryptionKey string,
	docker dockerpkg.DockerClient,
	deploymentRepo *repository.DeploymentRepository,
	projectRepo *repository.ProjectRepository,
	envVarRepo *repository.EnvVarRepository,
	sseHub *SSEHub,
	platformDomain string,
	tokenProvider InstallationTokenProvider,
) *BuildExecutor {
	return &BuildExecutor{
		cfg:            cfg,
		encryptionKey:  encryptionKey,
		docker:         docker,
		deploymentRepo: deploymentRepo,
		projectRepo:    projectRepo,
		envVarRepo:     envVarRepo,
		sseHub:         sseHub,
		readinessHook:  &noopPostBuildHook{},
		postBuild:      &noopPostBuildHook{},
		platformDomain: platformDomain,
		tokenProvider:  tokenProvider,
		cancelFns:      make(map[string]context.CancelFunc),
	}
}

// SetPostBuildHook allows Phase 4/5 to register callbacks.
func (e *BuildExecutor) SetPostBuildHook(hook PostBuildHook) {
	e.postBuild = hook
}

// SetReadinessHook installs the route activation hook that must succeed before
// the deployment can transition to ready.
func (e *BuildExecutor) SetReadinessHook(hook PostBuildHook) {
	e.readinessHook = hook
}

func (e *BuildExecutor) SetLifecycleReporter(reporter LifecycleReporter) {
	e.reporter = reporter
}

// Execute runs the full build pipeline for a deployment.
func (e *BuildExecutor) Execute(parentCtx context.Context, deploymentID string) {
	timeout := time.Duration(e.cfg.BuildTimeoutMinutes) * time.Minute
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	e.mu.Lock()
	e.cancelFns[deploymentID] = cancel
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.cancelFns, deploymentID)
		e.mu.Unlock()
	}()

	deployment, err := e.deploymentRepo.GetByID(ctx, deploymentID)
	if err != nil {
		slog.Error("executor: deployment not found", "id", deploymentID, "err", err)
		return
	}
	project, err := e.projectRepo.GetByID(ctx, deployment.ProjectID)
	if err != nil {
		slog.Error("executor: project not found", "id", deployment.ProjectID, "err", err)
		e.failDeployment(ctx, deployment, "Internal error: failed to load project for build")
		return
	}

	logPath, err := sanitize.SafeJoinPath(e.cfg.LogBaseDir, deploymentID+".log")
	if err != nil {
		slog.Error("executor: unsafe log path", "id", deploymentID, "err", err)
		e.failDeployment(context.Background(), deployment, "Internal error: invalid build log path")
		return
	}
	logger, err := NewBuildLogger(logPath, e.sseHub, deploymentID, e.cfg.MaxLogFileSizeBytes)
	if err != nil {
		slog.Error("executor: failed to create logger", "err", err)
		e.failDeployment(ctx, deployment, "Internal error: failed to create build logger")
		return
	}
	defer logger.Close()

	// Update status to building
	now := time.Now().UTC()
	deployment.Status = models.DeploymentStatusBuilding
	deployment.StartedAt = &now
	deployment.LogPath = &logPath
	updated, err := e.deploymentRepo.UpdateIfStatus(ctx, deployment, models.DeploymentStatusQueued)
	if err != nil {
		logger.Errorf("Failed to update status: %v", err)
		return
	}
	if !updated {
		logger.Info("Build skipped because deployment is no longer queued")
		e.publishCancelledIfCurrent(context.Background(), deployment.ID)
		return
	}
	e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building"})
	e.reportLifecycle(ctx, project, deployment, logger)

	startTime := time.Now()

	// === STEP 1: Clone Repository ===
	e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "clone"})
	logger.Info("▶ Cloning repository...")
	cloneDir, err := e.stepClone(ctx, project, deployment, logger)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Clone failed: "+err.Error())
		return
	}
	defer os.RemoveAll(cloneDir)

	// Resolve source directory (monorepo support)
	rootDirectory, err := sanitize.SafeRelativePath(deployment.BuildRootDirectory)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Invalid root directory: "+err.Error())
		return
	}
	sourceDir, err := sanitize.SafeJoinPath(cloneDir, rootDirectory)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Invalid root directory: "+err.Error())
		return
	}
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		e.handleFailure(ctx, deployment, project, logger,
			fmt.Sprintf("Root directory %q not found in repository", deployment.BuildRootDirectory))
		return
	}

	// === STEP 2: Detect Framework ===
	e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "install"})
	logger.Info("▶ Detecting framework...")
	fw, pkg, err := detect.DetectFramework(sourceDir)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Framework detection failed: "+err.Error())
		return
	}

	pm := detect.DetectPackageManager(sourceDir)
	packageManagerVersion := detect.PackageManagerVersion(pkg, pm.Name)
	lockHash := detect.HashLockFile(sourceDir, pm.LockFile)
	installCmd := ""
	buildCommand := ""
	nodeVersion := ""

	if deployment.BuildManifestResolved {
		if deployment.BuildFramework == nil || deployment.BuildServingMode == nil ||
			deployment.BuildPackageManager == nil || deployment.BuildOutputDirectory == nil {
			e.handleFailure(ctx, deployment, project, logger, "Resolved build manifest is incomplete")
			return
		}
		if pm.Name != *deployment.BuildPackageManager || lockHash != deployment.BuildLockFileHash {
			e.handleFailure(ctx, deployment, project, logger,
				"Build manifest does not match the checked-out commit's package manager or lockfile")
			return
		}
		if deployment.BuildPackageManagerVersion != nil &&
			*deployment.BuildPackageManagerVersion != packageManagerVersion {
			e.handleFailure(ctx, deployment, project, logger,
				"Build manifest package manager version does not match the checked-out commit")
			return
		}
		fw.Name = *deployment.BuildFramework
		fw.DisplayName = *deployment.BuildFramework
		fw.ServingMode = *deployment.BuildServingMode
		fw.OutputDirectory = *deployment.BuildOutputDirectory
		if deployment.BuildCommand != nil {
			buildCommand = *deployment.BuildCommand
		}
		if deployment.BuildInstallCommand != nil {
			installCmd = *deployment.BuildInstallCommand
		}
		nodeVersion = deployment.BuildNodeVersion
	} else {
		buildCmdOverride := pointerValue(deployment.BuildCommand)
		outputDir := pointerValue(deployment.BuildOutputDirectory)
		fw = detect.ApplyOverrides(fw, buildCmdOverride, "", outputDir)
		outputDirectory, err := sanitize.SafeRelativePath(fw.OutputDirectory)
		if err != nil {
			e.handleFailure(ctx, deployment, project, logger, "Invalid output directory: "+err.Error())
			return
		}
		fw.OutputDirectory = outputDirectory
		installCmd = pm.InstallCommand
		if deployment.BuildInstallCommand != nil && *deployment.BuildInstallCommand != "" {
			installCmd = *deployment.BuildInstallCommand
		}
		buildCommand = fw.BuildCommand
		if buildCmdOverride == "" {
			buildCommand = detect.AdaptCommandForPackageManager(buildCommand, pm.Name)
		}
		nodeVersion = detect.DetectNodeVersion(pkg, e.cfg.DefaultNodeVersion)
		if deployment.BuildNodeVersion != "" {
			nodeVersion = deployment.BuildNodeVersion
		}

		deployment.BuildFramework = stringPtr(fw.Name)
		deployment.BuildServingMode = stringPtr(fw.ServingMode)
		deployment.BuildPackageManager = stringPtr(pm.Name)
		deployment.BuildPackageManagerVersion = optionalStringPtr(packageManagerVersion)
		deployment.BuildNodeVersion = nodeVersion
		deployment.BuildRootDirectory = rootDirectory
		deployment.BuildOutputDirectory = stringPtr(fw.OutputDirectory)
		deployment.BuildInstallCommand = optionalStringPtr(installCmd)
		deployment.BuildCommand = optionalStringPtr(buildCommand)
		deployment.BuildLockFileHash = lockHash
		resolved, err := e.deploymentRepo.ResolveBuildManifest(ctx, deployment)
		if err != nil || !resolved {
			e.handleFailure(ctx, deployment, project, logger, "Failed to persist resolved build manifest")
			return
		}
	}
	outputDirectory, err := sanitize.SafeRelativePath(fw.OutputDirectory)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Invalid output directory: "+err.Error())
		return
	}
	fw.OutputDirectory = outputDirectory
	if strings.TrimSpace(nodeVersion) == "" {
		e.handleFailure(ctx, deployment, project, logger, "Resolved build manifest has no Node.js version")
		return
	}
	memoryMB := effectiveBuildMemoryMB(e.cfg.DefaultMemoryMB, sourceDir, pkg)

	logger.Infof("  Framework: %s", fw.DisplayName)
	logger.Infof("  Node.js: %s", nodeVersion)
	logger.Infof("  Package manager: %s", pm.Name)
	logger.Infof("  Build command: %s", buildCommand)
	logger.Infof("  Output directory: %s", fw.OutputDirectory)
	logger.Infof("  Build memory: %d MB", memoryMB)

	if ctx.Err() != nil {
		e.handleFailure(ctx, deployment, project, logger, "Build cancelled")
		return
	}

	// === Cache invalidation check ===
	cacheInvalidated := e.shouldInvalidateCache(project, nodeVersion, pm.Name, lockHash)
	if cacheInvalidated {
		logger.Info("  ⚠ Cache invalidated (dependency changes detected)")
		e.invalidateCache(ctx, project.ID)
	}

	if err := e.projectRepo.UpdateBuildMeta(ctx, project.ID, fw.Name, pm.Name, lockHash); err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Failed to persist detected build metadata: "+err.Error())
		return
	}
	project.Framework = &fw.Name

	// === STEP 3: Create Docker Container ===
	logger.Info("▶ Creating build container...")
	deployOutputDir, err := sanitize.SafeJoinPath(e.cfg.DeploymentBaseDir, project.ID, deploymentID)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Invalid deployment output path: "+err.Error())
		return
	}
	if err := os.MkdirAll(deployOutputDir, 0755); err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Failed to create output directory: "+err.Error())
		return
	}
	artifactCommitted := false
	defer func() {
		if !artifactCommitted {
			_ = os.RemoveAll(deployOutputDir)
		}
	}()

	envVars := e.resolveEnvVars(ctx, project, deployment)

	containerID, err := e.docker.CreateBuildContainer(ctx, dockerpkg.BuildContainerOpts{
		DeploymentID: deploymentID,
		NodeVersion:  nodeVersion,
		SourceDir:    sourceDir,
		OutputDir:    deployOutputDir,
		CacheVolume:  fmt.Sprintf("cache-%s-modules", project.ID),
		BuildCache:   fmt.Sprintf("cache-%s-build", project.ID),
		EnvVars:      envVars,
		MemoryBytes:  memoryMB * 1024 * 1024,
		NanoCPUs:     int64(e.cfg.DefaultCPUs * 1e9),
		PIDLimit:     e.cfg.PIDLimit,
	})
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Container creation failed: "+err.Error())
		return
	}
	defer func() {
		_ = e.docker.RemoveContainer(context.Background(), containerID)
	}()

	// === STEP 4: Execute install + build commands ===
	if fw.Name != "static" && fw.Name != "hugo" {
		if installCmd != "" {
			e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "install"})
			logger.Infof("▶ Running: %s", installCmd)
			if err := e.execInContainer(ctx, containerID, installCmd, logger); err != nil {
				e.handleFailure(ctx, deployment, project, logger, "Install failed: "+describeContainerExecError(err, memoryMB))
				return
			}
		}

		if buildCommand != "" {
			e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "build"})
			logger.Infof("▶ Running: %s", buildCommand)
			if err := e.execInContainer(ctx, containerID, buildCommand, logger); err != nil {
				e.handleFailure(ctx, deployment, project, logger, "Build failed: "+describeContainerExecError(err, memoryMB))
				return
			}
		}
	} else if fw.Name == "hugo" {
		e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "build"})
		logger.Infof("▶ Running: %s", buildCommand)
		if err := e.execInContainer(ctx, containerID, buildCommand, logger); err != nil {
			e.handleFailure(ctx, deployment, project, logger, "Build failed: "+describeContainerExecError(err, memoryMB))
			return
		}
	}

	if ctx.Err() != nil {
		e.handleFailure(ctx, deployment, project, logger, "Build cancelled")
		return
	}

	// === STEP 5: Copy build output ===
	e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "deploy"})
	logger.Info("▶ Copying build output...")
	var artifactSize int64

	if fw.Name == "static" && fw.OutputDirectory == "." {
		artifactSize, err = copyDirLimited(sourceDir, deployOutputDir, e.cfg.MaxArtifactSizeBytes)
	} else {
		containerOutputPath, err := sanitize.SafeJoinPath("/app/src", fw.OutputDirectory)
		if err != nil {
			e.handleFailure(ctx, deployment, project, logger, "Invalid output directory: "+err.Error())
			return
		}
		artifactSize, err = e.docker.CopyFromContainer(
			ctx,
			containerID,
			containerOutputPath,
			deployOutputDir,
			e.cfg.MaxArtifactSizeBytes,
		)
	}

	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, describeArtifactError(err))
		return
	}

	artifactSize, err = validateArtifactTree(deployOutputDir, e.cfg.MaxArtifactSizeBytes)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, describeArtifactError(err))
		return
	}
	if artifactSize == 0 {
		e.handleFailure(ctx, deployment, project, logger,
			fmt.Sprintf("Artifact output empty: build output directory %q has no files — check your build command and output directory setting", fw.OutputDirectory))
		return
	}

	if ctx.Err() != nil {
		e.handleFailure(ctx, deployment, project, logger, "Build cancelled")
		return
	}

	// === STEP 6: Finalize deployment ===
	duration := time.Since(startTime)
	deploymentURL := generateDeploymentURL(project, deployment, e.platformDomain)
	durationMs := duration.Milliseconds()

	deployment.ArtifactPath = &deployOutputDir
	sizePtr := &artifactSize
	deployment.ArtifactSizeBytes = sizePtr
	durationMsPtr := &durationMs
	deployment.BuildDurationMs = durationMsPtr
	deployment.DeploymentURL = &deploymentURL

	updated, err = e.finalizeDeploymentReady(ctx, project, deployment, logger)
	if err != nil {
		message := "Deployment finalization failed: " + err.Error()
		if errors.Is(err, errReadinessHook) {
			message = "Route activation failed: " + err.Error()
		}
		e.handleFailure(ctx, deployment, project, logger, message)
		return
	}
	if !updated {
		logger.Info("Build finalization skipped because deployment is no longer building")
		return
	}
	artifactCommitted = true

	logger.Infof("▶ Build complete (%s)", duration.Round(time.Second))
	logger.Infof("  Artifact size: %s", humanizeBytes(artifactSize))
	logger.Infof("  URL: %s", deploymentURL)
	logger.Info("✅ Deployment ready!")

	e.sseHub.PublishJSON(deploymentID, SSEEventDone, map[string]interface{}{
		"status":        "ready",
		"duration_ms":   duration.Milliseconds(),
		"url":           deploymentURL,
		"artifact_size": artifactSize,
	})

	if err := e.postBuild.OnBuildSuccess(ctx, project, deployment); err != nil {
		logger.Warn("Post-build hook error: " + err.Error())
	}
	e.cleanupSuccessfulBuildIfCancelled(project, deployment, logger)
	e.reportLifecycle(ctx, project, deployment, logger)
}

func (e *BuildExecutor) finalizeDeploymentReady(
	ctx context.Context,
	project *models.Project,
	deployment *models.Deployment,
	logger *BuildLogger,
) (bool, error) {
	if err := e.readinessHook.OnBuildSuccess(ctx, project, deployment); err != nil {
		e.cleanupReadinessSideEffects(project, deployment, logger)
		return false, fmt.Errorf("%w: %v", errReadinessHook, err)
	}

	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusReady
	deployment.CompletedAt = &completedAt
	updated, err := e.deploymentRepo.UpdateIfStatus(ctx, deployment, models.DeploymentStatusBuilding)
	if err != nil {
		e.cleanupReadinessSideEffects(project, deployment, logger)
		deployment.Status = models.DeploymentStatusBuilding
		deployment.CompletedAt = nil
		return false, err
	}
	if !updated {
		e.cleanupReadinessSideEffects(project, deployment, logger)
		e.publishCancelledIfCurrent(context.Background(), deployment.ID)
	}
	return updated, nil
}

func (e *BuildExecutor) cleanupReadinessSideEffects(project *models.Project, deployment *models.Deployment, logger *BuildLogger) {
	cleanup, ok := e.readinessHook.(PostBuildCancellationHook)
	if !ok {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cleanup.OnBuildCancelled(cleanupCtx, project, deployment); err != nil {
		logger.Warn("Deployment readiness cleanup failed: " + err.Error())
	}
}

func (e *BuildExecutor) cleanupSuccessfulBuildIfCancelled(project *models.Project, deployment *models.Deployment, logger *BuildLogger) {
	checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	current, err := e.deploymentRepo.GetByID(checkCtx, deployment.ID)
	if err != nil {
		logger.Warn("Could not verify deployment status after post-build hooks: " + err.Error())
		return
	}
	if current.Status != models.DeploymentStatusCancelled {
		return
	}

	*deployment = *current
	for _, hook := range []PostBuildHook{e.readinessHook, e.postBuild} {
		cleanup, ok := hook.(PostBuildCancellationHook)
		if !ok {
			continue
		}
		if err := cleanup.OnBuildCancelled(checkCtx, project, deployment); err != nil {
			logger.Warn("Cancelled deployment route cleanup failed: " + err.Error())
		}
	}
}

// CancelBuild cancels an in-flight build by deployment ID.
func (e *BuildExecutor) CancelBuild(deploymentID string) {
	e.mu.Lock()
	cancelFn, ok := e.cancelFns[deploymentID]
	e.mu.Unlock()

	if ok {
		cancelFn()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = e.docker.StopContainer(ctx, "build-"+deploymentID, 2*time.Second)
}

// stepClone performs Step 1: Git clone with retries.
func (e *BuildExecutor) stepClone(
	ctx context.Context,
	project *models.Project,
	deployment *models.Deployment,
	logger *BuildLogger,
) (string, error) {
	cloneDir, err := sanitize.SafeJoinPath(e.cfg.CloneBaseDir, "clone-"+deployment.ID)
	if err != nil {
		return "", fmt.Errorf("resolve clone dir: %w", err)
	}
	if err := os.RemoveAll(cloneDir); err != nil {
		return "", fmt.Errorf("clean clone dir: %w", err)
	}
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir clone dir: %w", err)
	}

	repoName := ""
	if deployment.SourceRepository != nil {
		repoName = *deployment.SourceRepository
	}
	if repoName == "" {
		return "", fmt.Errorf("project is not linked to a GitHub repository")
	}
	if err := validateGitHubRepository(repoName); err != nil {
		return "", err
	}
	cloneURL := fmt.Sprintf("https://github.com/%s.git", repoName)
	cloneToken := ""
	if deployment.SourceInstallationID != nil && e.tokenProvider != nil {
		token, err := e.tokenProvider.GetInstallationToken(*deployment.SourceInstallationID)
		if err != nil {
			return "", fmt.Errorf("get github installation token: %w", err)
		}
		cloneToken = token
	}
	gitEnv := gitAuthenticationEnv(cloneToken)

	cloneTimeout := time.Duration(e.cfg.CloneTimeoutSeconds) * time.Second
	maxRetries := e.cfg.CloneMaxRetries
	retryDelay := time.Duration(e.cfg.CloneRetryDelaySec) * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Infof("  Retry %d/%d (waiting %s)...", attempt, maxRetries, retryDelay)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
			}
			os.RemoveAll(cloneDir)
			os.MkdirAll(cloneDir, 0755)
		}

		cloneCtx, cloneCancel := context.WithTimeout(ctx, cloneTimeout)

		resolvedSHA, output, err := checkoutRepository(cloneCtx, cloneDir, cloneURL, deployment.Branch, deployment.CommitSHA, gitEnv)
		cloneCancel()

		if err == nil {
			if err := e.deploymentRepo.UpdateResolvedCommit(ctx, deployment.ID, resolvedSHA); err != nil {
				return "", fmt.Errorf("record resolved commit: %w", err)
			}
			deployment.CommitSHA = resolvedSHA
			logger.Infof("  Checked out %s@%s (%s)", repoName, deployment.Branch, resolvedSHA[:12])
			return cloneDir, nil
		}

		outputText := string(output)
		if cloneToken != "" {
			outputText = strings.ReplaceAll(outputText, cloneToken, "***")
			encodedCredential := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + cloneToken))
			outputText = strings.ReplaceAll(outputText, encodedCredential, "***")
		}
		lastErr = fmt.Errorf("git clone (attempt %d): %w\n%s", attempt, err, outputText)
		logger.Warn(fmt.Sprintf("  Clone attempt %d failed: %v", attempt, err))
	}

	return "", lastErr
}

func checkoutRepository(ctx context.Context, cloneDir, cloneURL, branch, requestedSHA string, gitEnv []string) (string, []byte, error) {
	if strings.HasPrefix(branch, "-") || strings.TrimSpace(branch) == "" {
		return "", nil, fmt.Errorf("invalid branch %q", branch)
	}
	if _, output, err := runGit(ctx, gitEnv, "check-ref-format", "refs/heads/"+branch); err != nil {
		return "", output, fmt.Errorf("invalid branch %q: %w", branch, err)
	}
	if _, output, err := runGit(ctx, gitEnv, "-C", cloneDir, "init", "--quiet"); err != nil {
		return "", output, err
	}
	if _, output, err := runGit(ctx, gitEnv, "-C", cloneDir, "remote", "add", "origin", cloneURL); err != nil {
		return "", output, err
	}

	fetchRef := "refs/heads/" + branch
	if !isUnresolvedCommit(requestedSHA) {
		if !isFullCommitSHA(requestedSHA) {
			return "", nil, fmt.Errorf("commit SHA must be a full 40-character hexadecimal value")
		}
		fetchRef = strings.ToLower(requestedSHA)
	}
	if _, output, err := runGit(ctx, gitEnv, "-C", cloneDir, "fetch", "--quiet", "--depth=1", "origin", fetchRef); err != nil {
		return "", output, err
	}
	if _, output, err := runGit(ctx, gitEnv, "-C", cloneDir, "checkout", "--quiet", "--detach", "FETCH_HEAD"); err != nil {
		return "", output, err
	}
	resolved, output, err := runGit(ctx, gitEnv, "-C", cloneDir, "rev-parse", "HEAD")
	if err != nil {
		return "", output, err
	}
	resolved = strings.TrimSpace(resolved)
	if !isFullCommitSHA(resolved) {
		return "", output, fmt.Errorf("git resolved invalid commit %q", resolved)
	}
	if !isUnresolvedCommit(requestedSHA) && !strings.EqualFold(resolved, requestedSHA) {
		return "", output, fmt.Errorf("checked out commit %s does not match requested commit %s", resolved, requestedSHA)
	}
	return strings.ToLower(resolved), output, nil
}

func runGit(ctx context.Context, extraEnv []string, args ...string) (string, []byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0"), extraEnv...)
	output, err := cmd.CombinedOutput()
	return string(output), output, err
}

func gitAuthenticationEnv(token string) []string {
	if token == "" {
		return nil
	}
	credential := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.https://github.com/.extraheader",
		"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + credential,
	}
}

func isUnresolvedCommit(sha string) bool {
	sha = strings.TrimSpace(sha)
	return sha == "" || strings.EqualFold(sha, "manual")
}

func isFullCommitSHA(sha string) bool {
	if len(sha) != 40 {
		return false
	}
	_, err := hex.DecodeString(sha)
	return err == nil
}

func validateGitHubRepository(repo string) error {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid GitHub repository %q", repo)
	}
	for _, part := range parts {
		if part == "." || part == ".." || strings.HasPrefix(part, "-") {
			return fmt.Errorf("invalid GitHub repository %q", repo)
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
				return fmt.Errorf("invalid GitHub repository %q", repo)
			}
		}
	}
	return nil
}

func (e *BuildExecutor) execInContainer(
	ctx context.Context,
	containerID string,
	cmd string,
	logger *BuildLogger,
) error {
	stdout := logger.StreamWriter(LogInfo)
	stderr := logger.StreamWriter(LogError)
	return e.docker.ExecCommand(ctx, containerID, cmd, stdout, stderr)
}

func (e *BuildExecutor) resolveEnvVars(ctx context.Context, project *models.Project, deployment *models.Deployment) []string {
	vars := baseBuildEnvVars(project, deployment)

	scope := "preview"
	if deployment.IsProduction {
		scope = "production"
	}
	projectVars, err := e.envVarRepo.GetDecryptedForBuild(ctx, project.ID, scope, e.encryptionKey)
	if err != nil {
		slog.Warn("Failed to load project env vars", "project_id", project.ID, "err", err)
	}
	for _, v := range projectVars {
		vars = append(vars, v.Key+"="+v.Value)
	}

	return vars
}

func baseBuildEnvVars(project *models.Project, deployment *models.Deployment) []string {
	vars := []string{
		"CI=true",
		"HOSTBOX=1",
		"HOSTBOX_PROJECT_ID=" + project.ID,
		"HOSTBOX_PROJECT_NAME=" + project.Name,
		"HOSTBOX_DEPLOYMENT_ID=" + deployment.ID,
		"HOSTBOX_BRANCH=" + deployment.Branch,
		"HOSTBOX_COMMIT_SHA=" + deployment.CommitSHA,
	}

	if deployment.IsProduction {
		return append(vars, "HOSTBOX_IS_PREVIEW=false")
	}

	return append(vars, "HOSTBOX_IS_PREVIEW=true")
}

func (e *BuildExecutor) shouldInvalidateCache(project *models.Project, nodeVersion, pkgManager, lockHash string) bool {
	if project.NodeVersion != nodeVersion && project.NodeVersion != "" {
		return true
	}
	if project.DetectedPackageManager != pkgManager && project.DetectedPackageManager != "" {
		return true
	}
	if project.LockFileHash != lockHash && project.LockFileHash != "" {
		return true
	}
	return false
}

func effectiveBuildMemoryMB(defaultMemoryMB int64, sourceDir string, pkg *detect.PackageJSON) int64 {
	if defaultMemoryMB >= 1024 {
		return defaultMemoryMB
	}
	if detect.IsWorkspaceProject(sourceDir, pkg) {
		return 1024
	}
	return defaultMemoryMB
}

func describeContainerExecError(err error, memoryMB int64) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	if strings.Contains(msg, "command exited with code 137") {
		return fmt.Sprintf("%s — build container was killed, likely due to memory pressure; increase BUILD_MEMORY_MB above %d", msg, memoryMB)
	}
	return msg
}

func describeArtifactError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "exceeds maximum size"):
		return "Artifact output oversized: " + message
	case strings.Contains(message, "symbolic link"),
		strings.Contains(message, "non-regular"),
		strings.Contains(message, "unsupported tar entry"):
		return "Artifact output unsafe: " + message
	default:
		return "Artifact output missing or unreadable: " + message
	}
}

func (e *BuildExecutor) invalidateCache(ctx context.Context, projectID string) {
	_ = e.docker.RemoveVolume(ctx, fmt.Sprintf("cache-%s-modules", projectID))
	_ = e.docker.RemoveVolume(ctx, fmt.Sprintf("cache-%s-build", projectID))
}

func (e *BuildExecutor) handleFailure(
	ctx context.Context,
	deployment *models.Deployment,
	project *models.Project,
	logger *BuildLogger,
	errMsg string,
) {
	if errors.Is(ctx.Err(), context.Canceled) {
		e.cancelDeployment(deployment, project, logger)
		return
	}
	logger.Errorf("❌ %s", errMsg)

	updateCtx := ctx
	var cancel context.CancelFunc
	if ctx.Err() != nil {
		updateCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusFailed
	deployment.ErrorMessage = &errMsg
	deployment.CompletedAt = &completedAt
	updated, updateErr := e.deploymentRepo.UpdateIfStatus(updateCtx, deployment, models.DeploymentStatusBuilding)
	if updateErr != nil {
		logger.Errorf("Failed to persist failed status: %v", updateErr)
		return
	}
	if !updated {
		logger.Info("Failure ignored because deployment is already terminal")
		e.publishCancelledIfCurrent(updateCtx, deployment.ID)
		return
	}

	e.sseHub.PublishJSON(deployment.ID, SSEEventDone, map[string]interface{}{
		"status":  "failed",
		"message": errMsg,
	})

	_ = e.postBuild.OnBuildFailure(updateCtx, project, deployment, fmt.Errorf("%s", errMsg))
	e.reportLifecycle(updateCtx, project, deployment, logger)
}

func (e *BuildExecutor) cancelDeployment(deployment *models.Deployment, project *models.Project, logger *BuildLogger) {
	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusCancelled
	deployment.ErrorMessage = nil
	deployment.CompletedAt = &completedAt
	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updated, err := e.deploymentRepo.UpdateIfStatus(updateCtx, deployment, models.DeploymentStatusBuilding)
	if err != nil {
		logger.Errorf("Failed to persist cancelled status: %v", err)
	}
	if !updated && err == nil {
		logger.Info("Deployment was already terminal when cancellation completed")
	}
	logger.Info("Build cancelled")
	e.sseHub.PublishJSON(deployment.ID, SSEEventDone, map[string]interface{}{"status": "cancelled"})
	if updated {
		e.reportLifecycle(updateCtx, project, deployment, logger)
	}
}

func (e *BuildExecutor) reportLifecycle(ctx context.Context, project *models.Project, deployment *models.Deployment, logger *BuildLogger) {
	if e.reporter == nil {
		return
	}
	reloadCtx := ctx
	var cancel context.CancelFunc
	if ctx.Err() != nil {
		reloadCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	current, err := e.deploymentRepo.GetByID(reloadCtx, deployment.ID)
	if err != nil {
		logger.Warn("Could not reload deployment for GitHub lifecycle feedback: " + err.Error())
		return
	}
	if err := e.reporter.Report(reloadCtx, project, current); err != nil {
		logger.Warn("GitHub lifecycle feedback failed: " + err.Error())
	}
	*deployment = *current
}

func (e *BuildExecutor) publishCancelledIfCurrent(ctx context.Context, deploymentID string) {
	current, err := e.deploymentRepo.GetByID(ctx, deploymentID)
	if err == nil && current.Status == models.DeploymentStatusCancelled {
		e.sseHub.PublishJSON(deploymentID, SSEEventDone, map[string]interface{}{"status": "cancelled"})
	}
}

func (e *BuildExecutor) failDeployment(ctx context.Context, deployment *models.Deployment, errMsg string) {
	updateCtx := ctx
	var cancel context.CancelFunc
	if ctx.Err() != nil {
		updateCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusFailed
	deployment.ErrorMessage = &errMsg
	deployment.CompletedAt = &completedAt
	updated, _ := e.deploymentRepo.UpdateIfStatus(updateCtx, deployment, models.DeploymentStatusQueued)
	if !updated {
		updated, _ = e.deploymentRepo.UpdateIfStatus(updateCtx, deployment, models.DeploymentStatusBuilding)
	}
	if !updated {
		return
	}
	e.sseHub.PublishJSON(deployment.ID, SSEEventDone, map[string]interface{}{
		"status":  "failed",
		"message": errMsg,
	})
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtr(value string) *string {
	return &value
}

func optionalStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return stringPtr(value)
}
