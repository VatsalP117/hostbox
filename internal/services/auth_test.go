package services

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/vatsalpatel/hostbox/internal/config"
	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/dto"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

// migrationsFS contains all SQL migration files for test setup.
var migrationsFS = fstest.MapFS{
	"001_initial.sql": &fstest.MapFile{Data: mustReadFile("../../migrations/001_initial.sql")},
	"002_password_reset.sql": &fstest.MapFile{Data: mustReadFile("../../migrations/002_password_reset.sql")},
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

func setupTestAuth(t *testing.T) (*AuthService, *repository.Repositories, *sql.DB) {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.Migrate(db, migrationsFS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repos := repository.New(db)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		JWTSecret:      "test-jwt-secret-that-is-long-enough",
		AccessTokenTTL: 15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Enable registration and mark setup complete for tests
	_, err = db.Exec(`UPDATE settings SET value = 'true' WHERE key IN ('registration_enabled', 'setup_complete')`)
	if err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	authService := NewAuthService(repos.User, repos.Session, repos.Settings, repos.Activity, cfg, logger)
	return authService, repos, db
}

func TestRegisterAndLogin(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	// Register
	user, accessToken, rawRefresh, err := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "alice@example.com",
		Password: "password123",
	}, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", user.Email)
	}
	if accessToken == "" || rawRefresh == "" {
		t.Error("tokens should not be empty")
	}

	// Validate access token
	claims, err := auth.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.Subject != user.ID {
		t.Errorf("subject = %q, want %q", claims.Subject, user.ID)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", claims.Email)
	}
	if claims.Admin {
		t.Error("should not be admin")
	}

	// Login
	user2, at2, rr2, err := auth.Login(ctx, &dto.LoginRequest{
		Email:    "alice@example.com",
		Password: "password123",
	}, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user2.ID != user.ID {
		t.Error("login should return same user")
	}
	if at2 == "" || rr2 == "" {
		t.Error("login tokens should not be empty")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	_, _, _, err := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "dup@example.com",
		Password: "password123",
	}, "", "")
	if err != nil {
		t.Fatalf("first register: %v", err)
	}

	_, _, _, err = auth.Register(ctx, &dto.RegisterRequest{
		Email:    "dup@example.com",
		Password: "password456",
	}, "", "")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Status != 409 {
		t.Errorf("expected 409, got %v", err)
	}
}

func TestRegisterDisabled(t *testing.T) {
	auth, _, db := setupTestAuth(t)
	ctx := context.Background()

	db.Exec(`UPDATE settings SET value = 'false' WHERE key = 'registration_enabled'`)

	_, _, _, err := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	}, "", "")
	if err == nil {
		t.Fatal("expected error when registration disabled")
	}
	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Status != 403 {
		t.Errorf("expected 403, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	auth.Register(ctx, &dto.RegisterRequest{
		Email:    "bob@example.com",
		Password: "correct-password",
	}, "", "")

	_, _, _, err := auth.Login(ctx, &dto.LoginRequest{
		Email:    "bob@example.com",
		Password: "wrong-password",
	}, "", "")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Status != 401 {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	_, _, _, err := auth.Login(ctx, &dto.LoginRequest{
		Email:    "nobody@example.com",
		Password: "whatever",
	}, "", "")
	if err == nil {
		t.Fatal("expected error for unknown email")
	}
}

func TestRefreshToken(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	_, _, rawRefresh, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "refresh@example.com",
		Password: "password123",
	}, "agent", "1.2.3.4")

	newAccess, err := auth.Refresh(ctx, rawRefresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newAccess == "" {
		t.Error("new access token should not be empty")
	}

	// Validate the new access token
	claims, err := auth.ValidateAccessToken(newAccess)
	if err != nil {
		t.Fatalf("validate refreshed token: %v", err)
	}
	if claims.Email != "refresh@example.com" {
		t.Error("email mismatch after refresh")
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	_, err := auth.Refresh(ctx, "totally-invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

func TestLogout(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, rawRefresh, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "logout@example.com",
		Password: "password123",
	}, "", "")

	err := auth.Logout(ctx, rawRefresh, &user.ID)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}

	// Refresh should now fail
	_, err = auth.Refresh(ctx, rawRefresh)
	if err == nil {
		t.Fatal("expected error after logout")
	}
}

func TestLogoutAll(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, rr1, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "logoutall@example.com",
		Password: "password123",
	}, "", "")

	// Create a second session via login
	_, _, rr2, _ := auth.Login(ctx, &dto.LoginRequest{
		Email:    "logoutall@example.com",
		Password: "password123",
	}, "agent2", "2.2.2.2")

	count, err := auth.LogoutAll(ctx, user.ID)
	if err != nil {
		t.Fatalf("logout all: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2 sessions deleted, got %d", count)
	}

	// Both tokens should be invalid
	_, err = auth.Refresh(ctx, rr1)
	if err == nil {
		t.Error("rr1 should be invalid")
	}
	_, err = auth.Refresh(ctx, rr2)
	if err == nil {
		t.Error("rr2 should be invalid")
	}
}

func TestChangePassword(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "change@example.com",
		Password: "old-password",
	}, "", "")

	err := auth.ChangePassword(ctx, user.ID, &dto.ChangePasswordRequest{
		CurrentPassword: "old-password",
		NewPassword:     "new-password",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	// Old password should fail
	_, _, _, err = auth.Login(ctx, &dto.LoginRequest{
		Email:    "change@example.com",
		Password: "old-password",
	}, "", "")
	if err == nil {
		t.Fatal("expected error with old password")
	}

	// New password should work
	_, _, _, err = auth.Login(ctx, &dto.LoginRequest{
		Email:    "change@example.com",
		Password: "new-password",
	}, "", "")
	if err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "wrongcurrent@example.com",
		Password: "my-password",
	}, "", "")

	err := auth.ChangePassword(ctx, user.ID, &dto.ChangePasswordRequest{
		CurrentPassword: "not-my-password",
		NewPassword:     "new-password",
	})
	if err == nil {
		t.Fatal("expected error for wrong current password")
	}
}

func TestForgotAndResetPassword(t *testing.T) {
	auth, repos, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "reset@example.com",
		Password: "old-password",
	}, "", "")

	// ForgotPassword always returns nil
	err := auth.ForgotPassword(ctx, "reset@example.com")
	if err != nil {
		t.Fatalf("forgot password: %v", err)
	}

	// Get the token hash from DB to construct a valid reset
	updatedUser, _ := repos.User.GetByID(ctx, user.ID)
	if updatedUser.ResetTokenHash == nil {
		t.Fatal("reset token hash should be set")
	}

	// We can't directly get the raw token from the service (it would be emailed),
	// so we test ResetPassword by setting a known token ourselves.
	knownToken := "known-reset-token-value"
	knownHash := hashToken(knownToken)
	repos.User.SetResetToken(ctx, user.ID, knownHash, time.Now().Add(1*time.Hour))

	err = auth.ResetPassword(ctx, &dto.ResetPasswordRequest{
		Token:       knownToken,
		NewPassword: "brand-new-password",
	})
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}

	// Should be able to login with new password
	_, _, _, err = auth.Login(ctx, &dto.LoginRequest{
		Email:    "reset@example.com",
		Password: "brand-new-password",
	}, "", "")
	if err != nil {
		t.Fatalf("login after reset: %v", err)
	}
}

func TestResetPasswordExpired(t *testing.T) {
	auth, repos, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "expired@example.com",
		Password: "password123",
	}, "", "")

	knownToken := "expired-token"
	knownHash := hashToken(knownToken)
	repos.User.SetResetToken(ctx, user.ID, knownHash, time.Now().Add(-1*time.Hour)) // already expired

	err := auth.ResetPassword(ctx, &dto.ResetPasswordRequest{
		Token:       knownToken,
		NewPassword: "new-password",
	})
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestUpdateProfile(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:       "profile@example.com",
		Password:    "password123",
		DisplayName: strPtr("Old Name"),
	}, "", "")

	newName := "New Name"
	updated, err := auth.UpdateProfile(ctx, user.ID, &dto.UpdateProfileRequest{
		DisplayName: &newName,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.DisplayName == nil || *updated.DisplayName != "New Name" {
		t.Error("display name not updated")
	}
}

func TestGetCurrentUser(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "getme@example.com",
		Password: "password123",
	}, "", "")

	found, err := auth.GetCurrentUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("get current user: %v", err)
	}
	if found.Email != "getme@example.com" {
		t.Error("email mismatch")
	}
}

func TestGetCurrentUserNotFound(t *testing.T) {
	auth, _, _ := setupTestAuth(t)
	ctx := context.Background()

	_, err := auth.GetCurrentUser(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	auth, repos, db := setupTestAuth(t)
	ctx := context.Background()

	user, _, _, _ := auth.Register(ctx, &dto.RegisterRequest{
		Email:    "cleanup@example.com",
		Password: "password123",
	}, "", "")

	// Manually insert an expired session
	db.Exec(`INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, created_at)
		VALUES ('expired-sess', ?, 'deadbeef', datetime('now', '-1 day'), datetime('now'))`, user.ID)

	count, err := auth.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 expired session cleaned, got %d", count)
	}

	_ = repos // keep reference
}

func TestJWTValidation(t *testing.T) {
	auth, _, _ := setupTestAuth(t)

	// Invalid token
	_, err := auth.ValidateAccessToken("not.a.real.jwt")
	if err == nil {
		t.Error("expected error for invalid JWT")
	}

	// Expired token (use short TTL)
	auth.config.AccessTokenTTL = -1 * time.Minute
	user, _, _, _ := auth.Register(context.Background(), &dto.RegisterRequest{
		Email:    "expired-jwt@example.com",
		Password: "password123",
	}, "", "")
	token, _ := auth.GenerateAccessToken(user)
	_, err = auth.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired JWT")
	}
}

func strPtr(s string) *string { return &s }
