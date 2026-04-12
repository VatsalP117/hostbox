package backup

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Service struct {
	db         *sql.DB
	backupDir  string
	maxBackups int
	logger     *slog.Logger
}

type BackupResult struct {
	Filename  string `json:"filename"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

func NewService(db *sql.DB, backupDir string, maxBackups int, logger *slog.Logger) *Service {
	if maxBackups <= 0 {
		maxBackups = 5
	}
	return &Service{
		db:         db,
		backupDir:  backupDir,
		maxBackups: maxBackups,
		logger:     logger,
	}
}

func (s *Service) CreateBackup(ctx context.Context, compress bool) (*BackupResult, error) {
	if err := os.MkdirAll(s.backupDir, 0700); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405.000000")
	filename := fmt.Sprintf("hostbox-%s.db", timestamp)
	destPath := filepath.Join(s.backupDir, filename)

	// SQLite online backup — safe while database is in use (WAL mode)
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", destPath))
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	var finalPath string
	var size int64

	if compress {
		gzPath := destPath + ".gz"
		if err := gzipFile(destPath, gzPath); err != nil {
			os.Remove(destPath)
			return nil, fmt.Errorf("compression failed: %w", err)
		}
		os.Remove(destPath)
		finalPath = gzPath
		fi, _ := os.Stat(gzPath)
		size = fi.Size()
		filename += ".gz"
	} else {
		finalPath = destPath
		fi, _ := os.Stat(destPath)
		size = fi.Size()
	}

	s.enforceRetention()

	s.logger.Info("backup created",
		"path", finalPath,
		"size_bytes", size,
		"compressed", compress,
	)

	return &BackupResult{
		Filename:  filename,
		Path:      finalPath,
		SizeBytes: size,
	}, nil
}

func (s *Service) ListBackups() ([]BackupResult, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list backups: %w", err)
	}

	var backups []BackupResult
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "hostbox-") || e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupResult{
			Filename:  e.Name(),
			Path:      filepath.Join(s.backupDir, e.Name()),
			SizeBytes: fi.Size(),
		})
	}

	// Newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Filename > backups[j].Filename
	})

	return backups, nil
}

func (s *Service) Restore(ctx context.Context, backupPath string) error {
	if err := s.validateBackup(backupPath); err != nil {
		return fmt.Errorf("invalid backup file: %w", err)
	}

	dbPath := backupPath
	if strings.HasSuffix(backupPath, ".gz") {
		dbPath = strings.TrimSuffix(backupPath, ".gz") + ".restore-tmp"
		if err := gunzipFile(backupPath, dbPath); err != nil {
			return fmt.Errorf("decompression failed: %w", err)
		}
		defer os.Remove(dbPath)
	}

	// Get current database path from pragma
	var currentDB string
	err := s.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(new(int), new(string), &currentDB)
	if err != nil {
		return fmt.Errorf("get current db path: %w", err)
	}

	s.db.Close()

	// Safety backup of current database
	safetyBackup := currentDB + ".pre-restore"
	if err := copyFile(currentDB, safetyBackup); err != nil {
		return fmt.Errorf("safety backup failed: %w", err)
	}

	if err := copyFile(dbPath, currentDB); err != nil {
		copyFile(safetyBackup, currentDB) //nolint:errcheck
		return fmt.Errorf("database replacement failed: %w", err)
	}

	// Remove stale WAL/SHM files
	os.Remove(currentDB + "-wal")
	os.Remove(currentDB + "-shm")

	s.logger.Info("database restored", "from", backupPath)
	return nil
}

func (s *Service) validateBackup(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	actualPath := path
	if strings.HasSuffix(path, ".gz") {
		actualPath = path + ".validate-tmp"
		if err := gunzipFile(path, actualPath); err != nil {
			return err
		}
		defer os.Remove(actualPath)
	}

	db, err := sql.Open("sqlite3", actualPath+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		return fmt.Errorf("not a valid Hostbox database: %w", err)
	}
	return nil
}

func (s *Service) enforceRetention() {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return
	}

	var backups []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "hostbox-") && !e.IsDir() {
			backups = append(backups, e)
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() > backups[j].Name()
	})

	for i := s.maxBackups; i < len(backups); i++ {
		path := filepath.Join(s.backupDir, backups[i].Name())
		os.Remove(path)
		s.logger.Info("old backup removed", "path", path)
	}
}

func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	if _, err := io.Copy(gw, in); err != nil {
		gw.Close()
		return err
	}
	return gw.Close()
}

func gunzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	gr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gr.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gr)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
