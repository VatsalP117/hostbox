package middleware

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/config"
	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/dto"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/internal/services"
)

var testMigrationsFS = fstest.MapFS{
	"001_initial.sql":        &fstest.MapFile{Data: mustReadFile("../../../migrations/001_initial.sql")},
	"002_password_reset.sql": &fstest.MapFile{Data: mustReadFile("../../../migrations/002_password_reset.sql")},
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

type testAuthEnv struct {
	authService *services.AuthService
	repos       *repository.Repositories
	db          *sql.DB
	e           *echo.Echo
}

func setupAuthTestEnv(t *testing.T) *testAuthEnv {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.Migrate(db, testMigrationsFS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repos := repository.New(db)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		JWTSecret:       "test-jwt-secret-long-enough-for-hs256",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Enable registration and setup
	db.Exec(`UPDATE settings SET value = 'true' WHERE key IN ('registration_enabled', 'setup_complete')`)

	authSvc := services.NewAuthService(repos.User, repos.Session, repos.Settings, repos.Activity, cfg, logger)

	return &testAuthEnv{
		authService: authSvc,
		repos:       repos,
		db:          db,
		e:           echo.New(),
	}
}

func (env *testAuthEnv) createUser(t *testing.T, email, password string, admin bool) (*models.User, string) {
	t.Helper()
	user, accessToken, _, err := env.authService.Register(context.Background(), &dto.RegisterRequest{
		Email:    email,
		Password: password,
	}, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if admin {
		env.db.Exec(`UPDATE users SET is_admin = 1 WHERE id = ?`, user.ID)
		user.IsAdmin = true
		// Re-generate token with admin claim
		accessToken, _ = env.authService.GenerateAccessToken(user)
	}
	return user, accessToken
}

// --- JWTAuth Tests ---

func TestJWTAuth_ValidToken(t *testing.T) {
	env := setupAuthTestEnv(t)
	_, accessToken := env.createUser(t, "jwt@example.com", "password123", false)

	handler := JWTAuth(env.authService)(func(c echo.Context) error {
		user := GetUser(c)
		if user == nil {
			return echo.NewHTTPError(500, "no user in context")
		}
		return c.JSON(200, map[string]string{"email": user.Email})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)

	err := handler(c)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	env := setupAuthTestEnv(t)
	handler := JWTAuth(env.authService)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)

	err := handler(c)
	if err == nil {
		t.Fatal("expected error for missing auth header")
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	env := setupAuthTestEnv(t)
	handler := JWTAuth(env.authService)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)

	err := handler(c)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

// --- RequireAdmin Tests ---

func TestRequireAdmin_AdminUser(t *testing.T) {
	env := setupAuthTestEnv(t)
	user, _ := env.createUser(t, "admin@example.com", "password123", true)

	handler := RequireAdmin()(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)
	c.Set(string(UserContextKey), user)

	err := handler(c)
	if err != nil {
		t.Fatalf("expected no error for admin, got: %v", err)
	}
}

func TestRequireAdmin_NonAdmin(t *testing.T) {
	env := setupAuthTestEnv(t)
	user, _ := env.createUser(t, "regular@example.com", "password123", false)

	handler := RequireAdmin()(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)
	c.Set(string(UserContextKey), user)

	err := handler(c)
	if err == nil {
		t.Fatal("expected error for non-admin")
	}
}

func TestRequireAdmin_NoUser(t *testing.T) {
	env := setupAuthTestEnv(t)
	handler := RequireAdmin()(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)

	err := handler(c)
	if err == nil {
		t.Fatal("expected error when no user in context")
	}
}

// --- RequireSetupComplete Tests ---

func TestRequireSetupComplete_Complete(t *testing.T) {
	env := setupAuthTestEnv(t)
	// setup_complete is already 'true'

	handler := RequireSetupComplete(env.repos.Settings)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)

	err := handler(c)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRequireSetupComplete_NotComplete(t *testing.T) {
	env := setupAuthTestEnv(t)
	env.db.Exec(`UPDATE settings SET value = 'false' WHERE key = 'setup_complete'`)

	handler := RequireSetupComplete(env.repos.Settings)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := env.e.NewContext(req, rec)

	err := handler(c)
	if err == nil {
		t.Fatal("expected error when setup not complete")
	}
}

// --- GetUser Tests ---

func TestGetUser_Present(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := &models.User{ID: "test-id", Email: "test@example.com"}
	c.Set(string(UserContextKey), user)

	got := GetUser(c)
	if got == nil || got.ID != "test-id" {
		t.Error("expected user from context")
	}
}

func TestGetUser_Missing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	got := GetUser(c)
	if got != nil {
		t.Error("expected nil when no user set")
	}
}
