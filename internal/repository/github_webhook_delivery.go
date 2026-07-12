package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

type GitHubWebhookDeliveryRepository struct {
	db *sql.DB
}

func NewGitHubWebhookDeliveryRepository(db *sql.DB) *GitHubWebhookDeliveryRepository {
	return &GitHubWebhookDeliveryRepository{db: db}
}

// Create inserts a delivery exactly once. A duplicate delivery ID is not an
// error and returns created=false.
func (r *GitHubWebhookDeliveryRepository) Create(ctx context.Context, delivery *models.GitHubWebhookDelivery) (bool, error) {
	now := time.Now().UTC()
	if delivery.NextAttemptAt.IsZero() {
		delivery.NextAttemptAt = now
	}
	if delivery.Status == "" {
		delivery.Status = models.GitHubWebhookDeliveryQueued
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO github_webhook_deliveries (
			delivery_id, event_type, payload, status, attempts, next_attempt_at,
			last_error, processing_started_at, completed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(delivery_id) DO NOTHING`,
		delivery.DeliveryID,
		delivery.EventType,
		delivery.Payload,
		delivery.Status,
		delivery.Attempts,
		formatWebhookTime(delivery.NextAttemptAt),
		delivery.LastError,
		formatNullableWebhookTime(delivery.ProcessingStartedAt),
		formatNullableWebhookTime(delivery.CompletedAt),
		formatWebhookTime(now),
		formatWebhookTime(now),
	)
	if err != nil {
		return false, fmt.Errorf("create github webhook delivery: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("create github webhook delivery rows: %w", err)
	}
	if rows == 1 {
		delivery.CreatedAt = now
		delivery.UpdatedAt = now
	}
	return rows == 1, nil
}

func (r *GitHubWebhookDeliveryRepository) GetByDeliveryID(ctx context.Context, deliveryID string) (*models.GitHubWebhookDelivery, error) {
	return scanGitHubWebhookDelivery(r.db.QueryRowContext(ctx, webhookDeliverySelectSQL+` WHERE delivery_id = ?`, deliveryID))
}

// ClaimNext atomically moves the oldest due queued delivery to processing and
// increments its attempt count. sql.ErrNoRows means no work is currently due.
func (r *GitHubWebhookDeliveryRepository) ClaimNext(ctx context.Context, now time.Time, maxAttempts int) (*models.GitHubWebhookDelivery, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin github webhook claim: %w", err)
	}
	defer tx.Rollback()

	delivery, err := scanGitHubWebhookDelivery(tx.QueryRowContext(ctx, webhookDeliverySelectSQL+`
		WHERE status = 'queued' AND attempts < ? AND next_attempt_at <= ?
		ORDER BY next_attempt_at ASC, created_at ASC
		LIMIT 1`, maxAttempts, formatWebhookTime(now)))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("select github webhook delivery for claim: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE github_webhook_deliveries
		SET status = 'processing', attempts = attempts + 1,
			processing_started_at = ?, updated_at = ?
		WHERE delivery_id = ? AND status = 'queued'`,
		formatWebhookTime(now), formatWebhookTime(now), delivery.DeliveryID)
	if err != nil {
		return nil, fmt.Errorf("claim github webhook delivery: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("claim github webhook delivery rows: %w", err)
	}
	if rows != 1 {
		return nil, sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit github webhook claim: %w", err)
	}

	delivery.Status = models.GitHubWebhookDeliveryProcessing
	delivery.Attempts++
	delivery.ProcessingStartedAt = ptrTime(now)
	delivery.UpdatedAt = now
	return delivery, nil
}

// RecoverProcessing makes interrupted work eligible again. Rows whose final
// attempt was interrupted become terminal instead of remaining permanently due.
func (r *GitHubWebhookDeliveryRepository) RecoverProcessing(ctx context.Context, now time.Time, maxAttempts int) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE github_webhook_deliveries
		SET status = CASE WHEN attempts >= ? THEN 'failed' ELSE 'queued' END,
			next_attempt_at = ?,
			last_error = CASE
				WHEN attempts >= ? THEN 'processing interrupted on final attempt'
				ELSE 'processing interrupted; recovered on startup'
			END,
			processing_started_at = NULL,
			completed_at = CASE WHEN attempts >= ? THEN ? ELSE completed_at END,
			updated_at = ?
		WHERE status = 'processing'`,
		maxAttempts,
		formatWebhookTime(now),
		maxAttempts,
		maxAttempts,
		formatWebhookTime(now),
		formatWebhookTime(now),
	)
	if err != nil {
		return 0, fmt.Errorf("recover github webhook deliveries: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("recover github webhook delivery rows: %w", err)
	}
	return rows, nil
}

func (r *GitHubWebhookDeliveryRepository) MarkCompleted(ctx context.Context, deliveryID string, now time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE github_webhook_deliveries
		SET status = 'completed', last_error = NULL, processing_started_at = NULL,
			completed_at = ?, updated_at = ?
		WHERE delivery_id = ? AND status = 'processing'`,
		formatWebhookTime(now), formatWebhookTime(now), deliveryID)
	return webhookTransitionResult(result, err, "complete github webhook delivery")
}

func (r *GitHubWebhookDeliveryRepository) MarkForRetry(ctx context.Context, deliveryID, lastError string, nextAttemptAt, now time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE github_webhook_deliveries
		SET status = 'queued', next_attempt_at = ?, last_error = ?,
			processing_started_at = NULL, updated_at = ?
		WHERE delivery_id = ? AND status = 'processing'`,
		formatWebhookTime(nextAttemptAt), lastError, formatWebhookTime(now), deliveryID)
	return webhookTransitionResult(result, err, "retry github webhook delivery")
}

func (r *GitHubWebhookDeliveryRepository) MarkFailed(ctx context.Context, deliveryID, lastError string, now time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE github_webhook_deliveries
		SET status = 'failed', last_error = ?, processing_started_at = NULL,
			completed_at = ?, updated_at = ?
		WHERE delivery_id = ? AND status = 'processing'`,
		lastError, formatWebhookTime(now), formatWebhookTime(now), deliveryID)
	return webhookTransitionResult(result, err, "fail github webhook delivery")
}

// DeleteCompletedBefore prunes terminal successful payloads in bounded batches.
// Failed rows are retained for diagnostics and manual recovery.
func (r *GitHubWebhookDeliveryRepository) DeleteCompletedBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM github_webhook_deliveries
		WHERE delivery_id IN (
			SELECT delivery_id FROM github_webhook_deliveries
			WHERE status = 'completed' AND completed_at < ?
			ORDER BY completed_at ASC LIMIT ?
		)`, formatWebhookTime(before), limit)
	if err != nil {
		return 0, fmt.Errorf("prune github webhook deliveries: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune github webhook delivery rows: %w", err)
	}
	return rows, nil
}

func webhookTransitionResult(result sql.Result, err error, operation string) (bool, error) {
	if err != nil {
		return false, fmt.Errorf("%s: %w", operation, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s rows: %w", operation, err)
	}
	return rows == 1, nil
}

const webhookDeliverySelectSQL = `SELECT delivery_id, event_type, payload, status, attempts,
	next_attempt_at, last_error, processing_started_at, completed_at, created_at, updated_at
	FROM github_webhook_deliveries`

func scanGitHubWebhookDelivery(s scanner) (*models.GitHubWebhookDelivery, error) {
	var (
		delivery                                             models.GitHubWebhookDelivery
		nextAttempt, processing, completed, created, updated string
		processingNull, completedNull                        sql.NullString
	)
	if err := s.Scan(
		&delivery.DeliveryID,
		&delivery.EventType,
		&delivery.Payload,
		&delivery.Status,
		&delivery.Attempts,
		&nextAttempt,
		&delivery.LastError,
		&processingNull,
		&completedNull,
		&created,
		&updated,
	); err != nil {
		return nil, err
	}

	var err error
	if delivery.NextAttemptAt, err = time.Parse(time.RFC3339Nano, nextAttempt); err != nil {
		return nil, fmt.Errorf("parse webhook next attempt time: %w", err)
	}
	if delivery.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return nil, fmt.Errorf("parse webhook created time: %w", err)
	}
	if delivery.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return nil, fmt.Errorf("parse webhook updated time: %w", err)
	}
	processing = processingNull.String
	if processingNull.Valid {
		t, parseErr := time.Parse(time.RFC3339Nano, processing)
		if parseErr != nil {
			return nil, fmt.Errorf("parse webhook processing time: %w", parseErr)
		}
		delivery.ProcessingStartedAt = &t
	}
	completed = completedNull.String
	if completedNull.Valid {
		t, parseErr := time.Parse(time.RFC3339Nano, completed)
		if parseErr != nil {
			return nil, fmt.Errorf("parse webhook completed time: %w", parseErr)
		}
		delivery.CompletedAt = &t
	}
	return &delivery, nil
}

func formatWebhookTime(t time.Time) string {
	// Fixed-width fractional seconds preserve chronological ordering in SQLite's
	// TEXT comparisons, including values that land exactly on a whole second.
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func formatNullableWebhookTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatWebhookTime(*t)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
