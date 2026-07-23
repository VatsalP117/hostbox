package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/VatsalP117/hostbox/internal/models"
)

func createTestUserForProject(t *testing.T, db *sql.DB, email string) *models.User {
	t.Helper()
	repo := NewUserRepository(db)
	user := &models.User{Email: email, PasswordHash: "hash"}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user
}

func TestProjectRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "proj@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := &models.Project{
		OwnerID:          user.ID,
		Name:             "My Project",
		Slug:             "my-project",
		ProductionBranch: "main",
		RootDirectory:    "/",
		NodeVersion:      "20",
		AutoDeploy:       true,
	}
	if err := repo.Create(ctx, project); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "My Project" {
		t.Errorf("Name = %q, want %q", got.Name, "My Project")
	}
	if got.Slug != "my-project" {
		t.Errorf("Slug = %q, want %q", got.Slug, "my-project")
	}
}

func TestProjectRepository_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "slug@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := &models.Project{
		OwnerID: user.ID, Name: "Test", Slug: "test-slug",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	repo.Create(ctx, project)

	got, err := repo.GetBySlug(ctx, "test-slug")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != project.ID {
		t.Errorf("ID mismatch")
	}
}

func TestProjectRepository_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "dupslug@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	p := models.Project{
		OwnerID: user.ID, Name: "P1", Slug: "same-slug",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	repo.Create(ctx, &p)

	p2 := models.Project{
		OwnerID: user.ID, Name: "P2", Slug: "same-slug",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	err := repo.Create(ctx, &p2)
	if err == nil {
		t.Fatal("expected error for duplicate slug")
	}
}

func TestProjectRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "upd@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := &models.Project{
		OwnerID: user.ID, Name: "Before", Slug: "before",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	repo.Create(ctx, project)

	project.Name = "After"
	if err := repo.Update(ctx, project); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, project.ID)
	if got.Name != "After" {
		t.Errorf("Name = %q, want %q", got.Name, "After")
	}
}

func TestProjectRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "del@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := &models.Project{
		OwnerID: user.ID, Name: "Del", Slug: "del",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	repo.Create(ctx, project)
	repo.Delete(ctx, project.ID)

	_, err := repo.GetByID(ctx, project.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestProjectRepository_ListByOwner(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "list@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.Create(ctx, &models.Project{
			OwnerID: user.ID, Name: "Project " + string(rune('A'+i)),
			Slug: "proj-" + string(rune('a'+i)), ProductionBranch: "main",
			RootDirectory: "/", NodeVersion: "20",
		})
	}

	projects, total, err := repo.ListByOwner(ctx, user.ID, 1, 3, "")
	if err != nil {
		t.Fatalf("ListByOwner: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(projects) != 3 {
		t.Errorf("len = %d, want 3", len(projects))
	}
}

func TestProjectRepository_ListByOwner_Search(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "search@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Project{
		OwnerID: user.ID, Name: "Frontend App", Slug: "frontend",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	})
	repo.Create(ctx, &models.Project{
		OwnerID: user.ID, Name: "Backend API", Slug: "backend",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	})

	projects, total, _ := repo.ListByOwner(ctx, user.ID, 1, 10, "Front")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(projects) != 1 || projects[0].Name != "Frontend App" {
		t.Errorf("expected to find 'Frontend App'")
	}
}

func TestProjectRepository_GitHubSourceLifecycleAndFanout(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "github-lifecycle@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()
	repositoryName := "Octo/Repo"
	installationID := int64(99)
	repositoryID := int64(123)

	var createdProjects []*models.Project
	for _, project := range []*models.Project{
		{
			OwnerID: user.ID, Name: "Frontend", Slug: "frontend-github",
			GitHubRepo: &repositoryName, GitHubInstallationID: &installationID, GitHubRepositoryID: &repositoryID,
			ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
		},
		{
			OwnerID: user.ID, Name: "Docs", Slug: "docs-github",
			GitHubRepo: &repositoryName, GitHubInstallationID: &installationID, GitHubRepositoryID: &repositoryID,
			ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
		},
	} {
		if err := repo.Create(ctx, project); err != nil {
			t.Fatal(err)
		}
		createdProjects = append(createdProjects, project)
	}
	sourceRepository := "octo/repo"
	sourceInstallationID := int64(99)
	deployment := &models.Deployment{
		ProjectID: createdProjects[0].ID, CommitSHA: "0123456789012345678901234567890123456789",
		Branch: "main", Status: models.DeploymentStatusFailed,
		SourceRepository: &sourceRepository, SourceInstallationID: &sourceInstallationID,
	}
	if err := NewDeploymentRepository(db).Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}

	projects, err := repo.ListByGitHubSource(ctx, installationID, "octo/repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 {
		t.Fatalf("active projects = %d, want 2", len(projects))
	}

	if err := repo.SetInstallationStatus(ctx, installationID, models.GitHubConnectionSuspended); err != nil {
		t.Fatal(err)
	}
	projects, err = repo.ListByGitHubSource(ctx, installationID, "octo/repo")
	if err != nil || len(projects) != 0 {
		t.Fatalf("suspended projects = %d, err=%v", len(projects), err)
	}
	if err := repo.SetInstallationStatus(ctx, installationID, models.GitHubConnectionActive); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetRepositoryAccess(ctx, installationID, []string{"octo/repo"}, false); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetInstallationStatus(ctx, installationID, models.GitHubConnectionSuspended); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetInstallationStatus(ctx, installationID, models.GitHubConnectionActive); err != nil {
		t.Fatal(err)
	}
	projects, err = repo.ListByGitHubSource(ctx, installationID, "octo/repo")
	if err != nil || len(projects) != 0 {
		t.Fatalf("access-removed projects = %d, err=%v", len(projects), err)
	}
	if err := repo.SetRepositoryAccess(ctx, installationID, []string{"octo/repo"}, true); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdateGitHubRepositoryIdentity(ctx, 100, repositoryID, "octo/repo", "new-owner/new-repo"); err != nil {
		t.Fatal(err)
	}
	projects, err = repo.ListByGitHubSource(ctx, 100, "new-owner/new-repo")
	if err != nil || len(projects) != 2 {
		t.Fatalf("renamed projects = %d, err=%v", len(projects), err)
	}
	storedDeployment, err := NewDeploymentRepository(db).GetByID(ctx, deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedDeployment.SourceRepository == nil || *storedDeployment.SourceRepository != "new-owner/new-repo" ||
		storedDeployment.SourceInstallationID == nil || *storedDeployment.SourceInstallationID != 100 {
		t.Fatalf("deployment source after rename = %v installation=%v",
			storedDeployment.SourceRepository, storedDeployment.SourceInstallationID)
	}
}

func TestProjectRepository_CountByOwner(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "count@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Project{
		OwnerID: user.ID, Name: "P1", Slug: "p1",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	})

	count, _ := repo.CountByOwner(ctx, user.ID)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestProjectRepository_UpdateBuildMetaPersistsFramework(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUserForProject(t, db, "meta@test.com")
	repo := NewProjectRepository(db)
	ctx := context.Background()
	project := &models.Project{OwnerID: user.ID, Name: "Meta", Slug: "meta", ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20"}
	if err := repo.Create(ctx, project); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdateBuildMeta(ctx, project.ID, "vite", "pnpm", "lock-hash"); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Framework == nil || *got.Framework != "vite" || got.DetectedPackageManager != "pnpm" || got.LockFileHash != "lock-hash" {
		t.Fatalf("unexpected build metadata: %#v", got)
	}
}
