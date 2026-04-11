package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type DeploymentRepository struct {
	db *sql.DB
}

func NewDeploymentRepository(db *sql.DB) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

func (r *DeploymentRepository) Create(ctx context.Context, deployment *models.Deployment) error {
	if deployment.ID == "" {
		deployment.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO deployments (id, project_id, commit_sha, commit_message, commit_author,
		  branch, status, is_production, deployment_url, artifact_path, artifact_size_bytes,
		  log_path, error_message, is_rollback, rollback_source_id, github_pr_number,
		  build_duration_ms, started_at, completed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deployment.ID, deployment.ProjectID, deployment.CommitSHA,
		deployment.CommitMessage, deployment.CommitAuthor,
		deployment.Branch, deployment.Status, deployment.IsProduction,
		deployment.DeploymentURL, deployment.ArtifactPath,
		deployment.ArtifactSizeBytes, deployment.LogPath,
		deployment.ErrorMessage, deployment.IsRollback,
		deployment.RollbackSourceID, deployment.GitHubPRNumber,
		deployment.BuildDurationMs,
		formatNullableTime(deployment.StartedAt),
		formatNullableTime(deployment.CompletedAt),
		now,
	)
	if err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}
	deployment.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *DeploymentRepository) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	row := r.db.QueryRowContext(ctx, deploymentSelectSQL+` WHERE d.id = ?`, id)
	return scanDeployment(row)
}

func (r *DeploymentRepository) Update(ctx context.Context, deployment *models.Deployment) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE deployments SET status = ?, deployment_url = ?, artifact_path = ?,
		  artifact_size_bytes = ?, log_path = ?, error_message = ?,
		  build_duration_ms = ?, started_at = ?, completed_at = ?
		 WHERE id = ?`,
		deployment.Status, deployment.DeploymentURL, deployment.ArtifactPath,
		deployment.ArtifactSizeBytes, deployment.LogPath, deployment.ErrorMessage,
		deployment.BuildDurationMs,
		formatNullableTime(deployment.StartedAt),
		formatNullableTime(deployment.CompletedAt),
		deployment.ID,
	)
	if err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id string, status models.DeploymentStatus, errorMsg *string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE deployments SET status = ?, error_message = ? WHERE id = ?`,
		status, errorMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update deployment status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DeploymentRepository) ListByProject(ctx context.Context, projectID string, page, perPage int, status, branch *string) ([]models.Deployment, int, error) {
	countQuery := `SELECT COUNT(*) FROM deployments WHERE project_id = ?`
	listQuery := deploymentSelectSQL + ` WHERE d.project_id = ?`
	args := []interface{}{projectID}

	if status != nil {
		countQuery += ` AND status = ?`
		listQuery += ` AND d.status = ?`
		args = append(args, *status)
	}
	if branch != nil {
		countQuery += ` AND branch = ?`
		listQuery += ` AND d.branch = ?`
		args = append(args, *branch)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deployments: %w", err)
	}

	offset := (page - 1) * perPage
	listQuery += ` ORDER BY d.created_at DESC LIMIT ? OFFSET ?`
	listArgs := append(args, perPage, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		d, err := scanDeploymentRows(rows)
		if err != nil {
			return nil, 0, err
		}
		deployments = append(deployments, *d)
	}
	return deployments, total, rows.Err()
}

func (r *DeploymentRepository) GetLatestByProjectAndBranch(ctx context.Context, projectID, branch string) (*models.Deployment, error) {
	row := r.db.QueryRowContext(ctx,
		deploymentSelectSQL+` WHERE d.project_id = ? AND d.branch = ? ORDER BY d.created_at DESC, d.rowid DESC LIMIT 1`,
		projectID, branch)
	return scanDeployment(row)
}

func (r *DeploymentRepository) GetActiveByProjectAndBranch(ctx context.Context, projectID, branch string) (*models.Deployment, error) {
	row := r.db.QueryRowContext(ctx,
		deploymentSelectSQL+` WHERE d.project_id = ? AND d.branch = ? AND d.status = 'ready' ORDER BY d.created_at DESC, d.rowid DESC LIMIT 1`,
		projectID, branch)
	return scanDeployment(row)
}

func (r *DeploymentRepository) CountByProject(ctx context.Context, projectID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployments WHERE project_id = ?`, projectID).Scan(&count)
	return count, err
}

func (r *DeploymentRepository) CancelQueuedByProjectAndBranch(ctx context.Context, projectID, branch string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE deployments SET status = 'cancelled' WHERE project_id = ? AND branch = ? AND status = 'queued'`,
		projectID, branch)
	if err != nil {
		return 0, fmt.Errorf("cancel queued deployments: %w", err)
	}
	return result.RowsAffected()
}

const deploymentSelectSQL = `SELECT d.id, d.project_id, d.commit_sha, d.commit_message, d.commit_author,
	d.branch, d.status, d.is_production, d.deployment_url, d.artifact_path, d.artifact_size_bytes,
	d.log_path, d.error_message, d.is_rollback, d.rollback_source_id, d.github_pr_number,
	d.build_duration_ms, d.started_at, d.completed_at, d.created_at
	FROM deployments d`

func scanDeployment(s scanner) (*models.Deployment, error) {
	var d models.Deployment
	var startedAt, completedAt, createdAt sql.NullString
	err := s.Scan(&d.ID, &d.ProjectID, &d.CommitSHA, &d.CommitMessage, &d.CommitAuthor,
		&d.Branch, &d.Status, &d.IsProduction, &d.DeploymentURL, &d.ArtifactPath,
		&d.ArtifactSizeBytes, &d.LogPath, &d.ErrorMessage, &d.IsRollback,
		&d.RollbackSourceID, &d.GitHubPRNumber, &d.BuildDurationMs,
		&startedAt, &completedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		d.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		d.CompletedAt = &t
	}
	if createdAt.Valid {
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	return &d, nil
}

func scanDeploymentRows(rows *sql.Rows) (*models.Deployment, error) {
	return scanDeployment(rows)
}

func formatNullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
