package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/logger"
	"github.com/vatsalpatel/hostbox/migrations"
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

func TestCreateBackup_Uncompressed(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	l := logger.Setup("error", "text")

	svc := NewService(db, backupDir, 5, l)

	result, err := svc.CreateBackup(context.Background(), false)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if result.SizeBytes == 0 {
		t.Error("expected non-zero backup size")
	}

	if _, err := os.Stat(result.Path); err != nil {
		t.Errorf("backup file not found: %v", err)
	}

	if filepath.Ext(result.Filename) != ".db" {
		t.Errorf("expected .db extension, got %s", result.Filename)
	}
}

func TestCreateBackup_Compressed(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	l := logger.Setup("error", "text")

	svc := NewService(db, backupDir, 5, l)

	result, err := svc.CreateBackup(context.Background(), true)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if result.SizeBytes == 0 {
		t.Error("expected non-zero backup size")
	}

	if filepath.Ext(result.Filename) != ".gz" {
		t.Errorf("expected .gz extension, got %s", result.Filename)
	}

	// Verify the uncompressed file was removed
	dbPath := result.Path[:len(result.Path)-3] // strip .gz
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Error("uncompressed file should have been removed")
	}
}

func TestListBackups(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	l := logger.Setup("error", "text")

	svc := NewService(db, backupDir, 5, l)

	svc.CreateBackup(context.Background(), false)
	svc.CreateBackup(context.Background(), true)

	backups, err := svc.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}

	// Should be newest first
	if backups[0].Filename < backups[1].Filename {
		t.Error("expected newest backup first")
	}
}

func TestEnforceRetention(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	l := logger.Setup("error", "text")

	svc := NewService(db, backupDir, 2, l)

	// Create 4 backups — only 2 should remain
	for i := 0; i < 4; i++ {
		_, err := svc.CreateBackup(context.Background(), false)
		if err != nil {
			t.Fatalf("CreateBackup %d failed: %v", i, err)
		}
	}

	backups, _ := svc.ListBackups()
	if len(backups) != 2 {
		t.Errorf("expected 2 backups after retention, got %d", len(backups))
	}
}

func TestValidateBackup(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	l := logger.Setup("error", "text")

	svc := NewService(db, backupDir, 5, l)

	result, _ := svc.CreateBackup(context.Background(), false)

	// Valid backup should pass
	err := svc.validateBackup(result.Path)
	if err != nil {
		t.Errorf("valid backup should pass validation: %v", err)
	}

	// Invalid file should fail
	badFile := filepath.Join(backupDir, "bad.db")
	os.WriteFile(badFile, []byte("not a database"), 0644)
	err = svc.validateBackup(badFile)
	if err == nil {
		t.Error("invalid backup should fail validation")
	}
}

func TestValidateBackup_GzipFile(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	l := logger.Setup("error", "text")

	svc := NewService(db, backupDir, 5, l)

	result, _ := svc.CreateBackup(context.Background(), true)

	err := svc.validateBackup(result.Path)
	if err != nil {
		t.Errorf("valid gzip backup should pass validation: %v", err)
	}
}

func TestGzipRoundtrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	gz := filepath.Join(dir, "test.txt.gz")
	dst := filepath.Join(dir, "test-restored.txt")

	original := []byte("hello world backup test data")
	os.WriteFile(src, original, 0644)

	if err := gzipFile(src, gz); err != nil {
		t.Fatalf("gzipFile failed: %v", err)
	}

	if err := gunzipFile(gz, dst); err != nil {
		t.Fatalf("gunzipFile failed: %v", err)
	}

	restored, _ := os.ReadFile(dst)
	if string(restored) != string(original) {
		t.Errorf("roundtrip mismatch: got %q, want %q", restored, original)
	}
}
