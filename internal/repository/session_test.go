package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

func createTestUser(t *testing.T, db *sql.DB) *models.User {
	t.Helper()
	repo := NewUserRepository(db)
	user := &models.User{Email: "session-test@example.com", PasswordHash: "hash"}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user
}

func TestSessionRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "tokenhash",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	if err := repo.Create(ctx, session); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected ID to be set")
	}

	got, err := repo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", got.UserID, user.ID)
	}
}

func TestSessionRepository_DeleteByID(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "hash",
		ExpiresAt:        time.Now().Add(time.Hour),
	}
	repo.Create(ctx, session)

	if err := repo.DeleteByID(ctx, session.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	_, err := repo.GetByID(ctx, session.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSessionRepository_DeleteByUserID(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		repo.Create(ctx, &models.Session{
			UserID:           user.ID,
			RefreshTokenHash: "hash",
			ExpiresAt:        time.Now().Add(time.Hour),
		})
	}

	count, err := repo.DeleteByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteByUserID: %v", err)
	}
	if count != 3 {
		t.Errorf("deleted = %d, want 3", count)
	}
}

func TestSessionRepository_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create expired session
	repo.Create(ctx, &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "expired",
		ExpiresAt:        time.Now().Add(-time.Hour),
	})
	// Create valid session
	repo.Create(ctx, &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "valid",
		ExpiresAt:        time.Now().Add(time.Hour),
	})

	count, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if count != 1 {
		t.Errorf("deleted = %d, want 1", count)
	}

	sessions, _ := repo.ListByUserID(ctx, user.ID)
	if len(sessions) != 1 {
		t.Errorf("remaining = %d, want 1", len(sessions))
	}
}

func TestSessionRepository_ListByUserID(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		repo.Create(ctx, &models.Session{
			UserID:           user.ID,
			RefreshTokenHash: "hash",
			ExpiresAt:        time.Now().Add(time.Hour),
		})
	}

	sessions, err := repo.ListByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByUserID: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("len = %d, want 2", len(sessions))
	}
}

func TestSessionRepository_CascadeDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: "hash",
		ExpiresAt:        time.Now().Add(time.Hour),
	})

	// Delete the user → sessions should cascade delete
	NewUserRepository(db).Delete(ctx, user.ID)

	sessions, _ := repo.ListByUserID(ctx, user.ID)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after user delete, got %d", len(sessions))
	}
}
