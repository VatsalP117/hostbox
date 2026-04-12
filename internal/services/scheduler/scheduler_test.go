package scheduler

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/logger"
	"github.com/vatsalpatel/hostbox/migrations"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSessionCleaner_CleansExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	l := logger.Setup("error", "text")

	// Create a user first
	_, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, display_name, is_admin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"user-1", "test@test.com", "hash", "Test", true, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert expired session
	expired := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	_, err = db.Exec(
		`INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		"sess-1", "user-1", "token1", expired, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert active session
	active := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	_, err = db.Exec(
		`INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		"sess-2", "user-1", "token2", active, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatal(err)
	}

	cleaner := NewSessionCleaner(db, l)
	cleaner.Clean(context.Background())

	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 session remaining, got %d", count)
	}
}

func TestGarbageCollector_CleansOldArtifacts(t *testing.T) {
	db := setupTestDB(t)
	l := logger.Setup("error", "text")

	repos := repository.New(db)

	// Create user + project
	_, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, display_name, is_admin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"user-1", "test@test.com", "hash", "Test", true, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	project := &models.Project{
		ID: "proj-1", OwnerID: "user-1", Name: "Test", Slug: "test",
		ProductionBranch: "main", AutoDeploy: true, PreviewDeployments: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Project.Create(context.Background(), project); err != nil {
		t.Fatal(err)
	}

	// Set low retention limits for testing
	repos.Settings.Set(context.Background(), "max_deployments_per_project", "5")

	// Create temp artifact dirs (15 deployments to exceed 2x5=10 retention)
	artifactDir := t.TempDir()

	for i := 0; i < 15; i++ {
		artPath := filepath.Join(artifactDir, "deploy-"+itoa(i))
		os.MkdirAll(artPath, 0o755)
		os.WriteFile(filepath.Join(artPath, "index.html"), []byte("test"), 0o644)

		createdAt := now.Add(-time.Duration(i) * 24 * time.Hour).Format(time.RFC3339)
		isProd := 0
		if i == 0 {
			isProd = 1
		}
		_, err := db.Exec(
			`INSERT INTO deployments (id, project_id, commit_sha, branch, status, is_production, artifact_path, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"deploy-"+itoa(i), "proj-1", "sha"+itoa(i), "main", "ready", isProd, artPath, createdAt,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	gc := NewGarbageCollector(repos.Deployment, repos.Project, repos.Settings, "", l)
	gc.Collect(context.Background())

	// Production deployment (deploy-0) should be protected
	d0, _ := repos.Deployment.GetByID(context.Background(), "deploy-0")
	if d0.ArtifactPath == nil || *d0.ArtifactPath == "" {
		t.Error("production deployment artifact should be protected")
	}

	// Deployment beyond 2x max (deploy-11 = 11 days old, kept > 10) should be cleaned
	d11, _ := repos.Deployment.GetByID(context.Background(), "deploy-11")
	if d11.ArtifactPath != nil && *d11.ArtifactPath != "" {
		t.Error("deployment beyond 2x max should have been cleared")
	}
}

func TestGarbageCollector_CleansOrphanedLogs(t *testing.T) {
	db := setupTestDB(t)
	l := logger.Setup("error", "text")
	repos := repository.New(db)

	logDir := t.TempDir()
	// Create an orphaned log file (no matching deployment)
	os.WriteFile(filepath.Join(logDir, "orphan-deploy.log"), []byte("logs"), 0o644)

	gc := NewGarbageCollector(repos.Deployment, repos.Project, repos.Settings, logDir, l)
	gc.Collect(context.Background())

	if _, err := os.Stat(filepath.Join(logDir, "orphan-deploy.log")); !os.IsNotExist(err) {
		t.Error("orphaned log file should have been removed")
	}
}

func TestDomainReVerifier_MarksExpiredDomainsUnverified(t *testing.T) {
	db := setupTestDB(t)
	l := logger.Setup("error", "text")
	repos := repository.New(db)

	// Create user + project
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, display_name, is_admin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"user-1", "test@test.com", "hash", "Test", true, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatal(err)
	}

	project := &models.Project{
		ID: "proj-1", OwnerID: "user-1", Name: "Test", Slug: "test",
		ProductionBranch: "main", AutoDeploy: true, PreviewDeployments: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Project.Create(context.Background(), project); err != nil {
		t.Fatal(err)
	}

	// Create a domain verified long ago (beyond grace period) with a non-resolvable domain
	oldVerified := now.Add(-30 * 24 * time.Hour)
	domain := &models.Domain{
		ID:        "dom-1",
		ProjectID: "proj-1",
		Domain:    "this-domain-does-not-exist-hostbox-test.invalid",
		Verified:  true,
		VerifiedAt: &oldVerified,
		CreatedAt: now,
	}
	if err := repos.Domain.Create(context.Background(), domain); err != nil {
		t.Fatal(err)
	}

	rv := NewDomainReVerifier(repos.Domain, l)
	rv.ReVerify(context.Background())

	// Domain should be marked unverified (grace period expired + DNS fails)
	d, _ := repos.Domain.GetByID(context.Background(), "dom-1")
	if d.Verified {
		t.Error("domain should have been marked unverified after grace period + DNS failure")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
