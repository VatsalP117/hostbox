package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestWALMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode error: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestForeignKeysEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	var fk int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys error: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestBasicReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	_, err = db.Exec("INSERT INTO test (name) VALUES (?)", "hello")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM test WHERE id = 1").Scan(&name)
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if name != "hello" {
		t.Errorf("name = %q, want %q", name, "hello")
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Error("parent directory was not created")
	}
}

func TestCloseNil(t *testing.T) {
	if err := Close(nil); err != nil {
		t.Errorf("Close(nil) error: %v", err)
	}
}
