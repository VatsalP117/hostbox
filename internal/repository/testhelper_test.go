package repository

import (
	"database/sql"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/migrations"
)

// setupTestDB creates a temporary SQLite database with migrations applied.
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
