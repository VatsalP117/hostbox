package deployment

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/util"
	"github.com/VatsalP117/hostbox/migrations"
)

type recordingActivator struct {
	activations []ProductionActivation
	err         error
}

type recordingLifecycleReporter struct {
	statuses []models.DeploymentStatus
	err      error
}

func (r *recordingLifecycleReporter) Report(_ context.Context, _ *models.Project, deployment *models.Deployment) error {
	r.statuses = append(r.statuses, deployment.Status)
	return r.err
}

func (a *recordingActivator) ActivateProduction(_ context.Context, activation ProductionActivation) error {
	a.activations = append(a.activations, activation)
	return a.err
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func createTestUser(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := util.NewID()
	_, err := db.Exec(`INSERT INTO users (id, email, password_hash, is_admin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, id+"@test.com", "hash", true, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func createTestProject(t *testing.T, repo *repository.ProjectRepository, ownerID string) *models.Project {
	t.Helper()
	fw := "nextjs"
	p := &models.Project{
		ID:               util.NewID(),
		OwnerID:          ownerID,
		Name:             "Test Project",
		Slug:             "test-project",
		Framework:        &fw,
		ProductionBranch: "main",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func createTestDeployment(t *testing.T, repo *repository.DeploymentRepository, projectID, status string, production bool) *models.Deployment {
	t.Helper()
	now := time.Now().UTC()
	d := &models.Deployment{
		ID:           util.NewID(),
		ProjectID:    projectID,
		CommitSHA:    "abc123def456",
		Branch:       "main",
		Status:       models.DeploymentStatus(status),
		IsProduction: production,
		CreatedAt:    now,
	}
	if status == "ready" {
		d.CompletedAt = &now
		path := t.TempDir()
		if err := os.WriteFile(filepath.Join(path, "index.html"), []byte("ready"), 0o600); err != nil {
			t.Fatal(err)
		}
		d.ArtifactPath = &path
		url := "https://test-project.example.com"
		d.DeploymentURL = &url
	}
	if err := repo.Create(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return d
}

func newTestService(t *testing.T) (*Service, *repository.DeploymentRepository, *repository.ProjectRepository, string) {
	t.Helper()
	db := setupTestDB(t)
	deployRepo := repository.NewDeploymentRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	logger := slog.Default()
	userID := createTestUser(t, db)

	svc := &Service{
		deployRepo:     deployRepo,
		projectRepo:    projectRepo,
		activator:      &recordingActivator{},
		platformDomain: "example.com",
		logger:         logger,
	}
	return svc, deployRepo, projectRepo, userID
}

func TestService_GetDeployment(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "queued", false)

	got, err := svc.GetDeployment(context.Background(), dep.ID)
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if got.ID != dep.ID {
		t.Errorf("got ID %s, want %s", got.ID, dep.ID)
	}
}

func TestService_QueuedSupersessionReportsCancellationOnce(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	reporter := &recordingLifecycleReporter{}
	svc.SetLifecycleReporter(reporter)

	first, err := svc.TriggerDeployment(context.Background(), TriggerRequest{
		ProjectID: project.ID, Branch: "main", CommitSHA: "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	reporter.statuses = nil
	if _, err := svc.TriggerDeployment(context.Background(), TriggerRequest{
		ProjectID: project.ID, Branch: "main", CommitSHA: "second",
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := deployRepo.GetByID(context.Background(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.DeploymentStatusCancelled {
		t.Fatalf("superseded status = %q, want cancelled", stored.Status)
	}
	want := []models.DeploymentStatus{
		models.DeploymentStatusCancelled,
		models.DeploymentStatusQueued,
	}
	if len(reporter.statuses) != len(want) {
		t.Fatalf("reported statuses = %v, want %v", reporter.statuses, want)
	}
	for i := range want {
		if reporter.statuses[i] != want[i] {
			t.Fatalf("reported statuses = %v, want %v", reporter.statuses, want)
		}
	}
}

func TestService_TriggerDeploymentSnapshotsProjectBuildSettings(t *testing.T) {
	svc, _, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	project.RootDirectory = "apps/web"
	project.NodeVersion = "22"
	project.BuildCommand = stringPointer("npm run export")
	project.InstallCommand = stringPointer("npm ci")
	project.OutputDirectory = stringPointer("out")
	repositoryName := "owner/original"
	installationID := int64(42)
	project.GitHubRepo = &repositoryName
	project.GitHubInstallationID = &installationID
	if err := projectRepo.Update(context.Background(), project); err != nil {
		t.Fatal(err)
	}

	deployment, err := svc.TriggerDeployment(context.Background(), TriggerRequest{
		ProjectID: project.ID, Branch: "main", CommitSHA: "manual",
	})
	if err != nil {
		t.Fatal(err)
	}
	project.RootDirectory = "apps/changed"
	project.NodeVersion = "20"
	project.BuildCommand = stringPointer("npm run changed")
	changedRepository := "owner/changed"
	project.GitHubRepo = &changedRepository
	if err := projectRepo.Update(context.Background(), project); err != nil {
		t.Fatal(err)
	}

	if deployment.BuildRootDirectory != filepath.Join("apps", "web") ||
		deployment.BuildNodeVersion != "22" ||
		testPointerValue(deployment.BuildCommand) != "npm run export" ||
		testPointerValue(deployment.BuildInstallCommand) != "npm ci" ||
		testPointerValue(deployment.BuildOutputDirectory) != "out" ||
		testPointerValue(deployment.SourceRepository) != repositoryName ||
		deployment.SourceInstallationID == nil || *deployment.SourceInstallationID != installationID {
		t.Fatalf("deployment did not retain project snapshot: %+v", deployment)
	}
	if deployment.BuildManifestResolved {
		t.Fatal("new branch deployment must remain unresolved until worker detection")
	}
}

func TestService_RebuildAndDeployLatestHaveDistinctSourceSemantics(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	project.RootDirectory = "apps/current"
	project.NodeVersion = "22"
	if err := projectRepo.Update(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	framework := "vite"
	mode := "spa"
	manager := "pnpm"
	version := "9.12.0"
	output := "dist"
	install := "pnpm install --frozen-lockfile"
	build := "pnpm run build"
	sourceRepository := "owner/original"
	sourceInstallationID := int64(7)
	source := &models.Deployment{
		ProjectID: project.ID,
		CommitSHA: strings.Repeat("a", 40),
		Branch:    "main", Status: models.DeploymentStatusReady, IsProduction: true,
		BuildFramework: &framework, BuildServingMode: &mode,
		BuildPackageManager: &manager, BuildPackageManagerVersion: &version,
		BuildNodeVersion: "20", BuildRootDirectory: "apps/old",
		BuildOutputDirectory: &output, BuildInstallCommand: &install,
		BuildCommand: &build, BuildLockFileHash: "lock-old",
		BuildManifestResolved: true,
		SourceRepository:      &sourceRepository, SourceInstallationID: &sourceInstallationID,
	}
	if err := deployRepo.Create(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	project.ProductionBranch = "release"
	if err := projectRepo.Update(context.Background(), project); err != nil {
		t.Fatal(err)
	}

	rebuilt, err := svc.RebuildDeployment(context.Background(), source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.CommitSHA != source.CommitSHA || !rebuilt.BuildManifestResolved ||
		rebuilt.BuildRootDirectory != "apps/old" ||
		testPointerValue(rebuilt.BuildCommand) != build ||
		rebuilt.BuildLockFileHash != "lock-old" ||
		testPointerValue(rebuilt.SourceRepository) != sourceRepository ||
		!rebuilt.IsProduction {
		t.Fatalf("rebuild did not copy immutable source manifest: %+v", rebuilt)
	}

	latest, err := svc.DeployLatest(context.Background(), project.ID, "main")
	if err != nil {
		t.Fatal(err)
	}
	if latest.CommitSHA != "manual" || latest.BuildManifestResolved ||
		latest.BuildRootDirectory != filepath.Join("apps", "current") ||
		latest.BuildNodeVersion != "22" {
		t.Fatalf("deploy latest did not use fresh project snapshot: %+v", latest)
	}
}

func testPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func TestService_RebuildRejectsLegacyDeploymentWithoutManifest(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	source := &models.Deployment{
		ProjectID: project.ID, CommitSHA: strings.Repeat("a", 40),
		Branch: "main", Status: models.DeploymentStatusFailed,
	}
	if err := deployRepo.Create(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RebuildDeployment(context.Background(), source.ID); err == nil ||
		!strings.Contains(err.Error(), "resolved build manifest") {
		t.Fatalf("expected unresolved manifest rejection, got %v", err)
	}
}

func TestService_GetDeployment_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.GetDeployment(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent deployment")
	}
}

func TestService_ListDeployments(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)

	for i := 0; i < 3; i++ {
		createTestDeployment(t, deployRepo, project.ID, "ready", true)
	}

	deployments, total, err := svc.ListDeployments(context.Background(), project.ID, ListOpts{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(deployments) != 3 {
		t.Errorf("len = %d, want 3", len(deployments))
	}
}

func TestService_Rollback(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	target := createTestDeployment(t, deployRepo, project.ID, "ready", true)

	rolled, err := svc.Rollback(context.Background(), project.ID, target.ID)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if rolled.ID == target.ID {
		t.Error("rollback should create a new deployment")
	}
	if rolled.Status != models.DeploymentStatusReady {
		t.Errorf("status = %s, want ready", rolled.Status)
	}
	if !rolled.IsRollback {
		t.Error("expected IsRollback = true")
	}
	if rolled.RollbackSourceID == nil || *rolled.RollbackSourceID != target.ID {
		t.Error("expected RollbackSourceID to point to target")
	}
	if rolled.ArtifactPath == nil || *rolled.ArtifactPath != *target.ArtifactPath {
		t.Error("expected same artifact path")
	}
	if rolled.DeploymentURL == nil || *rolled.DeploymentURL != "https://test-project.example.com" {
		t.Errorf("unexpected deployment URL: %v", rolled.DeploymentURL)
	}

	activator := svc.activator.(*recordingActivator)
	if len(activator.activations) != 1 {
		t.Fatalf("activation count = %d, want 1", len(activator.activations))
	}
	activation := activator.activations[0]
	if activation.ProjectID != project.ID || activation.ProjectSlug != project.Slug {
		t.Errorf("unexpected activation project: %+v", activation)
	}
	if activation.ArtifactPath != *target.ArtifactPath || activation.Framework != *project.Framework {
		t.Errorf("unexpected activation artifact: %+v", activation)
	}
}

func TestService_Rollback_NotReady(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "failed", false)

	_, err := svc.Rollback(context.Background(), project.ID, dep.ID)
	if err == nil {
		t.Fatal("expected error for non-ready deployment")
	}
}

func TestService_Rollback_WrongProject(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "ready", true)

	_, err := svc.Rollback(context.Background(), "other-project", dep.ID)
	if err == nil {
		t.Fatal("expected error for wrong project")
	}
}

func TestService_Promote(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	source := createTestDeployment(t, deployRepo, project.ID, "ready", false)

	promoted, err := svc.Promote(context.Background(), project.ID, source.ID)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	if promoted.ID == source.ID {
		t.Error("promote should create a new deployment")
	}
	if promoted.Status != models.DeploymentStatusReady {
		t.Errorf("status = %s, want ready", promoted.Status)
	}
	if !promoted.IsProduction {
		t.Error("expected IsProduction = true")
	}
	if promoted.Branch != project.ProductionBranch {
		t.Errorf("branch = %s, want %s", promoted.Branch, project.ProductionBranch)
	}
	if promoted.DeploymentURL == nil || *promoted.DeploymentURL != "https://test-project.example.com" {
		t.Errorf("unexpected URL: %v", promoted.DeploymentURL)
	}

	activator := svc.activator.(*recordingActivator)
	if len(activator.activations) != 1 {
		t.Fatalf("activation count = %d, want 1", len(activator.activations))
	}
	activation := activator.activations[0]
	if activation.ProjectID != project.ID || activation.ProjectSlug != project.Slug {
		t.Errorf("unexpected activation project: %+v", activation)
	}
	if activation.ArtifactPath != *source.ArtifactPath || activation.Framework != *project.Framework {
		t.Errorf("unexpected activation artifact: %+v", activation)
	}
}

func TestService_Promote_NotReady(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "building", false)

	_, err := svc.Promote(context.Background(), project.ID, dep.ID)
	if err == nil {
		t.Fatal("expected error for non-ready deployment")
	}
}

func TestService_RollbackAndPromote_RejectInvalidArtifactsBeforeCreateOrActivate(t *testing.T) {
	operations := map[string]func(*Service, context.Context, string, string) (*models.Deployment, error){
		"rollback": (*Service).Rollback,
		"promote":  (*Service).Promote,
	}

	artifactCases := map[string]func(*testing.T) *string{
		"nil": func(*testing.T) *string { return nil },
		"blank": func(*testing.T) *string {
			path := "  "
			return &path
		},
		"missing": func(t *testing.T) *string {
			path := filepath.Join(t.TempDir(), "missing")
			return &path
		},
		"empty directory": func(t *testing.T) *string {
			path := t.TempDir()
			return &path
		},
		"file": func(t *testing.T) *string {
			path := filepath.Join(t.TempDir(), "artifact.txt")
			if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
				t.Fatal(err)
			}
			return &path
		},
	}

	for operationName, operation := range operations {
		for caseName, artifact := range artifactCases {
			t.Run(operationName+"/"+caseName, func(t *testing.T) {
				svc, deployRepo, projectRepo, userID := newTestService(t)
				project := createTestProject(t, projectRepo, userID)
				source := createTestDeployment(t, deployRepo, project.ID, "ready", false)
				source.ArtifactPath = artifact(t)
				if err := deployRepo.Update(context.Background(), source); err != nil {
					t.Fatal(err)
				}

				before, err := deployRepo.CountByProject(context.Background(), project.ID)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := operation(svc, context.Background(), project.ID, source.ID); err == nil {
					t.Fatal("expected invalid artifact error")
				}

				after, err := deployRepo.CountByProject(context.Background(), project.ID)
				if err != nil {
					t.Fatal(err)
				}
				if after != before {
					t.Errorf("deployment count = %d, want unchanged %d", after, before)
				}
				if got := len(svc.activator.(*recordingActivator).activations); got != 0 {
					t.Errorf("activation count = %d, want 0", got)
				}
			})
		}
	}
}

func TestService_RollbackAndPromote_ActivationFailureIsReturnedAndRecorded(t *testing.T) {
	operations := map[string]func(*Service, context.Context, string, string) (*models.Deployment, error){
		"rollback": (*Service).Rollback,
		"promote":  (*Service).Promote,
	}

	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			svc, deployRepo, projectRepo, userID := newTestService(t)
			project := createTestProject(t, projectRepo, userID)
			source := createTestDeployment(t, deployRepo, project.ID, "ready", false)
			activationErr := errors.New("caddy unavailable")
			svc.activator.(*recordingActivator).err = activationErr

			deployment, err := operation(svc, context.Background(), project.ID, source.ID)
			if !errors.Is(err, activationErr) {
				t.Fatalf("error = %v, want wrapped activation error", err)
			}
			if deployment != nil {
				t.Fatalf("deployment = %+v, want nil on activation failure", deployment)
			}

			deployments, total, err := deployRepo.ListByProject(context.Background(), project.ID, 1, 10, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if total != 2 {
				t.Fatalf("deployment total = %d, want 2", total)
			}
			var failed *models.Deployment
			for i := range deployments {
				if deployments[i].ID != source.ID {
					failed = &deployments[i]
					break
				}
			}
			if failed == nil {
				t.Fatal("created deployment not found")
			}
			if failed.Status != models.DeploymentStatusFailed {
				t.Errorf("status = %s, want failed", failed.Status)
			}
			if failed.ErrorMessage == nil || !strings.Contains(*failed.ErrorMessage, activationErr.Error()) {
				t.Errorf("error message = %v, want activation failure", failed.ErrorMessage)
			}
		})
	}
}

func TestService_CancelDeployment_InvalidStatus(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "ready", true)

	_, err := svc.CancelDeployment(context.Background(), dep.ID)
	if err == nil {
		t.Fatal("expected error when cancelling a ready deployment")
	}
}

func TestService_CancelDeployment_UsesTerminalCompareAndSet(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	reporter := &recordingLifecycleReporter{}
	svc.SetLifecycleReporter(reporter)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "queued", false)

	cancelled, err := svc.CancelDeployment(context.Background(), dep.ID)
	if err != nil {
		t.Fatalf("CancelDeployment: %v", err)
	}
	if cancelled.Status != models.DeploymentStatusCancelled || cancelled.CompletedAt == nil {
		t.Fatalf("cancelled deployment = %+v", cancelled)
	}
	stored, err := deployRepo.GetByID(context.Background(), dep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.DeploymentStatusCancelled {
		t.Fatalf("stored status = %q, want cancelled", stored.Status)
	}
	if len(reporter.statuses) != 1 || reporter.statuses[0] != models.DeploymentStatusCancelled {
		t.Fatalf("reported statuses = %v, want [cancelled]", reporter.statuses)
	}
}

func TestService_LifecycleFeedbackFailureDoesNotUndoCancellation(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	svc.SetLifecycleReporter(&recordingLifecycleReporter{err: errors.New("github unavailable")})
	project := createTestProject(t, projectRepo, userID)
	deployment := createTestDeployment(t, deployRepo, project.ID, "queued", false)

	if _, err := svc.CancelDeployment(context.Background(), deployment.ID); err != nil {
		t.Fatalf("CancelDeployment should remain successful: %v", err)
	}
	stored, err := deployRepo.GetByID(context.Background(), deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.DeploymentStatusCancelled {
		t.Fatalf("stored status = %q, want cancelled", stored.Status)
	}
}

func TestService_DeactivateBranchReportsCancelledOnceAcrossCleanupRetries(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	reporter := &recordingLifecycleReporter{}
	svc.SetLifecycleReporter(reporter)
	project := createTestProject(t, projectRepo, userID)
	deployment := &models.Deployment{
		ID: util.NewID(), ProjectID: project.ID, CommitSHA: "abc123", Branch: "feature/cleanup",
		Status: models.DeploymentStatusReady, IsProduction: false, CreatedAt: time.Now().UTC(),
	}
	if err := deployRepo.Create(context.Background(), deployment); err != nil {
		t.Fatal(err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		deployments, err := svc.DeactivateBranchDeployments(context.Background(), project.ID, deployment.Branch)
		if err != nil {
			t.Fatal(err)
		}
		if len(deployments) != 1 {
			t.Fatalf("attempt %d returned %d deployments", attempt+1, len(deployments))
		}
	}
	if len(reporter.statuses) != 1 || reporter.statuses[0] != models.DeploymentStatusCancelled {
		t.Fatalf("reported statuses = %v, want one cancelled", reporter.statuses)
	}
}

func TestService_AssociatePullRequestPersistsAndReportsCurrentState(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	reporter := &recordingLifecycleReporter{}
	svc.SetLifecycleReporter(reporter)
	project := createTestProject(t, projectRepo, userID)
	deployment := createTestDeployment(t, deployRepo, project.ID, "building", false)
	stale := *deployment
	if err := deployRepo.UpdateStatus(context.Background(), deployment.ID, models.DeploymentStatusReady, nil); err != nil {
		t.Fatal(err)
	}

	if err := svc.AssociatePullRequest(context.Background(), &stale, 42); err != nil {
		t.Fatal(err)
	}
	stored, err := deployRepo.GetByID(context.Background(), deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GitHubPRNumber == nil || *stored.GitHubPRNumber != 42 {
		t.Fatalf("stored PR number = %v, want 42", stored.GitHubPRNumber)
	}
	if len(reporter.statuses) != 1 || reporter.statuses[0] != models.DeploymentStatusReady {
		t.Fatalf("reported statuses = %v, want [ready]", reporter.statuses)
	}
	if stale.Status != models.DeploymentStatusReady || stale.GitHubPRNumber == nil {
		t.Fatalf("caller snapshot was not refreshed: %+v", stale)
	}
}

func TestService_CancelDeployment_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.CancelDeployment(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent deployment")
	}
}
