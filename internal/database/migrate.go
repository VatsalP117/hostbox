package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

// Migrate runs all pending SQL migrations in order.
// migrations is an fs.FS containing SQL files named like "001_initial.sql".
func Migrate(db *sql.DB, migrations fs.FS) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	// Collect and sort migration files
	type migration struct {
		version  int
		filename string
	}
	var pending []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseMigrationVersion(entry.Name())
		if err != nil {
			return fmt.Errorf("parse migration %q: %w", entry.Name(), err)
		}
		if version > currentVersion {
			pending = append(pending, migration{version: version, filename: entry.Name()})
		}
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].version < pending[j].version
	})

	if len(pending) == 0 {
		slog.Info("no pending migrations")
		return nil
	}

	for _, m := range pending {
		content, err := fs.ReadFile(migrations, m.filename)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", m.filename, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin transaction for %q: %w", m.filename, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %q: %w", m.filename, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, filename) VALUES (?, ?)",
			m.version, m.filename,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %q: %w", m.filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %q: %w", m.filename, err)
		}

		slog.Info("migration applied", "version", m.version, "filename", m.filename)
	}

	slog.Info("migrations complete", "applied", len(pending))
	return nil
}

// getCurrentVersion returns the highest applied migration version, or 0 if none.
func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// parseMigrationVersion extracts the version number from a filename like "001_initial.sql".
func parseMigrationVersion(filename string) (int, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename: %q", filename)
	}
	return strconv.Atoi(parts[0])
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			filename TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		)
	`)
	return err
}
