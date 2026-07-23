package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/platform/detect"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/migrations"
)

func TestEffectiveBuildMemoryMB_BumpsWorkspaceDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkg := &detect.PackageJSON{Workspaces: json.RawMessage(`["apps/*"]`)}

	got := effectiveBuildMemoryMB(512, dir, pkg)
	if got != 1024 {
		t.Fatalf("expected workspace build memory to be bumped to 1024, got %d", got)
	}
}

func TestCheckoutRepositoryChecksOutRequestedCommit(t *testing.T) {
	remote, firstSHA, secondSHA := createGitRemote(t)
	cloneDir := filepath.Join(t.TempDir(), "checkout")
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		t.Fatal(err)
	}

	resolved, _, err := checkoutRepository(context.Background(), cloneDir, remote, "main", firstSHA, nil)
	if err != nil {
		t.Fatalf("checkoutRepository: %v", err)
	}
	if resolved != firstSHA || resolved == secondSHA {
		t.Fatalf("resolved %q, want exact requested commit %q (branch head %q)", resolved, firstSHA, secondSHA)
	}
}

func TestCheckoutRepositoryResolvesManualDeploymentToBranchHead(t *testing.T) {
	remote, _, headSHA := createGitRemote(t)
	cloneDir := filepath.Join(t.TempDir(), "checkout")
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		t.Fatal(err)
	}

	resolved, _, err := checkoutRepository(context.Background(), cloneDir, remote, "main", "manual", nil)
	if err != nil {
		t.Fatalf("checkoutRepository: %v", err)
	}
	if resolved != headSHA {
		t.Fatalf("resolved %q, want branch head %q", resolved, headSHA)
	}
}

func TestIsUnresolvedCommitAcceptsManualAndEmptySentinels(t *testing.T) {
	for _, value := range []string{"", "   ", "manual", "MANUAL"} {
		if !isUnresolvedCommit(value) {
			t.Fatalf("expected %q to request branch-head resolution", value)
		}
	}
	if isUnresolvedCommit(strings.Repeat("a", 40)) {
		t.Fatal("full SHA must not be treated as unresolved")
	}
}

func TestCheckoutRepositoryRejectsInvalidCommit(t *testing.T) {
	remote, _, _ := createGitRemote(t)
	cloneDir := filepath.Join(t.TempDir(), "checkout")
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, _, err := checkoutRepository(context.Background(), cloneDir, remote, "main", "abc123", nil)
	if err == nil || !strings.Contains(err.Error(), "full 40-character") {
		t.Fatalf("expected full SHA validation error, got %v", err)
	}
}

func TestGitAuthenticationEnvDoesNotExposeRawToken(t *testing.T) {
	token := "installation-secret-token"
	env := gitAuthenticationEnv(token)
	for _, value := range env {
		if strings.Contains(value, token) {
			t.Fatalf("raw token exposed in git environment entry %q", value)
		}
	}
	if len(env) != 3 || env[0] != "GIT_CONFIG_COUNT=1" {
		t.Fatalf("unexpected authentication environment: %v", env)
	}
}

func TestHandleFailureKeepsCancellationTerminalAndSkipsFailureHook(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)
	user := &models.User{Email: "cancel@hostbox.local", PasswordHash: "hash"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	project := &models.Project{OwnerID: user.ID, Name: "Cancel", Slug: "cancel", ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40), Branch: "main", Status: models.DeploymentStatusBuilding}
	if err := deploymentRepo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}

	hub := NewSSEHub()
	events, unsubscribe := hub.Subscribe(deployment.ID)
	defer unsubscribe()
	hook := &recordingPostBuildHook{}
	executor := &BuildExecutor{deploymentRepo: deploymentRepo, sseHub: hub, postBuild: hook}
	logger, err := NewBuildLogger(filepath.Join(t.TempDir(), "cancel.log"), hub, deployment.ID, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	executor.handleFailure(cancelledCtx, deployment, project, logger, "Install failed: context canceled")

	got, err := deploymentRepo.GetByID(context.Background(), deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.DeploymentStatusCancelled {
		t.Fatalf("status = %q, want cancelled", got.Status)
	}
	if hook.failures != 0 {
		t.Fatalf("failure hook called %d times, want 0", hook.failures)
	}
	deadline := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.Type == SSEEventDone {
				if !strings.Contains(event.Data, `"status":"cancelled"`) {
					t.Fatalf("unexpected done event: %s", event.Data)
				}
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for cancelled event")
		}
	}
}

func TestCleanupSuccessfulBuildIfCancelledRunsOptionalCleanupAndRefreshesDeployment(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)
	user := &models.User{Email: "cleanup@hostbox.local", PasswordHash: "hash"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	project := &models.Project{OwnerID: user.ID, Name: "Cleanup", Slug: "cleanup", ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40), Branch: "feature/test", Status: models.DeploymentStatusBuilding}
	if err := deploymentRepo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	deployment.Status = models.DeploymentStatusCancelled
	updated, err := deploymentRepo.UpdateIfStatus(ctx, deployment, models.DeploymentStatusBuilding)
	if err != nil || !updated {
		t.Fatal(err)
	}

	hook := &recordingCancellationHook{}
	executor := &BuildExecutor{deploymentRepo: deploymentRepo, readinessHook: &noopPostBuildHook{}, postBuild: hook}
	stale := *deployment
	stale.Status = models.DeploymentStatusReady
	hub := NewSSEHub()
	logger, err := NewBuildLogger(filepath.Join(t.TempDir(), "cleanup.log"), hub, deployment.ID, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	executor.cleanupSuccessfulBuildIfCancelled(project, &stale, logger)

	if hook.cancelled != 1 {
		t.Fatalf("cleanup calls = %d, want 1", hook.cancelled)
	}
	if stale.Status != models.DeploymentStatusCancelled {
		t.Fatalf("deployment status = %q, want cancelled", stale.Status)
	}
}

func TestFinalizeDeploymentReadyCleansRoutesWhenReadinessCancelsBuild(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)
	user := &models.User{Email: "readiness-race@hostbox.local", PasswordHash: "hash"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	project := &models.Project{
		OwnerID: user.ID, Name: "Readiness Race", Slug: "readiness-race",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{
		ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40), Branch: "feature/race",
		Status: models.DeploymentStatusBuilding,
	}
	if err := deploymentRepo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	hook := &cancellingReadinessHook{repo: deploymentRepo}
	executor := &BuildExecutor{
		deploymentRepo: deploymentRepo,
		readinessHook:  hook,
		postBuild:      &noopPostBuildHook{},
		sseHub:         NewSSEHub(),
	}
	logger, err := NewBuildLogger(filepath.Join(t.TempDir(), "readiness.log"), executor.sseHub, deployment.ID, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	updated, err := executor.finalizeDeploymentReady(ctx, project, deployment, logger)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("ready compare-and-set should lose to cancellation")
	}
	if hook.cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", hook.cleanupCalls)
	}
	stored, err := deploymentRepo.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.DeploymentStatusCancelled {
		t.Fatalf("stored status = %q, want cancelled", stored.Status)
	}
}

func TestFinalizeDeploymentReadyCleansPartialRoutesWhenReadinessFails(t *testing.T) {
	hook := &failingReadinessHook{}
	executor := &BuildExecutor{
		readinessHook: hook,
		sseHub:        NewSSEHub(),
	}
	deployment := &models.Deployment{ID: "deployment", Status: models.DeploymentStatusBuilding}
	logger, err := NewBuildLogger(filepath.Join(t.TempDir(), "readiness-error.log"), executor.sseHub, deployment.ID, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	updated, err := executor.finalizeDeploymentReady(
		context.Background(),
		&models.Project{ID: "project"},
		deployment,
		logger,
	)
	if err == nil {
		t.Fatal("expected readiness hook failure")
	}
	if updated {
		t.Fatal("failed readiness must not persist ready")
	}
	if hook.cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", hook.cleanupCalls)
	}
	if deployment.Status != models.DeploymentStatusBuilding {
		t.Fatalf("in-memory status = %q, want building", deployment.Status)
	}
}

func TestFinalizeDeploymentReadyRestoresBuildingAfterPersistenceError(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)
	user := &models.User{Email: "ready-persistence@hostbox.local", PasswordHash: "hash"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	project := &models.Project{
		OwnerID: user.ID, Name: "Ready Persistence", Slug: "ready-persistence",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{
		ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40), Branch: "main",
		Status: models.DeploymentStatusBuilding,
	}
	if err := deploymentRepo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TRIGGER fail_ready_persistence
		BEFORE UPDATE OF status ON deployments
		WHEN NEW.status = 'ready'
		BEGIN
			SELECT RAISE(FAIL, 'ready persistence failed');
		END`); err != nil {
		t.Fatal(err)
	}

	hook := &recordingCancellationHook{}
	executor := &BuildExecutor{
		deploymentRepo: deploymentRepo,
		readinessHook:  hook,
		postBuild:      &noopPostBuildHook{},
		sseHub:         NewSSEHub(),
	}
	logger, err := NewBuildLogger(filepath.Join(t.TempDir(), "ready-persistence.log"), executor.sseHub, deployment.ID, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	updated, err := executor.finalizeDeploymentReady(
		ctx,
		project,
		deployment,
		logger,
	)
	if err == nil {
		t.Fatal("expected ready persistence failure")
	}
	if updated {
		t.Fatal("failed persistence must not report ready")
	}
	if deployment.Status != models.DeploymentStatusBuilding {
		t.Fatalf("in-memory status = %q, want building", deployment.Status)
	}
	if deployment.CompletedAt != nil {
		t.Fatal("completion time should be cleared after persistence failure")
	}
	if hook.cancelled != 1 {
		t.Fatalf("cleanup calls = %d, want 1", hook.cancelled)
	}
	executor.handleFailure(ctx, deployment, project, logger, "Deployment finalization failed")
	stored, err := deploymentRepo.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.DeploymentStatusFailed {
		t.Fatalf("stored status = %q, want failed", stored.Status)
	}
}

type cancellingReadinessHook struct {
	repo         *repository.DeploymentRepository
	cleanupCalls int
}

type failingReadinessHook struct {
	cleanupCalls int
}

func (h *failingReadinessHook) OnBuildSuccess(context.Context, *models.Project, *models.Deployment) error {
	return errors.New("partial route activation failed")
}

func (h *failingReadinessHook) OnBuildFailure(context.Context, *models.Project, *models.Deployment, error) error {
	return nil
}

func (h *failingReadinessHook) OnBuildCancelled(context.Context, *models.Project, *models.Deployment) error {
	h.cleanupCalls++
	return nil
}

func (h *cancellingReadinessHook) OnBuildSuccess(ctx context.Context, _ *models.Project, deployment *models.Deployment) error {
	cancelled := *deployment
	cancelled.Status = models.DeploymentStatusCancelled
	updated, err := h.repo.UpdateIfStatus(ctx, &cancelled, models.DeploymentStatusBuilding)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("could not cancel deployment")
	}
	return nil
}

func (h *cancellingReadinessHook) OnBuildFailure(context.Context, *models.Project, *models.Deployment, error) error {
	return nil
}

func (h *cancellingReadinessHook) OnBuildCancelled(context.Context, *models.Project, *models.Deployment) error {
	h.cleanupCalls++
	return nil
}

type recordingWorkerLifecycleReporter struct {
	status   models.DeploymentStatus
	prNumber *int
}

func (r *recordingWorkerLifecycleReporter) Report(_ context.Context, _ *models.Project, deployment *models.Deployment) error {
	r.status = deployment.Status
	r.prNumber = deployment.GitHubPRNumber
	return nil
}

func TestReportLifecycleReloadsPRAssociationAndTerminalState(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	user := &models.User{Email: "feedback@hostbox.local", PasswordHash: "hash"}
	if err := repository.NewUserRepository(db).Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	project := &models.Project{OwnerID: user.ID, Name: "Feedback", Slug: "feedback", ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20"}
	if err := repository.NewProjectRepository(db).Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40), Branch: "feature/test", Status: models.DeploymentStatusBuilding}
	if err := deploymentRepo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	stale := *deployment
	if err := deploymentRepo.UpdateStatus(ctx, deployment.ID, models.DeploymentStatusReady, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := deploymentRepo.SetGitHubPRNumberIfUnset(ctx, deployment.ID, 42); err != nil {
		t.Fatal(err)
	}
	reporter := &recordingWorkerLifecycleReporter{}
	executor := &BuildExecutor{deploymentRepo: deploymentRepo, reporter: reporter}
	hub := NewSSEHub()
	logger, err := NewBuildLogger(filepath.Join(t.TempDir(), "feedback.log"), hub, deployment.ID, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	executor.reportLifecycle(ctx, project, &stale, logger)
	if reporter.status != models.DeploymentStatusReady || reporter.prNumber == nil || *reporter.prNumber != 42 {
		t.Fatalf("reported status/PR = %q/%v, want ready/42", reporter.status, reporter.prNumber)
	}
	if stale.Status != models.DeploymentStatusReady || stale.GitHubPRNumber == nil || *stale.GitHubPRNumber != 42 {
		t.Fatalf("executor snapshot was not refreshed: %+v", stale)
	}
}

type recordingPostBuildHook struct {
	failures int
}

type recordingCancellationHook struct {
	cancelled int
}

func (h *recordingCancellationHook) OnBuildSuccess(context.Context, *models.Project, *models.Deployment) error {
	return nil
}

func (h *recordingCancellationHook) OnBuildFailure(context.Context, *models.Project, *models.Deployment, error) error {
	return nil
}

func (h *recordingCancellationHook) OnBuildCancelled(context.Context, *models.Project, *models.Deployment) error {
	h.cancelled++
	return nil
}

func (h *recordingPostBuildHook) OnBuildSuccess(context.Context, *models.Project, *models.Deployment) error {
	return nil
}

func (h *recordingPostBuildHook) OnBuildFailure(context.Context, *models.Project, *models.Deployment, error) error {
	h.failures++
	return nil
}

func createGitRemote(t *testing.T) (remoteURL, firstSHA, headSHA string) {
	t.Helper()
	work := filepath.Join(t.TempDir(), "work")
	runGitTest(t, "init", "--quiet", "--initial-branch=main", work)
	runGitTest(t, "-C", work, "config", "user.email", "test@hostbox.local")
	runGitTest(t, "-C", work, "config", "user.name", "Hostbox Test")
	if err := os.WriteFile(filepath.Join(work, "index.html"), []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, "-C", work, "add", "index.html")
	runGitTest(t, "-C", work, "commit", "--quiet", "-m", "first")
	firstSHA = strings.TrimSpace(runGitTest(t, "-C", work, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(work, "index.html"), []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, "-C", work, "commit", "--quiet", "-am", "second")
	headSHA = strings.TrimSpace(runGitTest(t, "-C", work, "rev-parse", "HEAD"))

	bare := filepath.Join(t.TempDir(), "origin.git")
	runGitTest(t, "clone", "--quiet", "--bare", work, bare)
	return "file://" + bare, firstSHA, headSHA
}

func runGitTest(t *testing.T, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

func TestEffectiveBuildMemoryMB_PreservesConfiguredMemory(t *testing.T) {
	t.Parallel()

	got := effectiveBuildMemoryMB(1536, t.TempDir(), &detect.PackageJSON{})
	if got != 1536 {
		t.Fatalf("expected configured build memory to be preserved, got %d", got)
	}
}

func TestDescribeContainerExecError_AnnotatesOOMKill(t *testing.T) {
	t.Parallel()

	got := describeContainerExecError(assertErr("command exited with code 137"), 1024)
	want := "command exited with code 137 — build container was killed, likely due to memory pressure; increase BUILD_MEMORY_MB above 1024"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDescribeContainerExecError_PassesThroughOtherErrors(t *testing.T) {
	t.Parallel()

	got := describeContainerExecError(assertErr("command exited with code 1"), 1024)
	if got != "command exited with code 1" {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func TestDescribeArtifactErrorCategories(t *testing.T) {
	tests := []struct {
		message string
		prefix  string
	}{
		{"artifact exceeds maximum size of 100 B", "Artifact output oversized:"},
		{"artifact contains symbolic link", "Artifact output unsafe:"},
		{"artifact contains non-regular entry", "Artifact output unsafe:"},
		{"output directory does not exist", "Artifact output missing or unreadable:"},
	}
	for _, test := range tests {
		if got := describeArtifactError(errors.New(test.message)); !strings.HasPrefix(got, test.prefix) {
			t.Errorf("describeArtifactError(%q) = %q, want prefix %q", test.message, got, test.prefix)
		}
	}
}

func TestBaseBuildEnvVars_DoesNotForceNodeEnv(t *testing.T) {
	t.Parallel()

	project := &models.Project{ID: "project-1", Name: "Manifest"}
	deployment := &models.Deployment{
		ID:           "deploy-1",
		Branch:       "main",
		CommitSHA:    "abc123",
		IsProduction: true,
	}

	vars := baseBuildEnvVars(project, deployment)

	if slices.Contains(vars, "NODE_ENV=production") {
		t.Fatal("build env should not force NODE_ENV=production")
	}
	if !slices.Contains(vars, "HOSTBOX_IS_PREVIEW=false") {
		t.Fatal("expected production build env flag")
	}
}

func assertErr(msg string) error {
	return simpleError(msg)
}

type simpleError string

func (e simpleError) Error() string {
	return string(e)
}
