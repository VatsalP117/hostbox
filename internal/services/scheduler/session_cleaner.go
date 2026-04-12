package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// SessionCleaner deletes expired sessions periodically.
type SessionCleaner struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewSessionCleaner(db *sql.DB, logger *slog.Logger) *SessionCleaner {
	return &SessionCleaner{db: db, logger: logger}
}

func (sc *SessionCleaner) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			sc.logger.Info("session cleaner stopped")
			return
		case <-ticker.C:
			sc.Clean(ctx)
		}
	}
}

func (sc *SessionCleaner) Clean(ctx context.Context) {
	result, err := sc.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE expires_at < strftime('%Y-%m-%dT%H:%M:%SZ', 'now')")
	if err != nil {
		sc.logger.Error("session cleanup failed", "error", err)
		return
	}
	count, _ := result.RowsAffected()
	if count > 0 {
		sc.logger.Info("expired sessions cleaned", "count", count)
	}
}
