package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/util"
)

type DeploymentRepository struct {
	db *sql.DB
}

type DeploymentHealthSummary struct {
	Total                  int64
	Successful             int64
	Failed                 int64
	Cancelled              int64
	AverageBuildDurationMs *int64
	LastSuccessAt          *time.Time
	LastFailureAt          *time.Time
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
		  github_deploy_id, build_duration_ms, started_at, completed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deployment.ID, deployment.ProjectID, deployment.CommitSHA,
		deployment.CommitMessage, deployment.CommitAuthor,
		deployment.Branch, deployment.Status, deployment.IsProduction,
		deployment.DeploymentURL, deployment.ArtifactPath,
		deployment.ArtifactSizeBytes, deployment.LogPath,
		deployment.ErrorMessage, deployment.IsRollback,
		deployment.RollbackSourceID, deployment.GitHubPRNumber,
		deployment.GitHubDeployID, deployment.BuildDurationMs,
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
		  github_deploy_id = ?, build_duration_ms = ?, started_at = ?, completed_at = ?
		 WHERE id = ?`,
		deployment.Status, deployment.DeploymentURL, deployment.ArtifactPath,
		deployment.ArtifactSizeBytes, deployment.LogPath, deployment.ErrorMessage,
		deployment.GitHubDeployID, deployment.BuildDurationMs,
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

func (r *DeploymentRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployments`).Scan(&count)
	return count, err
}

func (r *DeploymentRepository) CountByStatuses(ctx context.Context, statuses ...string) (int, error) {
	if len(statuses) == 0 {
		return 0, nil
	}

	placeholders := ""
	args := make([]interface{}, len(statuses))
	for i, status := range statuses {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = status
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM deployments WHERE status IN (%s)`, placeholders)
	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *DeploymentRepository) SummarizeSince(ctx context.Context, since time.Time) (DeploymentHealthSummary, error) {
	var (
		summary       DeploymentHealthSummary
		avgDuration   sql.NullFloat64
		lastSuccessAt sql.NullString
		lastFailureAt sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN status = 'ready' THEN 1 ELSE 0 END), 0) AS successful,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed,
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) AS cancelled,
			AVG(CASE WHEN status = 'ready' AND build_duration_ms IS NOT NULL THEN build_duration_ms END) AS avg_build_duration_ms,
			MAX(CASE WHEN status = 'ready' THEN COALESCE(completed_at, created_at) END) AS last_success_at,
			MAX(CASE WHEN status = 'failed' THEN COALESCE(completed_at, created_at) END) AS last_failure_at
		FROM deployments
		WHERE created_at >= ?`,
		since.UTC().Format(time.RFC3339),
	).Scan(
		&summary.Total,
		&summary.Successful,
		&summary.Failed,
		&summary.Cancelled,
		&avgDuration,
		&lastSuccessAt,
		&lastFailureAt,
	)
	if err != nil {
		return DeploymentHealthSummary{}, fmt.Errorf("summarize deployments since %s: %w", since.UTC().Format(time.RFC3339), err)
	}

	if avgDuration.Valid {
		v := int64(avgDuration.Float64 + 0.5)
		summary.AverageBuildDurationMs = &v
	}
	if lastSuccessAt.Valid {
		t, err := time.Parse(time.RFC3339, lastSuccessAt.String)
		if err == nil {
			summary.LastSuccessAt = &t
		}
	}
	if lastFailureAt.Valid {
		t, err := time.Parse(time.RFC3339, lastFailureAt.String)
		if err == nil {
			summary.LastFailureAt = &t
		}
	}

	return summary, nil
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

// FindByStatus returns all deployments with the given status.
func (r *DeploymentRepository) FindByStatus(ctx context.Context, status string) ([]models.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, deploymentSelectSQL+` WHERE d.status = ? ORDER BY d.created_at ASC`, status)
	if err != nil {
		return nil, fmt.Errorf("find by status: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		d, err := scanDeploymentRows(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *d)
	}
	return deployments, rows.Err()
}

// FindQueuedOrBuilding returns the first queued or building deployment for a project+branch.
func (r *DeploymentRepository) FindQueuedOrBuilding(ctx context.Context, projectID, branch string) (*models.Deployment, error) {
	row := r.db.QueryRowContext(ctx,
		deploymentSelectSQL+` WHERE d.project_id = ? AND d.branch = ? AND d.status IN ('queued', 'building') ORDER BY d.created_at ASC LIMIT 1`,
		projectID, branch)
	return scanDeployment(row)
}

// FindLatestReady returns the latest ready deployment for a project,
// optionally filtered to production-only.
func (r *DeploymentRepository) FindLatestReady(ctx context.Context, projectID string, production bool) (*models.Deployment, error) {
	query := deploymentSelectSQL + ` WHERE d.project_id = ? AND d.status = 'ready'`
	args := []interface{}{projectID}
	if production {
		query += ` AND d.is_production = TRUE`
	}
	query += ` ORDER BY d.created_at DESC, d.rowid DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, args...)
	return scanDeployment(row)
}

const deploymentSelectSQL = `SELECT d.id, d.project_id, d.commit_sha, d.commit_message, d.commit_author,
	d.branch, d.status, d.is_production, d.deployment_url, d.artifact_path, d.artifact_size_bytes,
	d.log_path, d.error_message, d.is_rollback, d.rollback_source_id, d.github_pr_number,
	d.github_deploy_id, d.build_duration_ms, d.started_at, d.completed_at, d.created_at
	FROM deployments d`

func scanDeployment(s scanner) (*models.Deployment, error) {
	var d models.Deployment
	var startedAt, completedAt, createdAt sql.NullString
	err := s.Scan(&d.ID, &d.ProjectID, &d.CommitSHA, &d.CommitMessage, &d.CommitAuthor,
		&d.Branch, &d.Status, &d.IsProduction, &d.DeploymentURL, &d.ArtifactPath,
		&d.ArtifactSizeBytes, &d.LogPath, &d.ErrorMessage, &d.IsRollback,
		&d.RollbackSourceID, &d.GitHubPRNumber, &d.GitHubDeployID, &d.BuildDurationMs,
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

// ListActiveWithProject returns all ready deployments joined with project info for Caddy sync.
func (r *DeploymentRepository) ListActiveWithProject(ctx context.Context) ([]ActiveDeploymentRow, error) {
	query := `SELECT d.id, d.project_id, p.slug, d.branch, d.commit_sha,
		d.is_production, d.artifact_path, p.framework
		FROM deployments d
		JOIN projects p ON d.project_id = p.id
		WHERE d.status = 'ready' AND d.artifact_path IS NOT NULL`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active deployments: %w", err)
	}
	defer rows.Close()

	var result []ActiveDeploymentRow
	for rows.Next() {
		var row ActiveDeploymentRow
		if err := rows.Scan(&row.DeploymentID, &row.ProjectID, &row.ProjectSlug,
			&row.Branch, &row.CommitSHA, &row.IsProduction, &row.ArtifactPath, &row.Framework); err != nil {
			return nil, fmt.Errorf("scan active deployment: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// FindByCommitSHA finds a deployment by project and commit SHA.
func (r *DeploymentRepository) FindByCommitSHA(ctx context.Context, projectID, commitSHA string) (*models.Deployment, error) {
	row := r.db.QueryRowContext(ctx,
		deploymentSelectSQL+` WHERE d.project_id = ? AND d.commit_sha = ? ORDER BY d.created_at DESC LIMIT 1`,
		projectID, commitSHA)
	d, err := scanDeployment(row)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// DeactivateBranchDeployments cancels all ready deployments for a branch, returning the affected deployments.
func (r *DeploymentRepository) DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error) {
	// Find affected deployments first
	rows, err := r.db.QueryContext(ctx,
		deploymentSelectSQL+` WHERE d.project_id = ? AND d.branch = ? AND d.status = 'ready' AND d.is_production = FALSE`,
		projectID, branch)
	if err != nil {
		return nil, fmt.Errorf("find branch deployments: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		d, err := scanDeploymentRows(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(deployments) == 0 {
		return nil, nil
	}

	// Cancel them
	_, err = r.db.ExecContext(ctx,
		`UPDATE deployments SET status = 'cancelled' WHERE project_id = ? AND branch = ? AND status = 'ready' AND is_production = FALSE`,
		projectID, branch)
	if err != nil {
		return nil, fmt.Errorf("deactivate branch deployments: %w", err)
	}

	return deployments, nil
}

// ListAllByProject returns all deployments for a project ordered by created_at DESC.
func (r *DeploymentRepository) ListAllByProject(ctx context.Context, projectID string) ([]models.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, deploymentSelectSQL+` WHERE d.project_id = ? ORDER BY d.created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list all by project: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		d, err := scanDeploymentRows(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *d)
	}
	return deployments, rows.Err()
}

func (r *DeploymentRepository) ListRecent(ctx context.Context, page, perPage int) ([]models.Deployment, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployments`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deployments: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx, deploymentSelectSQL+` ORDER BY d.created_at DESC LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list recent deployments: %w", err)
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

// ClearArtifact nullifies artifact and log paths for a deployment (keeps record for history).
func (r *DeploymentRepository) ClearArtifact(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE deployments SET artifact_path = NULL, log_path = NULL WHERE id = ?`, id)
	return err
}

// HasLogPath checks if a deployment with the given ID has a non-null log_path.
func (r *DeploymentRepository) HasLogPath(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM deployments WHERE id = ? AND log_path IS NOT NULL)`, id).Scan(&exists)
	return exists, err
}

type ActiveDeploymentRow struct {
	DeploymentID string
	ProjectID    string
	ProjectSlug  string
	Branch       string
	CommitSHA    string
	IsProduction bool
	ArtifactPath *string
	Framework    *string
}
