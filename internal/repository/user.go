package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if user.ID == "" {
		user.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, display_name, is_admin, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.DisplayName,
		user.IsAdmin, user.EmailVerified, now, now,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, now)
	user.UpdatedAt = user.CreatedAt
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, is_admin, email_verified, created_at, updated_at
		 FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, is_admin, email_verified, created_at, updated_at
		 FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET email = ?, display_name = ?, is_admin = ?, email_verified = ?, updated_at = ?
		 WHERE id = ?`,
		user.Email, user.DisplayName, user.IsAdmin, user.EmailVerified, now, user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	user.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *UserRepository) List(ctx context.Context, page, perPage int) ([]models.User, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, password_hash, display_name, is_admin, email_verified, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	return users, total, rows.Err()
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, now, id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(s scanner) (*models.User, error) {
	var u models.User
	var createdAt, updatedAt string
	err := s.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.IsAdmin, &u.EmailVerified, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func scanUserRows(rows *sql.Rows) (*models.User, error) {
	return scanUser(rows)
}
