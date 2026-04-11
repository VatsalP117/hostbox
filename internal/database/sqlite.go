package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Open creates a new SQLite connection with WAL mode and recommended pragmas.
func Open(path string) (*sql.DB, error) {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite works best with a single writer connection
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)

	if err := applyPragmas(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply pragmas: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("database connected", "path", path)
	return db, nil
}

// applyPragmas sets WAL mode and performance/safety pragmas.
func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -20000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("exec %q: %w", p, err)
		}
	}
	return nil
}

// Close gracefully closes the database connection.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	slog.Info("closing database")
	return db.Close()
}
