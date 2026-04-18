package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/util"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, config *models.NotificationConfig) error {
	if config.ID == "" {
		config.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification_configs (id, project_id, channel, webhook_url, events, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		config.ID, config.ProjectID, config.Channel, config.WebhookURL,
		config.Events, config.Enabled, now,
	)
	if err != nil {
		return fmt.Errorf("create notification config: %w", err)
	}
	config.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*models.NotificationConfig, error) {
	row := r.db.QueryRowContext(ctx, notifSelectSQL+` WHERE id = ?`, id)
	return scanNotification(row)
}

func (r *NotificationRepository) Update(ctx context.Context, config *models.NotificationConfig) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE notification_configs SET webhook_url = ?, events = ?, enabled = ? WHERE id = ?`,
		config.WebhookURL, config.Events, config.Enabled, config.ID,
	)
	if err != nil {
		return fmt.Errorf("update notification config: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM notification_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete notification config: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *NotificationRepository) ListByProject(ctx context.Context, projectID string) ([]models.NotificationConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		notifSelectSQL+` WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list notification configs: %w", err)
	}
	defer rows.Close()

	var configs []models.NotificationConfig
	for rows.Next() {
		c, err := scanNotificationRows(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, *c)
	}
	return configs, rows.Err()
}

func (r *NotificationRepository) ListGlobal(ctx context.Context) ([]models.NotificationConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		notifSelectSQL+` WHERE project_id IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list global notification configs: %w", err)
	}
	defer rows.Close()

	var configs []models.NotificationConfig
	for rows.Next() {
		c, err := scanNotificationRows(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, *c)
	}
	return configs, rows.Err()
}

func (r *NotificationRepository) FindByProjectAndEvent(ctx context.Context, projectID, event string) ([]models.NotificationConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		notifSelectSQL+` WHERE project_id = ? AND enabled = TRUE AND (events = 'all' OR events LIKE '%' || ? || '%')`,
		projectID, event)
	if err != nil {
		return nil, fmt.Errorf("find notification configs by project and event: %w", err)
	}
	defer rows.Close()

	var configs []models.NotificationConfig
	for rows.Next() {
		c, err := scanNotificationRows(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, *c)
	}
	return configs, rows.Err()
}

func (r *NotificationRepository) FindGlobalByEvent(ctx context.Context, event string) ([]models.NotificationConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		notifSelectSQL+` WHERE project_id IS NULL AND enabled = TRUE AND (events = 'all' OR events LIKE '%' || ? || '%')`,
		event)
	if err != nil {
		return nil, fmt.Errorf("find global notification configs by event: %w", err)
	}
	defer rows.Close()

	var configs []models.NotificationConfig
	for rows.Next() {
		c, err := scanNotificationRows(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, *c)
	}
	return configs, rows.Err()
}

const notifSelectSQL = `SELECT id, project_id, channel, webhook_url, events, enabled, created_at FROM notification_configs`

func scanNotification(s scanner) (*models.NotificationConfig, error) {
	var c models.NotificationConfig
	var createdAt string
	err := s.Scan(&c.ID, &c.ProjectID, &c.Channel, &c.WebhookURL, &c.Events, &c.Enabled, &createdAt)
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &c, nil
}

func scanNotificationRows(rows *sql.Rows) (*models.NotificationConfig, error) {
	return scanNotification(rows)
}
