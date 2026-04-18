package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/dto"
	"github.com/VatsalP117/hostbox/migrations"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestHealthHandler_Health(t *testing.T) {
	db := setupTestDB(t)
	startTime := time.Now().Add(-10 * time.Second)
	handler := NewHealthHandler(startTime, db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Health(c); err != nil {
		t.Fatalf("Health: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var resp dto.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("status = %q, want 'ok'", resp.Status)
	}
	if resp.UptimeSeconds < 10 {
		t.Errorf("uptime = %d, expected >= 10", resp.UptimeSeconds)
	}
	if resp.Version == "" {
		t.Error("version should not be empty")
	}
}

func TestHealthHandler_DegradedWhenDBClosed(t *testing.T) {
	db := setupTestDB(t)
	startTime := time.Now()
	handler := NewHealthHandler(startTime, db)

	// Close DB to simulate degraded state
	db.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler.Health(c)

	var resp dto.HealthResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Status != "degraded" {
		t.Errorf("status = %q, want 'degraded'", resp.Status)
	}
}
