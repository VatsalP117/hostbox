package database

import (
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestMigrateApplies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"001_create_users.sql": {Data: []byte("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT);")},
		"002_create_posts.sql": {Data: []byte("CREATE TABLE posts (id TEXT PRIMARY KEY, title TEXT);")},
	}

	if err := Migrate(db, migrations); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Verify tables exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("users table not created: %v", err)
	}
	err = db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	if err != nil {
		t.Fatalf("posts table not created: %v", err)
	}

	// Verify schema_migrations
	var migCount int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migCount)
	if err != nil {
		t.Fatalf("schema_migrations query error: %v", err)
	}
	if migCount != 2 {
		t.Errorf("schema_migrations count = %d, want 2", migCount)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"001_create_users.sql": {Data: []byte("CREATE TABLE users (id TEXT PRIMARY KEY);")},
	}

	if err := Migrate(db, migrations); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}
	if err := Migrate(db, migrations); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if count != 1 {
		t.Errorf("schema_migrations count = %d, want 1", count)
	}
}

func TestMigrateOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"003_third.sql":  {Data: []byte("CREATE TABLE third (id TEXT PRIMARY KEY);")},
		"001_first.sql":  {Data: []byte("CREATE TABLE first (id TEXT PRIMARY KEY);")},
		"002_second.sql": {Data: []byte("CREATE TABLE second (id TEXT PRIMARY KEY);")},
	}

	if err := Migrate(db, migrations); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		rows.Scan(&v)
		versions = append(versions, v)
	}

	if len(versions) != 3 || versions[0] != 1 || versions[1] != 2 || versions[2] != 3 {
		t.Errorf("versions = %v, want [1, 2, 3]", versions)
	}
}

func TestMigrateInvalidSQL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"001_bad.sql": {Data: []byte("THIS IS NOT VALID SQL;")},
	}

	err = Migrate(db, migrations)
	if err == nil {
		t.Fatal("Migrate() should fail with invalid SQL")
	}

	// Verify migration was NOT recorded
	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if count != 0 {
		t.Errorf("schema_migrations count = %d, want 0 (rollback should have occurred)", count)
	}
}

func TestParseMigrationVersion(t *testing.T) {
	tests := []struct {
		filename string
		want     int
		wantErr  bool
	}{
		{"001_initial.sql", 1, false},
		{"010_add_index.sql", 10, false},
		{"bad.sql", 0, true},
		{"abc_test.sql", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got, err := parseMigrationVersion(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMigrationVersion(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMigrationVersion(%q) = %d, want %d", tt.filename, got, tt.want)
			}
		})
	}
}
