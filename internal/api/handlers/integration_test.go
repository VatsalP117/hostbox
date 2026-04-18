package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/VatsalP117/hostbox/internal/api/handlers"
	appmiddleware "github.com/VatsalP117/hostbox/internal/api/middleware"
	"github.com/VatsalP117/hostbox/internal/api/routes"
	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/dto"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/services"
	"github.com/VatsalP117/hostbox/migrations"
)

// testEnv holds a fully wired test environment.
type testEnv struct {
	echo        *echo.Echo
	db          *sql.DB
	repos       *repository.Repositories
	authService *services.AuthService
	cfg         *config.Config
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repos := repository.New(db)

	cfg := &config.Config{
		DatabasePath:    dbPath,
		JWTSecret:       "test-jwt-secret-that-is-long-enough-32chars!",
		EncryptionKey:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		PlatformDomain:  "test.example.com",
		DashboardDomain: "hostbox.test.example.com",
		PlatformHTTPS:   false,
		DeploymentsDir:  dir + "/deployments",
		LogsDir:         dir + "/logs",
	}
	os.MkdirAll(cfg.DeploymentsDir, 0o755)
	os.MkdirAll(cfg.LogsDir, 0o755)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	authService := services.NewAuthService(
		repos.User, repos.Session, repos.Settings, repos.Activity,
		cfg, logger,
	)

	e := echo.New()
	e.HideBanner = true
	e.Validator = &echoValidator{}

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		switch appErr := err.(type) {
		case *apperrors.AppError:
			c.JSON(appErr.Status, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    appErr.Code,
					"message": appErr.Message,
					"details": appErr.Details,
				},
			})
		case *echo.HTTPError:
			msg := "An error occurred"
			if m, ok := appErr.Message.(string); ok {
				msg = m
			}
			c.JSON(appErr.Code, map[string]interface{}{"error": msg})
		default:
			c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		}
	}

	startTime := time.Now()
	healthHandler := handlers.NewHealthHandler(startTime, db)
	setupHandler := handlers.NewSetupHandler(authService, repos.User, repos.Settings, repos.Activity, false, logger)
	authHandler := handlers.NewAuthHandler(authService, false, logger)
	projectHandler := handlers.NewProjectHandler(
		repos.Project,
		repos.Deployment,
		repos.Domain,
		repos.Activity,
		cfg.PlatformDomain,
		cfg.DashboardDomain,
		logger,
	)
	deploymentHandler := handlers.NewDeploymentHandler(repos.Deployment, repos.Project, repos.Activity, logger)
	domainHandler := handlers.NewDomainHandler(repos.Domain, repos.Project, repos.Activity, "test.example.com", logger)
	envVarHandler := handlers.NewEnvVarHandler(repos.EnvVar, repos.Project, repos.Activity, cfg, logger)
	adminHandler := handlers.NewAdminHandler(repos.User, repos.Project, repos.Deployment, repos.Activity, repos.Settings, cfg, logger)

	routes.Register(e, routes.Deps{
		AuthService:       authService,
		SettingsRepo:      repos.Settings,
		HealthHandler:     healthHandler,
		SetupHandler:      setupHandler,
		AuthHandler:       authHandler,
		ProjectHandler:    projectHandler,
		DeploymentHandler: deploymentHandler,
		DomainHandler:     domainHandler,
		EnvVarHandler:     envVarHandler,
		AdminHandler:      adminHandler,
	})

	return &testEnv{
		echo:        e,
		db:          db,
		repos:       repos,
		authService: authService,
		cfg:         cfg,
	}
}

type echoValidator struct{}

func (ev *echoValidator) Validate(i interface{}) error {
	return dto.ValidateStruct(i)
}

func jsonBody(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func doRequest(e *echo.Echo, method, path string, body *bytes.Buffer, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode response: %v, body: %s", err, rec.Body.String())
	}
}

// --- Setup & Auth Flow Tests ---

func TestSetupStatus_NotComplete(t *testing.T) {
	env := setupTestEnv(t)
	rec := doRequest(env.echo, http.MethodGet, "/api/v1/setup/status", nil, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp dto.SetupStatusResponse
	mustDecode(t, rec, &resp)
	if !resp.SetupRequired {
		t.Error("expected setup_required=true before setup")
	}
}

func TestSetup_CreatesAdminUser(t *testing.T) {
	env := setupTestEnv(t)

	body := jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "supersecret123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/setup", body, nil)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AuthResponse
	mustDecode(t, rec, &resp)
	if resp.User.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", resp.User.Email)
	}
	if !resp.User.IsAdmin {
		t.Error("user should be admin")
	}
	if resp.AccessToken == "" {
		t.Error("access token should not be empty")
	}
}

func TestSetup_CreatesRefreshSession(t *testing.T) {
	env := setupTestEnv(t)

	body := jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "supersecret123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/setup", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d %s", rec.Code, rec.Body.String())
	}

	var refreshCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "hostbox_refresh" {
			refreshCookie = cookie
			break
		}
	}
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatal("expected refresh cookie from setup response")
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	env.echo.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh after setup: %d %s", refreshRec.Code, refreshRec.Body.String())
	}

	var tokenResp dto.TokenResponse
	mustDecode(t, refreshRec, &tokenResp)
	if tokenResp.AccessToken == "" {
		t.Fatal("expected new access token from refresh")
	}
}

func TestSetup_SecondTimeFails(t *testing.T) {
	env := setupTestEnv(t)

	body := jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "supersecret123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/setup", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first setup failed: %d %s", rec.Code, rec.Body.String())
	}

	// Try again
	body = jsonBody(map[string]string{
		"email":    "admin2@test.com",
		"password": "supersecret123",
	})
	rec = doRequest(env.echo, http.MethodPost, "/api/v1/setup", body, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("second setup: status = %d, want 403, body: %s", rec.Code, rec.Body.String())
	}
}

func setupAdmin(t *testing.T, env *testEnv) (accessToken string) {
	t.Helper()
	body := jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "adminpass123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/setup", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d %s", rec.Code, rec.Body.String())
	}
	var resp dto.AuthResponse
	mustDecode(t, rec, &resp)
	return resp.AccessToken
}

func TestLogin_Success(t *testing.T) {
	env := setupTestEnv(t)
	_ = setupAdmin(t, env)

	body := jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "adminpass123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/auth/login", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AuthResponse
	mustDecode(t, rec, &resp)
	if resp.AccessToken == "" {
		t.Error("login should return access token")
	}
	if resp.User.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", resp.User.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	env := setupTestEnv(t)
	_ = setupAdmin(t, env)

	body := jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "wrongpassword",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/auth/login", body, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMe_Authenticated(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)

	rec := doRequest(env.echo, http.MethodGet, "/api/v1/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("me: status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var meResp struct {
		User dto.UserResponse `json:"user"`
	}
	mustDecode(t, rec, &meResp)
	if meResp.User.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", meResp.User.Email)
	}
}

func TestMe_Unauthenticated(t *testing.T) {
	env := setupTestEnv(t)
	rec := doRequest(env.echo, http.MethodGet, "/api/v1/auth/me", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// --- Project CRUD Tests ---

func TestProjectCRUD(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create
	body := jsonBody(map[string]string{"name": "My First Project"})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/projects", body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project: %d %s", rec.Code, rec.Body.String())
	}

	var createWrapper struct {
		Project dto.ProjectResponse `json:"project"`
	}
	mustDecode(t, rec, &createWrapper)
	createResp := createWrapper.Project
	if createResp.Name != "My First Project" {
		t.Errorf("name = %q", createResp.Name)
	}
	if createResp.Slug == "" {
		t.Error("slug should not be empty")
	}
	projectID := createResp.ID

	// List
	rec = doRequest(env.echo, http.MethodGet, "/api/v1/projects", nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("list projects: %d %s", rec.Code, rec.Body.String())
	}
	var listResp dto.ProjectListResponse
	mustDecode(t, rec, &listResp)
	if len(listResp.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(listResp.Projects))
	}

	// Get
	rec = doRequest(env.echo, http.MethodGet, "/api/v1/projects/"+projectID, nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("get project: %d %s", rec.Code, rec.Body.String())
	}

	// Update
	body = jsonBody(map[string]string{"name": "Updated Project"})
	rec = doRequest(env.echo, http.MethodPatch, "/api/v1/projects/"+projectID, body, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("update project: %d %s", rec.Code, rec.Body.String())
	}
	var updateWrapper struct {
		Project dto.ProjectResponse `json:"project"`
	}
	mustDecode(t, rec, &updateWrapper)
	if updateWrapper.Project.Name != "Updated Project" {
		t.Errorf("updated name = %q, want 'Updated Project'", updateWrapper.Project.Name)
	}

	// Delete
	rec = doRequest(env.echo, http.MethodDelete, "/api/v1/projects/"+projectID, nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete project: %d %s", rec.Code, rec.Body.String())
	}

	// Verify deleted
	rec = doRequest(env.echo, http.MethodGet, "/api/v1/projects/"+projectID, nil, headers)
	if rec.Code != http.StatusNotFound {
		t.Errorf("get deleted: status = %d, want 404", rec.Code)
	}
}

func TestProjectCreate_RejectsDashboardSlug(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	body := jsonBody(map[string]string{"name": "Hostbox"})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/projects", body, headers)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectCreate_NormalizesLongSlug(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	body := jsonBody(map[string]string{
		"name": "This Project Name Is Wildly Longer Than A Safe DNS Label And Should Still Produce A Stable Slug",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/projects", body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body: %s", rec.Code, rec.Body.String())
	}

	var wrapper struct {
		Project dto.ProjectResponse `json:"project"`
	}
	mustDecode(t, rec, &wrapper)
	if len(wrapper.Project.Slug) > 54 {
		t.Fatalf("slug length = %d, want <= 54", len(wrapper.Project.Slug))
	}
}

// --- Deployment Tests ---

func createTestProject(t *testing.T, env *testEnv, token, name string) string {
	t.Helper()
	headers := map[string]string{"Authorization": "Bearer " + token}
	body := jsonBody(map[string]string{"name": name})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/projects", body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project %q: %d %s", name, rec.Code, rec.Body.String())
	}
	var wrapper struct {
		Project dto.ProjectResponse `json:"project"`
	}
	mustDecode(t, rec, &wrapper)
	return wrapper.Project.ID
}

func TestDeploymentCreateAndList(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}
	projectID := createTestProject(t, env, token, "Deploy Test")

	// Create deployment
	body := jsonBody(map[string]interface{}{
		"commit_sha": "abc123def456",
		"branch":     "main",
	})
	rec := doRequest(env.echo, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/deployments", projectID), body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create deployment: %d %s", rec.Code, rec.Body.String())
	}

	var deployWrapper struct {
		Deployment dto.DeploymentResponse `json:"deployment"`
	}
	mustDecode(t, rec, &deployWrapper)
	deployResp := deployWrapper.Deployment
	if deployResp.CommitSHA != "abc123def456" {
		t.Errorf("commit_sha = %q", deployResp.CommitSHA)
	}
	if deployResp.Status != "queued" {
		t.Errorf("status = %q, want 'queued'", deployResp.Status)
	}
	deployID := deployResp.ID

	// List by project
	rec = doRequest(env.echo, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/deployments", projectID), nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("list deployments: %d %s", rec.Code, rec.Body.String())
	}
	var listResp dto.DeploymentListResponse
	mustDecode(t, rec, &listResp)
	if len(listResp.Deployments) != 1 {
		t.Errorf("expected 1 deployment, got %d", len(listResp.Deployments))
	}

	// Get single
	rec = doRequest(env.echo, http.MethodGet, "/api/v1/deployments/"+deployID, nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("get deployment: %d %s", rec.Code, rec.Body.String())
	}
}

// --- Domain Tests ---

func TestDomainCreateAndDelete(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}
	projectID := createTestProject(t, env, token, "Domain Test")

	// Create domain
	body := jsonBody(map[string]string{"domain": "myapp.example.com"})
	rec := doRequest(env.echo, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/domains", projectID), body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create domain: %d %s", rec.Code, rec.Body.String())
	}

	var domainResp dto.CreateDomainResponse
	mustDecode(t, rec, &domainResp)
	if domainResp.Domain.Domain != "myapp.example.com" {
		t.Errorf("domain = %q", domainResp.Domain.Domain)
	}
	domainID := domainResp.Domain.ID

	// List by project
	rec = doRequest(env.echo, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/domains", projectID), nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("list domains: %d %s", rec.Code, rec.Body.String())
	}

	// Delete
	rec = doRequest(env.echo, http.MethodDelete, "/api/v1/domains/"+domainID, nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete domain: %d %s", rec.Code, rec.Body.String())
	}
}

// --- Env Var Tests ---

func TestEnvVarCRUD(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}
	projectID := createTestProject(t, env, token, "EnvVar Test")

	// Create
	body := jsonBody(map[string]interface{}{
		"key":   "API_KEY",
		"value": "super-secret-key",
	})
	rec := doRequest(env.echo, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/env-vars", projectID), body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create env var: %d %s", rec.Code, rec.Body.String())
	}

	var envWrapper struct {
		EnvVar dto.EnvVarResponse `json:"env_var"`
	}
	mustDecode(t, rec, &envWrapper)
	envResp := envWrapper.EnvVar
	if envResp.Key != "API_KEY" {
		t.Errorf("key = %q", envResp.Key)
	}
	// Non-secret value should be decrypted and returned
	if envResp.Value != "super-secret-key" {
		t.Errorf("non-secret value = %q, want 'super-secret-key'", envResp.Value)
	}
	envVarID := envResp.ID

	// List
	rec = doRequest(env.echo, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/env-vars", projectID), nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("list env vars: %d %s", rec.Code, rec.Body.String())
	}

	// Update
	body = jsonBody(map[string]interface{}{
		"value": "new-secret-value",
	})
	rec = doRequest(env.echo, http.MethodPatch, "/api/v1/env-vars/"+envVarID, body, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("update env var: %d %s", rec.Code, rec.Body.String())
	}

	// Delete
	rec = doRequest(env.echo, http.MethodDelete, "/api/v1/env-vars/"+envVarID, nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete env var: %d %s", rec.Code, rec.Body.String())
	}
}

func TestEnvVarSecret_IsMasked(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}
	projectID := createTestProject(t, env, token, "Secret EnvVar Test")

	isSecret := true
	body := jsonBody(map[string]interface{}{
		"key":       "SECRET_KEY",
		"value":     "hidden-value",
		"is_secret": isSecret,
	})
	rec := doRequest(env.echo, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/env-vars", projectID), body, headers)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create secret env var: %d %s", rec.Code, rec.Body.String())
	}

	var wrapper struct {
		EnvVar dto.EnvVarResponse `json:"env_var"`
	}
	mustDecode(t, rec, &wrapper)
	if wrapper.EnvVar.Value == "hidden-value" {
		t.Error("secret env var value should be masked")
	}
	if !wrapper.EnvVar.IsSecret {
		t.Error("env var should be marked as secret")
	}
}

func TestEnvVarBulkCreate(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}
	projectID := createTestProject(t, env, token, "Bulk EnvVar Test")

	body := jsonBody(map[string]interface{}{
		"env_vars": []map[string]string{
			{"key": "DB_HOST", "value": "localhost"},
			{"key": "DB_PORT", "value": "5432"},
			{"key": "DB_NAME", "value": "mydb"},
		},
	})
	rec := doRequest(env.echo, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/env-vars/bulk", projectID), body, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("bulk create: %d %s", rec.Code, rec.Body.String())
	}

	var resp dto.BulkCreateEnvVarResponse
	mustDecode(t, rec, &resp)
	if len(resp.EnvVars) != 3 {
		t.Errorf("expected 3 env vars, got %d", len(resp.EnvVars))
	}
}

// --- Admin Tests ---

func TestAdminStats(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	rec := doRequest(env.echo, http.MethodGet, "/api/v1/admin/stats", nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin stats: %d %s", rec.Code, rec.Body.String())
	}

	var resp dto.AdminStatsResponse
	mustDecode(t, rec, &resp)
	if resp.UserCount != 1 {
		t.Errorf("user count = %d, want 1", resp.UserCount)
	}
}

func TestAdminUsers(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	rec := doRequest(env.echo, http.MethodGet, "/api/v1/admin/users", nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin users: %d %s", rec.Code, rec.Body.String())
	}

	var resp dto.UserListResponse
	mustDecode(t, rec, &resp)
	if len(resp.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(resp.Users))
	}
}

func TestAdminSettings_GetAndUpdate(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	// Get settings
	rec := doRequest(env.echo, http.MethodGet, "/api/v1/admin/settings", nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("get settings: %d %s", rec.Code, rec.Body.String())
	}

	// Update settings
	enabled := true
	body := jsonBody(dto.UpdateSettingsRequest{RegistrationEnabled: &enabled})
	rec = doRequest(env.echo, http.MethodPut, "/api/v1/admin/settings", body, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("update settings: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAdminActivity(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	// Create a project first so there's activity
	createTestProject(t, env, token, "Activity Test")

	rec := doRequest(env.echo, http.MethodGet, "/api/v1/admin/activity", nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin activity: %d %s", rec.Code, rec.Body.String())
	}
}

// --- Authorization Tests ---

func TestNonAdminCannotAccessAdmin(t *testing.T) {
	env := setupTestEnv(t)
	_ = setupAdmin(t, env) // Complete setup

	// Enable registration
	env.repos.Settings.Set(context.Background(), "registration_enabled", "true")

	// Register a normal user
	body := jsonBody(map[string]string{
		"email":    "user@test.com",
		"password": "userpass123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/auth/register", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	var resp dto.AuthResponse
	mustDecode(t, rec, &resp)

	headers := map[string]string{"Authorization": "Bearer " + resp.AccessToken}
	rec = doRequest(env.echo, http.MethodGet, "/api/v1/admin/stats", nil, headers)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-admin accessing admin: status = %d, want 403", rec.Code)
	}
}

func TestRequireSetupComplete_BlocksAuthBeforeSetup(t *testing.T) {
	env := setupTestEnv(t)

	body := jsonBody(map[string]string{
		"email":    "test@test.com",
		"password": "password123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/auth/register", body, nil)
	// Should return 503 (setup required)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("register before setup: status = %d, want 503, body: %s", rec.Code, rec.Body.String())
	}
}

// --- Logout Tests ---

func TestLogout(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	rec := doRequest(env.echo, http.MethodPost, "/api/v1/auth/logout", nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body.String())
	}
}

// --- ChangePassword Tests ---

func TestChangePassword(t *testing.T) {
	env := setupTestEnv(t)
	token := setupAdmin(t, env)
	headers := map[string]string{"Authorization": "Bearer " + token}

	body := jsonBody(map[string]interface{}{
		"current_password": "adminpass123",
		"new_password":     "newadminpass456",
	})
	rec := doRequest(env.echo, http.MethodPut, "/api/v1/auth/me/password", body, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}

	// Login with new password
	body = jsonBody(map[string]string{
		"email":    "admin@test.com",
		"password": "newadminpass456",
	})
	rec = doRequest(env.echo, http.MethodPost, "/api/v1/auth/login", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login with new password: %d %s", rec.Code, rec.Body.String())
	}
}

// --- Health endpoint ---

func TestHealth_Integration(t *testing.T) {
	env := setupTestEnv(t)
	rec := doRequest(env.echo, http.MethodGet, "/api/v1/health", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health: %d %s", rec.Code, rec.Body.String())
	}

	var resp dto.HealthResponse
	mustDecode(t, rec, &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want 'ok'", resp.Status)
	}
}

// --- Ownership Tests ---

func TestProjectOwnership_OtherUserCantAccess(t *testing.T) {
	env := setupTestEnv(t)
	adminToken := setupAdmin(t, env)

	// Enable registration and create normal user
	env.repos.Settings.Set(context.Background(), "registration_enabled", "true")
	body := jsonBody(map[string]string{
		"email":    "user@test.com",
		"password": "userpass123",
	})
	rec := doRequest(env.echo, http.MethodPost, "/api/v1/auth/register", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	var userResp dto.AuthResponse
	mustDecode(t, rec, &userResp)
	userToken := userResp.AccessToken

	// Admin creates a project
	projectID := createTestProject(t, env, adminToken, "Admin Project")

	// Normal user tries to access it
	userHeaders := map[string]string{"Authorization": "Bearer " + userToken}
	rec = doRequest(env.echo, http.MethodGet, "/api/v1/projects/"+projectID, nil, userHeaders)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-owner access: status = %d, want 403", rec.Code)
	}
}

// --- Unused import guard ---
var _ = appmiddleware.GetUser
