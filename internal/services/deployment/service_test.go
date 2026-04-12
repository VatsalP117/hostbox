package deployment

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/internal/util"
	"github.com/vatsalpatel/hostbox/migrations"
)

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
		path := "/tmp/artifacts"
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

func TestService_CancelDeployment_InvalidStatus(t *testing.T) {
	svc, deployRepo, projectRepo, userID := newTestService(t)
	project := createTestProject(t, projectRepo, userID)
	dep := createTestDeployment(t, deployRepo, project.ID, "ready", true)

	_, err := svc.CancelDeployment(context.Background(), dep.ID)
	if err == nil {
		t.Fatal("expected error when cancelling a ready deployment")
	}
}

func TestService_CancelDeployment_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.CancelDeployment(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent deployment")
	}
}
