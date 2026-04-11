package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

func (r *SettingsRepository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

func (r *SettingsRepository) GetBool(ctx context.Context, key string) (bool, error) {
	val, err := r.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return val == "true", nil
}

func (r *SettingsRepository) GetInt(ctx context.Context, key string) (int, error) {
	val, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("setting %q is not an integer: %w", key, err)
	}
	return n, nil
}
