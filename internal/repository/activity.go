package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) Create(ctx context.Context, entry *models.ActivityLog) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO activity_log (user_id, action, resource_type, resource_id, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID, entry.Metadata, now,
	)
	if err != nil {
		return fmt.Errorf("create activity log: %w", err)
	}
	id, _ := result.LastInsertId()
	entry.ID = id
	entry.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *ActivityRepository) List(ctx context.Context, page, perPage int, action, resourceType *string) ([]models.ActivityLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM activity_log WHERE 1=1`
	listQuery := activitySelectSQL + ` WHERE 1=1`
	var args []interface{}

	if action != nil {
		countQuery += ` AND action = ?`
		listQuery += ` AND action = ?`
		args = append(args, *action)
	}
	if resourceType != nil {
		countQuery += ` AND resource_type = ?`
		listQuery += ` AND resource_type = ?`
		args = append(args, *resourceType)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count activity logs: %w", err)
	}

	offset := (page - 1) * perPage
	listQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs := append(args, perPage, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list activity logs: %w", err)
	}
	defer rows.Close()

	var entries []models.ActivityLog
	for rows.Next() {
		e, err := scanActivityRows(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, *e)
	}
	return entries, total, rows.Err()
}

func (r *ActivityRepository) ListByResource(ctx context.Context, resourceType, resourceID string, page, perPage int) ([]models.ActivityLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM activity_log WHERE resource_type = ? AND resource_id = ?`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, resourceType, resourceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count activity logs by resource: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		activitySelectSQL+` WHERE resource_type = ? AND resource_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		resourceType, resourceID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list activity logs by resource: %w", err)
	}
	defer rows.Close()

	var entries []models.ActivityLog
	for rows.Next() {
		e, err := scanActivityRows(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, *e)
	}
	return entries, total, rows.Err()
}

const activitySelectSQL = `SELECT id, user_id, action, resource_type, resource_id, metadata, created_at FROM activity_log`

func scanActivity(s scanner) (*models.ActivityLog, error) {
	var a models.ActivityLog
	var createdAt string
	err := s.Scan(&a.ID, &a.UserID, &a.Action, &a.ResourceType, &a.ResourceID, &a.Metadata, &createdAt)
	if err != nil {
		return nil, err
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &a, nil
}

func scanActivityRows(rows *sql.Rows) (*models.ActivityLog, error) {
	return scanActivity(rows)
}
