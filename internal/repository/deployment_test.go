package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
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
