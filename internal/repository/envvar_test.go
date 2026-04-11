package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
)

func createTestProjectForEnvVar(t *testing.T, db *sql.DB) *models.Project {
	t.Helper()
	ctx := context.Background()
	user := &models.User{Email: "envvar@test.com", PasswordHash: "hash"}
	NewUserRepository(db).Create(ctx, user)
	project := &models.Project{
		OwnerID: user.ID, Name: "EnvVar Test", Slug: "envvar-test",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	NewProjectRepository(db).Create(ctx, project)
	return project
}

func TestEnvVarRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	ev := &models.EnvVar{
		ProjectID:      project.ID,
		Key:            "API_KEY",
		EncryptedValue: "encrypted-data",
		IsSecret:       true,
		Scope:          "all",
	}
	if err := repo.Create(ctx, ev); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, ev.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Key != "API_KEY" {
		t.Errorf("Key = %q, want 'API_KEY'", got.Key)
	}
	if !got.IsSecret {
		t.Error("expected IsSecret = true")
	}
}

func TestEnvVarRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	ev := &models.EnvVar{
		ProjectID: project.ID, Key: "KEY", EncryptedValue: "old", Scope: "all",
	}
	repo.Create(ctx, ev)

	ev.EncryptedValue = "new"
	if err := repo.Update(ctx, ev); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, ev.ID)
	if got.EncryptedValue != "new" {
		t.Errorf("EncryptedValue = %q, want 'new'", got.EncryptedValue)
	}
}

func TestEnvVarRepository_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "A", EncryptedValue: "x", Scope: "all"})
	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "B", EncryptedValue: "y", Scope: "production"})

	envVars, err := repo.ListByProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(envVars) != 2 {
		t.Errorf("len = %d, want 2", len(envVars))
	}
	// Should be ordered by key ASC
	if envVars[0].Key != "A" {
		t.Errorf("first key = %q, want 'A'", envVars[0].Key)
	}
}

func TestEnvVarRepository_ListByProjectAndScope(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "ALL_VAR", EncryptedValue: "x", Scope: "all"})
	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "PROD_VAR", EncryptedValue: "y", Scope: "production"})
	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "PREVIEW_VAR", EncryptedValue: "z", Scope: "preview"})

	// Scope "production" should return "all" + "production"
	envVars, err := repo.ListByProjectAndScope(ctx, project.ID, "production")
	if err != nil {
		t.Fatalf("ListByProjectAndScope: %v", err)
	}
	if len(envVars) != 2 {
		t.Errorf("len = %d, want 2 (all + production)", len(envVars))
	}
}

func TestEnvVarRepository_GetByProjectKeyScope(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "DB_URL", EncryptedValue: "enc", Scope: "all"})

	got, err := repo.GetByProjectKeyScope(ctx, project.ID, "DB_URL", "all")
	if err != nil {
		t.Fatalf("GetByProjectKeyScope: %v", err)
	}
	if got.Key != "DB_URL" {
		t.Errorf("Key = %q", got.Key)
	}
}

func TestEnvVarRepository_Upsert(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	// First insert
	ev := &models.EnvVar{ProjectID: project.ID, Key: "UPSERT_KEY", EncryptedValue: "first", Scope: "all"}
	repo.Upsert(ctx, ev)

	// Upsert with new value
	ev2 := &models.EnvVar{ProjectID: project.ID, Key: "UPSERT_KEY", EncryptedValue: "second", Scope: "all"}
	repo.Upsert(ctx, ev2)

	// Should have only 1 record
	envVars, _ := repo.ListByProject(ctx, project.ID)
	if len(envVars) != 1 {
		t.Errorf("expected 1 env var after upsert, got %d", len(envVars))
	}
	if envVars[0].EncryptedValue != "second" {
		t.Errorf("value = %q, want 'second'", envVars[0].EncryptedValue)
	}
}

func TestEnvVarRepository_UniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "DUP", EncryptedValue: "x", Scope: "all"})
	err := repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "DUP", EncryptedValue: "y", Scope: "all"})
	if err == nil {
		t.Fatal("expected error for duplicate key+scope")
	}

	// But same key with different scope should work
	err = repo.Create(ctx, &models.EnvVar{ProjectID: project.ID, Key: "DUP", EncryptedValue: "z", Scope: "production"})
	if err != nil {
		t.Fatalf("different scope should be allowed: %v", err)
	}
}

func TestEnvVarRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForEnvVar(t, db)
	repo := NewEnvVarRepository(db)
	ctx := context.Background()

	ev := &models.EnvVar{ProjectID: project.ID, Key: "DEL", EncryptedValue: "x", Scope: "all"}
	repo.Create(ctx, ev)
	repo.Delete(ctx, ev.ID)

	_, err := repo.GetByID(ctx, ev.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
