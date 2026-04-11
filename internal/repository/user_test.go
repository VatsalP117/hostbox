package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
)

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hashed",
		IsAdmin:      false,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == "" {
		t.Fatal("expected ID to be set")
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("email = %q, want %q", got.Email, "test@example.com")
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{Email: "find@example.com", PasswordHash: "hash"}
	repo.Create(ctx, user)

	got, err := repo.GetByEmail(ctx, "find@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("ID = %q, want %q", got.ID, user.ID)
	}
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nobody@example.com")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{Email: "update@example.com", PasswordHash: "hash"}
	repo.Create(ctx, user)

	name := "Updated Name"
	user.DisplayName = &name
	user.IsAdmin = true
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, user.ID)
	if got.DisplayName == nil || *got.DisplayName != "Updated Name" {
		t.Errorf("display_name = %v, want 'Updated Name'", got.DisplayName)
	}
	if !got.IsAdmin {
		t.Error("expected IsAdmin = true")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{Email: "delete@example.com", PasswordHash: "hash"}
	repo.Create(ctx, user)

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(ctx, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUserRepository_ListAndCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.Create(ctx, &models.User{
			Email:        "user" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "hash",
		})
	}

	users, total, err := repo.List(ctx, 1, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(users) != 3 {
		t.Errorf("len(users) = %d, want 3", len(users))
	}

	count, _ := repo.Count(ctx)
	if count != 5 {
		t.Errorf("Count = %d, want 5", count)
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{Email: "pw@example.com", PasswordHash: "oldhash"}
	repo.Create(ctx, user)

	if err := repo.UpdatePassword(ctx, user.ID, "newhash"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	got, _ := repo.GetByID(ctx, user.ID)
	if got.PasswordHash != "newhash" {
		t.Errorf("password_hash = %q, want 'newhash'", got.PasswordHash)
	}
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.User{Email: "dup@example.com", PasswordHash: "hash"})
	err := repo.Create(ctx, &models.User{Email: "dup@example.com", PasswordHash: "hash2"})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}
