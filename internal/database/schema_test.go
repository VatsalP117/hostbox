package database

import (
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/VatsalP117/hostbox/migrations"
)

func TestRealMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, migrations.FS); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Verify all tables exist
	tables := []string{"users", "sessions", "projects", "deployments", "domains", "env_vars", "notification_configs", "activity_log", "settings", "github_webhook_deliveries"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s not created: %v", table, err)
		}
	}

	// Verify default settings
	var settingsCount int
	db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&settingsCount)
	if settingsCount != 6 {
		t.Errorf("settings count = %d, want 6", settingsCount)
	}

	var setupComplete string
	db.QueryRow("SELECT value FROM settings WHERE key = 'setup_complete'").Scan(&setupComplete)
	if setupComplete != "false" {
		t.Errorf("setup_complete = %q, want %q", setupComplete, "false")
	}
}

func TestDeploymentManifestMigrationBackfillsQueuedProjectSettings(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "pre-manifest.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	oldMigrations := fstest.MapFS{}
	for _, name := range []string{
		"001_initial.sql", "002_password_reset.sql", "003_deployments_cache.sql",
		"004_github_deploy_id.sql", "005_system_metrics.sql", "006_github_webhook_deliveries.sql",
	} {
		data, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		oldMigrations[name] = &fstest.MapFile{Data: data}
	}
	if err := Migrate(db, oldMigrations); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash) VALUES ('user', 'user@example.com', 'hash');
		INSERT INTO projects (
			id, owner_id, name, slug, github_repo, github_installation_id,
			production_branch, framework, build_command,
			install_command, output_directory, root_directory, node_version
		) VALUES (
			'project', 'user', 'Project', 'project', 'owner/repo', 42,
			'main', 'vite', 'npm run build',
			'npm ci', 'dist', 'apps/web', '22'
		);
		INSERT INTO deployments (id, project_id, commit_sha, branch, status)
		VALUES ('deployment', 'project', 'manual', 'main', 'queued');
	`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}

	var framework, node, root, output, install, build string
	var sourceRepository string
	var sourceInstallationID int64
	var resolved bool
	if err := db.QueryRow(`
		SELECT build_framework, build_node_version, build_root_directory,
		       build_output_directory, build_install_command, build_command,
		       build_manifest_resolved, source_repository, source_installation_id
		FROM deployments WHERE id = 'deployment'
	`).Scan(
		&framework, &node, &root, &output, &install, &build, &resolved,
		&sourceRepository, &sourceInstallationID,
	); err != nil {
		t.Fatal(err)
	}
	if framework != "vite" || node != "22" || root != "apps/web" ||
		output != "dist" || install != "npm ci" || build != "npm run build" || resolved ||
		sourceRepository != "owner/repo" || sourceInstallationID != 42 {
		t.Fatalf("unexpected backfilled manifest: %q %q %q %q %q %q resolved=%v repo=%q installation=%d",
			framework, node, root, output, install, build, resolved, sourceRepository, sourceInstallationID)
	}
}

func TestGitHubProjectLifecycleMigrationBackfillsConnectionStatus(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "pre-github-lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	oldMigrations := fstest.MapFS{}
	for _, name := range []string{
		"001_initial.sql", "002_password_reset.sql", "003_deployments_cache.sql",
		"004_github_deploy_id.sql", "005_system_metrics.sql", "006_github_webhook_deliveries.sql",
		"007_deployment_build_manifest.sql",
	} {
		data, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		oldMigrations[name] = &fstest.MapFile{Data: data}
	}
	if err := Migrate(db, oldMigrations); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash) VALUES ('user', 'user@example.com', 'hash');
		INSERT INTO projects (
			id, owner_id, name, slug, github_repo, github_installation_id
		) VALUES ('connected', 'user', 'Connected', 'connected', 'owner/repo', 42);
		INSERT INTO projects (
			id, owner_id, name, slug
		) VALUES ('local', 'user', 'Local', 'local');
	`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}

	var connectedStatus, localStatus string
	if err := db.QueryRow(`SELECT github_connection_status FROM projects WHERE id = 'connected'`).Scan(&connectedStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT github_connection_status FROM projects WHERE id = 'local'`).Scan(&localStatus); err != nil {
		t.Fatal(err)
	}
	if connectedStatus != "active" || localStatus != "disconnected" {
		t.Fatalf("statuses = %q, %q", connectedStatus, localStatus)
	}
}

func TestRealMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, migrations.FS); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}
	if err := Migrate(db, migrations.FS); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}
}
