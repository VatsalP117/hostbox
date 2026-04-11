package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
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
