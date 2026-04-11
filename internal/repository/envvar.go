package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type EnvVarRepository struct {
	db *sql.DB
}

func NewEnvVarRepository(db *sql.DB) *EnvVarRepository {
	return &EnvVarRepository{db: db}
}

func (r *EnvVarRepository) Create(ctx context.Context, envVar *models.EnvVar) error {
	if envVar.ID == "" {
		envVar.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO env_vars (id, project_id, key, encrypted_value, is_secret, scope, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		envVar.ID, envVar.ProjectID, envVar.Key, envVar.EncryptedValue,
		envVar.IsSecret, envVar.Scope, now, now,
	)
	if err != nil {
		return fmt.Errorf("create env var: %w", err)
	}
	envVar.CreatedAt, _ = time.Parse(time.RFC3339, now)
	envVar.UpdatedAt = envVar.CreatedAt
	return nil
}

func (r *EnvVarRepository) GetByID(ctx context.Context, id string) (*models.EnvVar, error) {
	row := r.db.QueryRowContext(ctx, envVarSelectSQL+` WHERE id = ?`, id)
	return scanEnvVar(row)
}

func (r *EnvVarRepository) Update(ctx context.Context, envVar *models.EnvVar) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE env_vars SET key = ?, encrypted_value = ?, is_secret = ?, scope = ?, updated_at = ? WHERE id = ?`,
		envVar.Key, envVar.EncryptedValue, envVar.IsSecret, envVar.Scope, now, envVar.ID,
	)
	if err != nil {
		return fmt.Errorf("update env var: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	envVar.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *EnvVarRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM env_vars WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete env var: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *EnvVarRepository) ListByProject(ctx context.Context, projectID string) ([]models.EnvVar, error) {
	rows, err := r.db.QueryContext(ctx, envVarSelectSQL+` WHERE project_id = ? ORDER BY key ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list env vars: %w", err)
	}
	defer rows.Close()

	var envVars []models.EnvVar
	for rows.Next() {
		ev, err := scanEnvVarRows(rows)
		if err != nil {
			return nil, err
		}
		envVars = append(envVars, *ev)
	}
	return envVars, rows.Err()
}

func (r *EnvVarRepository) ListByProjectAndScope(ctx context.Context, projectID string, scope string) ([]models.EnvVar, error) {
	rows, err := r.db.QueryContext(ctx,
		envVarSelectSQL+` WHERE project_id = ? AND (scope = ? OR scope = 'all') ORDER BY key ASC`,
		projectID, scope)
	if err != nil {
		return nil, fmt.Errorf("list env vars by scope: %w", err)
	}
	defer rows.Close()

	var envVars []models.EnvVar
	for rows.Next() {
		ev, err := scanEnvVarRows(rows)
		if err != nil {
			return nil, err
		}
		envVars = append(envVars, *ev)
	}
	return envVars, rows.Err()
}

func (r *EnvVarRepository) GetByProjectKeyScope(ctx context.Context, projectID, key, scope string) (*models.EnvVar, error) {
	row := r.db.QueryRowContext(ctx,
		envVarSelectSQL+` WHERE project_id = ? AND key = ? AND scope = ?`,
		projectID, key, scope)
	return scanEnvVar(row)
}

func (r *EnvVarRepository) Upsert(ctx context.Context, envVar *models.EnvVar) error {
	if envVar.ID == "" {
		envVar.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO env_vars (id, project_id, key, encrypted_value, is_secret, scope, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(project_id, key, scope) DO UPDATE SET
		   encrypted_value = excluded.encrypted_value,
		   is_secret = excluded.is_secret,
		   updated_at = excluded.updated_at`,
		envVar.ID, envVar.ProjectID, envVar.Key, envVar.EncryptedValue,
		envVar.IsSecret, envVar.Scope, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert env var: %w", err)
	}
	envVar.CreatedAt, _ = time.Parse(time.RFC3339, now)
	envVar.UpdatedAt = envVar.CreatedAt
	return nil
}

const envVarSelectSQL = `SELECT id, project_id, key, encrypted_value, is_secret, scope, created_at, updated_at FROM env_vars`

func scanEnvVar(s scanner) (*models.EnvVar, error) {
	var ev models.EnvVar
	var createdAt, updatedAt string
	err := s.Scan(&ev.ID, &ev.ProjectID, &ev.Key, &ev.EncryptedValue,
		&ev.IsSecret, &ev.Scope, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	ev.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	ev.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &ev, nil
}

func scanEnvVarRows(rows *sql.Rows) (*models.EnvVar, error) {
	return scanEnvVar(rows)
}

// DecryptedEnvVar is a key-value pair with the decrypted value.
type DecryptedEnvVar struct {
	Key   string
	Value string
}

// GetDecryptedForBuild returns decrypted env vars for a project, filtered by scope.
// Matches env vars with the given scope OR scope="all".
func (r *EnvVarRepository) GetDecryptedForBuild(ctx context.Context, projectID, scope, encryptionKey string) ([]DecryptedEnvVar, error) {
	envVars, err := r.ListByProjectAndScope(ctx, projectID, scope)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedEnvVar, 0, len(envVars))
	for _, ev := range envVars {
		val, err := util.Decrypt(ev.EncryptedValue, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt env var %s: %w", ev.Key, err)
		}
		result = append(result, DecryptedEnvVar{Key: ev.Key, Value: val})
	}
	return result, nil
}
