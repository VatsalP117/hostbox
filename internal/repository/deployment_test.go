package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/VatsalP117/hostbox/internal/models"
)

func createTestProject(t *testing.T, db *sql.DB) (*models.User, *models.Project) {
	t.Helper()
	ctx := context.Background()
	user := &models.User{Email: "deploy@test.com", PasswordHash: "hash"}
	NewUserRepository(db).Create(ctx, user)

	project := &models.Project{
		OwnerID: user.ID, Name: "Deploy Test", Slug: "deploy-test",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	NewProjectRepository(db).Create(ctx, project)
	return user, project
}

func TestDeploymentRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	deployment := &models.Deployment{
		ProjectID: project.ID,
		CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch:    "main",
		Status:    models.DeploymentStatusQueued,
	}
	if err := repo.Create(ctx, deployment); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CommitSHA != deployment.CommitSHA {
		t.Errorf("CommitSHA mismatch")
	}
	if got.Status != models.DeploymentStatusQueued {
		t.Errorf("Status = %q, want queued", got.Status)
	}
}

func TestDeploymentRepository_UpdateIfStatusIsCompareAndSet(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()
	deployment := &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusBuilding,
	}
	if err := repo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(ctx, deployment.ID, models.DeploymentStatusCancelled, nil); err != nil {
		t.Fatal(err)
	}

	deployment.Status = models.DeploymentStatusFailed
	updated, err := repo.UpdateIfStatus(ctx, deployment, models.DeploymentStatusBuilding)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("cancelled deployment must not be overwritten")
	}
	got, err := repo.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.DeploymentStatusCancelled {
		t.Fatalf("status = %q, want cancelled", got.Status)
	}
}

func TestDeploymentRepository_UpdateResolvedCommit(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()
	deployment := &models.Deployment{ProjectID: project.ID, CommitSHA: "manual", Branch: "main", Status: models.DeploymentStatusQueued}
	if err := repo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}

	sha := "0123456789abcdef0123456789abcdef01234567"
	if err := repo.UpdateResolvedCommit(ctx, deployment.ID, sha); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CommitSHA != sha {
		t.Fatalf("commit SHA = %q, want %q", got.CommitSHA, sha)
	}
}

func TestDeploymentRepository_SetGitHubDeployIDIfUnset(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()
	deployment := &models.Deployment{ProjectID: project.ID, CommitSHA: "abc123", Branch: "main", Status: models.DeploymentStatusQueued}
	if err := repo.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}

	stored, err := repo.SetGitHubDeployIDIfUnset(ctx, deployment.ID, 1234)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 1234 {
		t.Fatalf("stored ID = %d, want 1234", stored)
	}
	stored, err = repo.SetGitHubDeployIDIfUnset(ctx, deployment.ID, 5678)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 1234 {
		t.Fatalf("second stored ID = %d, want original 1234", stored)
	}
	got, err := repo.GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.GitHubDeployID == nil || *got.GitHubDeployID != 1234 {
		t.Fatalf("persisted GitHub ID = %v, want 1234", got.GitHubDeployID)
	}
}

func TestDeploymentRepository_BranchScopedCommitLookupAndPRAssociation(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()
	sha := "same-commit"
	mainDeployment := &models.Deployment{ProjectID: project.ID, CommitSHA: sha, Branch: "main", Status: models.DeploymentStatusQueued}
	previewDeployment := &models.Deployment{ProjectID: project.ID, CommitSHA: sha, Branch: "feature/test", Status: models.DeploymentStatusBuilding}
	if err := repo.Create(ctx, mainDeployment); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, previewDeployment); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByCommitSHAAndBranch(ctx, project.ID, sha, "feature/test")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != previewDeployment.ID {
		t.Fatalf("lookup ID = %q, want preview %q", got.ID, previewDeployment.ID)
	}
	storedPR, err := repo.SetGitHubPRNumberIfUnset(ctx, previewDeployment.ID, 42)
	if err != nil || storedPR != 42 {
		t.Fatalf("PR association = (%d, %v), want (42, nil)", storedPR, err)
	}
	storedPR, err = repo.SetGitHubPRNumberIfUnset(ctx, previewDeployment.ID, 99)
	if err != nil || storedPR != 42 {
		t.Fatalf("second PR association = (%d, %v), want original 42", storedPR, err)
	}
}

func TestDeploymentRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	d := &models.Deployment{
		ProjectID: project.ID,
		CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch:    "main",
		Status:    models.DeploymentStatusQueued,
	}
	repo.Create(ctx, d)

	errMsg := "build failed"
	repo.UpdateStatus(ctx, d.ID, models.DeploymentStatusFailed, &errMsg)

	got, _ := repo.GetByID(ctx, d.ID)
	if got.Status != models.DeploymentStatusFailed {
		t.Errorf("Status = %q, want failed", got.Status)
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != "build failed" {
		t.Errorf("ErrorMessage mismatch")
	}
}

func TestDeploymentRepository_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.Create(ctx, &models.Deployment{
			ProjectID: project.ID,
			CommitSHA: "abc123def456abc123def456abc123def456abc1",
			Branch:    "main",
			Status:    models.DeploymentStatusQueued,
		})
	}

	deployments, total, err := repo.ListByProject(ctx, project.ID, 1, 3, nil, nil)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(deployments) != 3 {
		t.Errorf("len = %d, want 3", len(deployments))
	}
}

func TestDeploymentRepository_ListByProject_StatusFilter(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusReady,
	})
	repo.Create(ctx, &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusFailed,
	})

	status := "ready"
	deployments, total, _ := repo.ListByProject(ctx, project.ID, 1, 10, &status, nil)
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(deployments) != 1 {
		t.Errorf("len = %d, want 1", len(deployments))
	}
}

func TestDeploymentRepository_CancelQueuedByProjectAndBranch(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusQueued,
	})
	repo.Create(ctx, &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusBuilding,
	})

	cancelled, err := repo.CancelQueuedByProjectAndBranch(ctx, project.ID, "main")
	if err != nil {
		t.Fatalf("CancelQueued: %v", err)
	}
	if cancelled != 1 {
		t.Errorf("cancelled = %d, want 1", cancelled)
	}
}

func TestDeploymentRepository_DeactivateBranchDeploymentsIsRetryableAndStopsActiveBuilds(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	var previewIDs []string
	for _, status := range []models.DeploymentStatus{
		models.DeploymentStatusQueued,
		models.DeploymentStatusBuilding,
		models.DeploymentStatusReady,
		models.DeploymentStatusCancelled,
	} {
		deployment := &models.Deployment{
			ProjectID: project.ID, CommitSHA: "abc123", Branch: "feature/retry",
			Status: status, IsProduction: false,
		}
		if err := repo.Create(ctx, deployment); err != nil {
			t.Fatal(err)
		}
		previewIDs = append(previewIDs, deployment.ID)
	}
	production := &models.Deployment{
		ProjectID: project.ID, CommitSHA: "def456", Branch: "feature/retry",
		Status: models.DeploymentStatusReady, IsProduction: true,
	}
	if err := repo.Create(ctx, production); err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		deployments, err := repo.DeactivateBranchDeployments(ctx, project.ID, "feature/retry")
		if err != nil {
			t.Fatal(err)
		}
		if len(deployments) != len(previewIDs) {
			t.Fatalf("attempt %d returned %d previews, want %d", attempt, len(deployments), len(previewIDs))
		}
	}

	for _, id := range previewIDs {
		deployment, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if deployment.Status != models.DeploymentStatusCancelled {
			t.Fatalf("preview %s status = %q, want cancelled", id, deployment.Status)
		}
		if deployment.CompletedAt == nil {
			t.Fatalf("preview %s missing completion time", id)
		}
	}
	gotProduction, err := repo.GetByID(ctx, production.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotProduction.Status != models.DeploymentStatusReady {
		t.Fatalf("production status = %q, want ready", gotProduction.Status)
	}
}

func TestDeploymentRepository_GetLatestByProjectAndBranch(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusReady,
	})
	second := &models.Deployment{
		ProjectID: project.ID, CommitSHA: "def456abc123def456abc123def456abc123def4",
		Branch: "main", Status: models.DeploymentStatusQueued,
	}
	repo.Create(ctx, second)

	got, err := repo.GetLatestByProjectAndBranch(ctx, project.ID, "main")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got.ID != second.ID {
		t.Errorf("expected latest deployment, got %q", got.ID)
	}
}

func TestDeploymentRepository_CascadeDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc123def456abc123def456abc123def456abc1",
		Branch: "main", Status: models.DeploymentStatusQueued,
	})

	NewProjectRepository(db).Delete(ctx, project.ID)

	count, _ := repo.CountByProject(ctx, project.ID)
	if count != 0 {
		t.Errorf("expected 0 deployments after project delete, got %d", count)
	}
}
