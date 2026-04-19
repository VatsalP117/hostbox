package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/platform/detect"
	dockerpkg "github.com/VatsalP117/hostbox/internal/platform/docker"
	"github.com/VatsalP117/hostbox/internal/repository"
)

// PostBuildHook is called after a build completes.
type PostBuildHook interface {
	OnBuildSuccess(ctx context.Context, project *models.Project, deployment *models.Deployment) error
	OnBuildFailure(ctx context.Context, project *models.Project, deployment *models.Deployment, buildErr error) error
}

type noopPostBuildHook struct{}

func (n *noopPostBuildHook) OnBuildSuccess(_ context.Context, _ *models.Project, _ *models.Deployment) error {
	return nil
}
func (n *noopPostBuildHook) OnBuildFailure(_ context.Context, _ *models.Project, _ *models.Deployment, _ error) error {
	return nil
}

type InstallationTokenProvider interface {
	GetInstallationToken(installationID int64) (string, error)
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
	postBuild      PostBuildHook
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
		return
	}

	logPath := filepath.Join(e.cfg.LogBaseDir, deploymentID+".log")
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
	if err := e.deploymentRepo.Update(ctx, deployment); err != nil {
		logger.Errorf("Failed to update status: %v", err)
		return
	}
	e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building"})

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
	sourceDir := cloneDir
	if project.RootDirectory != "" && project.RootDirectory != "/" {
		sourceDir = filepath.Join(cloneDir, strings.TrimPrefix(project.RootDirectory, "/"))
		if _, err := os.Stat(sourceDir); err != nil {
			e.handleFailure(ctx, deployment, project, logger,
				fmt.Sprintf("Root directory %q not found in repository", project.RootDirectory))
			return
		}
	}

	// === STEP 2: Detect Framework ===
	e.sseHub.PublishJSON(deploymentID, SSEEventStatus, map[string]string{"status": "building", "phase": "install"})
	logger.Info("▶ Detecting framework...")
	fw, pkg, err := detect.DetectFramework(sourceDir)
	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Framework detection failed: "+err.Error())
		return
	}

	// Apply project-level overrides
	buildCmdOverride := ""
	if project.BuildCommand != nil {
		buildCmdOverride = *project.BuildCommand
	}
	outputDir := ""
	if project.OutputDirectory != nil {
		outputDir = *project.OutputDirectory
	}
	fw = detect.ApplyOverrides(fw, buildCmdOverride, "", outputDir)

	pm := detect.DetectPackageManager(sourceDir)
	installCmd := pm.InstallCommand
	if project.InstallCommand != nil && *project.InstallCommand != "" {
		installCmd = *project.InstallCommand
	}
	buildCommand := fw.BuildCommand
	if buildCmdOverride == "" {
		buildCommand = detect.AdaptCommandForPackageManager(buildCommand, pm.Name)
	}

	nodeVersion := detect.DetectNodeVersion(pkg, e.cfg.DefaultNodeVersion)
	if project.NodeVersion != "" {
		nodeVersion = project.NodeVersion
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
	lockHash := detect.HashLockFile(sourceDir, pm.LockFile)
	cacheInvalidated := e.shouldInvalidateCache(project, nodeVersion, pm.Name, lockHash)
	if cacheInvalidated {
		logger.Info("  ⚠ Cache invalidated (dependency changes detected)")
		e.invalidateCache(ctx, project.ID)
	}

	_ = e.projectRepo.UpdateBuildMeta(ctx, project.ID, pm.Name, lockHash)

	// === STEP 3: Create Docker Container ===
	logger.Info("▶ Creating build container...")
	deployOutputDir := filepath.Join(e.cfg.DeploymentBaseDir, project.ID, deploymentID)
	if err := os.MkdirAll(deployOutputDir, 0755); err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Failed to create output directory: "+err.Error())
		return
	}

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
		artifactSize, err = copyDir(sourceDir, deployOutputDir)
	} else {
		containerOutputPath := filepath.Join("/app/src", fw.OutputDirectory)
		artifactSize, err = e.docker.CopyFromContainer(ctx, containerID, containerOutputPath, deployOutputDir)
	}

	if err != nil {
		e.handleFailure(ctx, deployment, project, logger, "Output copy failed: "+err.Error())
		return
	}

	if isEmpty, _ := isDirEmpty(deployOutputDir); isEmpty {
		e.handleFailure(ctx, deployment, project, logger,
			fmt.Sprintf("Build output directory %q is empty — check your build command and output directory setting", fw.OutputDirectory))
		return
	}

	// === STEP 6: Finalize deployment ===
	duration := time.Since(startTime)
	deploymentURL := generateDeploymentURL(project, deployment, e.platformDomain)
	durationMs := duration.Milliseconds()

	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusReady
	deployment.ArtifactPath = &deployOutputDir
	sizePtr := &artifactSize
	deployment.ArtifactSizeBytes = sizePtr
	durationMsPtr := &durationMs
	deployment.BuildDurationMs = durationMsPtr
	deployment.DeploymentURL = &deploymentURL
	deployment.CompletedAt = &completedAt

	if err := e.deploymentRepo.Update(ctx, deployment); err != nil {
		logger.Errorf("Failed to update deployment record: %v", err)
	}

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
	cloneDir := filepath.Join(e.cfg.CloneBaseDir, "clone-"+deployment.ID)
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir clone dir: %w", err)
	}

	repoName := ""
	if project.GitHubRepo != nil {
		repoName = *project.GitHubRepo
	}
	if repoName == "" {
		return "", fmt.Errorf("project is not linked to a GitHub repository")
	}
	cloneURL := fmt.Sprintf("https://github.com/%s.git", repoName)
	cloneToken := ""
	if project.GitHubInstallationID != nil && e.tokenProvider != nil {
		token, err := e.tokenProvider.GetInstallationToken(*project.GitHubInstallationID)
		if err != nil {
			return "", fmt.Errorf("get github installation token: %w", err)
		}
		cloneToken = token
		cloneURL = fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, repoName)
	}

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

		args := []string{
			"clone",
			"--depth=1",
			"--single-branch",
			"--branch", deployment.Branch,
			cloneURL,
			cloneDir,
		}

		cmd := exec.CommandContext(cloneCtx, "git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		output, err := cmd.CombinedOutput()
		cloneCancel()

		if err == nil {
			logger.Infof("  Cloned %s@%s", repoName, deployment.Branch)
			return cloneDir, nil
		}

		outputText := string(output)
		if cloneToken != "" {
			outputText = strings.ReplaceAll(outputText, cloneToken, "***")
		}
		lastErr = fmt.Errorf("git clone (attempt %d): %w\n%s", attempt, err, outputText)
		logger.Warn(fmt.Sprintf("  Clone attempt %d failed: %v", attempt, err))
	}

	return "", lastErr
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
	vars := []string{
		"CI=true",
		"HOSTBOX=1",
		"NODE_ENV=production",
		"HOSTBOX_PROJECT_ID=" + project.ID,
		"HOSTBOX_PROJECT_NAME=" + project.Name,
		"HOSTBOX_DEPLOYMENT_ID=" + deployment.ID,
		"HOSTBOX_BRANCH=" + deployment.Branch,
		"HOSTBOX_COMMIT_SHA=" + deployment.CommitSHA,
	}

	if deployment.IsProduction {
		vars = append(vars, "HOSTBOX_IS_PREVIEW=false")
	} else {
		vars = append(vars, "HOSTBOX_IS_PREVIEW=true")
	}

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
	logger.Errorf("❌ %s", errMsg)

	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusFailed
	deployment.ErrorMessage = &errMsg
	deployment.CompletedAt = &completedAt
	_ = e.deploymentRepo.Update(ctx, deployment)

	e.sseHub.PublishJSON(deployment.ID, SSEEventDone, map[string]interface{}{
		"status":  "failed",
		"message": errMsg,
	})

	_ = e.postBuild.OnBuildFailure(ctx, project, deployment, fmt.Errorf("%s", errMsg))
}

func (e *BuildExecutor) failDeployment(ctx context.Context, deployment *models.Deployment, errMsg string) {
	completedAt := time.Now().UTC()
	deployment.Status = models.DeploymentStatusFailed
	deployment.ErrorMessage = &errMsg
	deployment.CompletedAt = &completedAt
	_ = e.deploymentRepo.Update(ctx, deployment)
}
