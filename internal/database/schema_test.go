package database

import (
	"path/filepath"
	"testing"

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
	tables := []string{"users", "sessions", "projects", "deployments", "domains", "env_vars", "notification_configs", "activity_log", "settings"}
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
