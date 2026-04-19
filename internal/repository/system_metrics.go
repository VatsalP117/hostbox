package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

type SystemMetricRepository struct {
	db *sql.DB
}

func NewSystemMetricRepository(db *sql.DB) *SystemMetricRepository {
	return &SystemMetricRepository{db: db}
}

func (r *SystemMetricRepository) CreateSnapshot(ctx context.Context, snapshot *models.SystemMetricSnapshot) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO system_metric_snapshots (
			cpu_usage_percent, load1, load5, load15,
			memory_used_bytes, memory_total_bytes, memory_available_bytes, memory_usage_percent,
			disk_used_bytes, disk_total_bytes, disk_available_bytes, disk_usage_percent,
			active_builds, queued_builds, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.CPUUsagePercent,
		snapshot.Load1,
		snapshot.Load5,
		snapshot.Load15,
		snapshot.MemoryUsedBytes,
		snapshot.MemoryTotalBytes,
		snapshot.MemoryAvailableBytes,
		snapshot.MemoryUsagePercent,
		snapshot.DiskUsedBytes,
		snapshot.DiskTotalBytes,
		snapshot.DiskAvailableBytes,
		snapshot.DiskUsagePercent,
		snapshot.ActiveBuilds,
		snapshot.QueuedBuilds,
		snapshot.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create system metric snapshot: %w", err)
	}

	snapshot.ID, _ = result.LastInsertId()
	return nil
}

func (r *SystemMetricRepository) ListSince(ctx context.Context, since time.Time, limit int) ([]models.SystemMetricSnapshot, error) {
	query := `
		SELECT id, cpu_usage_percent, load1, load5, load15,
			memory_used_bytes, memory_total_bytes, memory_available_bytes, memory_usage_percent,
			disk_used_bytes, disk_total_bytes, disk_available_bytes, disk_usage_percent,
			active_builds, queued_builds, created_at
		FROM system_metric_snapshots
		WHERE created_at >= ?
		ORDER BY created_at ASC`
	args := []interface{}{since.UTC().Format(time.RFC3339)}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list system metric snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []models.SystemMetricSnapshot
	for rows.Next() {
		var snapshot models.SystemMetricSnapshot
		var createdAt string
		if err := rows.Scan(
			&snapshot.ID,
			&snapshot.CPUUsagePercent,
			&snapshot.Load1,
			&snapshot.Load5,
			&snapshot.Load15,
			&snapshot.MemoryUsedBytes,
			&snapshot.MemoryTotalBytes,
			&snapshot.MemoryAvailableBytes,
			&snapshot.MemoryUsagePercent,
			&snapshot.DiskUsedBytes,
			&snapshot.DiskTotalBytes,
			&snapshot.DiskAvailableBytes,
			&snapshot.DiskUsagePercent,
			&snapshot.ActiveBuilds,
			&snapshot.QueuedBuilds,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan system metric snapshot: %w", err)
		}
		snapshot.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

func (r *SystemMetricRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM system_metric_snapshots WHERE created_at < ?`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete old system metric snapshots: %w", err)
	}
	return result.RowsAffected()
}
