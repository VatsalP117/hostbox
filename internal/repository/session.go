package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	if session.ID == "" {
		session.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.RefreshTokenHash,
		session.UserAgent, session.IPAddress,
		session.ExpiresAt.UTC().Format(time.RFC3339), now,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	session.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	var s models.Session
	var expiresAt, createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
		 FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &expiresAt, &createdAt)
	if err != nil {
		return nil, err
	}
	s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &s, nil
}

func (r *SessionRepository) DeleteByID(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return 0, fmt.Errorf("delete sessions by user: %w", err)
	}
	return result.RowsAffected()
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return result.RowsAffected()
}

func (r *SessionRepository) ListByUserID(ctx context.Context, userID string) ([]models.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
		 FROM sessions WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var expiresAt, createdAt string
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &expiresAt, &createdAt); err != nil {
			return nil, err
		}
		s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
