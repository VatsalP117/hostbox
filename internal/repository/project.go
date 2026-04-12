package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	if project.ID == "" {
		project.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id, owner_id, name, slug, github_repo, github_installation_id,
		  production_branch, framework, build_command, install_command, output_directory,
		  root_directory, node_version, auto_deploy, preview_deployments, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		project.ID, project.OwnerID, project.Name, project.Slug,
		project.GitHubRepo, project.GitHubInstallationID,
		project.ProductionBranch, project.Framework,
		project.BuildCommand, project.InstallCommand,
		project.OutputDirectory, project.RootDirectory,
		project.NodeVersion, project.AutoDeploy,
		project.PreviewDeployments, now, now,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	project.CreatedAt, _ = time.Parse(time.RFC3339, now)
	project.UpdatedAt = project.CreatedAt
	return nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*models.Project, error) {
	row := r.db.QueryRowContext(ctx, projectSelectSQL+` WHERE p.id = ?`, id)
	return scanProject(row)
}

func (r *ProjectRepository) GetBySlug(ctx context.Context, slug string) (*models.Project, error) {
	row := r.db.QueryRowContext(ctx, projectSelectSQL+` WHERE p.slug = ?`, slug)
	return scanProject(row)
}

func (r *ProjectRepository) GetByGitHubRepo(ctx context.Context, repo string) (*models.Project, error) {
	row := r.db.QueryRowContext(ctx, projectSelectSQL+` WHERE p.github_repo = ?`, repo)
	return scanProject(row)
}

func (r *ProjectRepository) Update(ctx context.Context, project *models.Project) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, slug = ?, github_repo = ?, github_installation_id = ?,
		  production_branch = ?, framework = ?, build_command = ?, install_command = ?,
		  output_directory = ?, root_directory = ?, node_version = ?,
		  auto_deploy = ?, preview_deployments = ?, updated_at = ?
		 WHERE id = ?`,
		project.Name, project.Slug, project.GitHubRepo, project.GitHubInstallationID,
		project.ProductionBranch, project.Framework,
		project.BuildCommand, project.InstallCommand,
		project.OutputDirectory, project.RootDirectory,
		project.NodeVersion, project.AutoDeploy,
		project.PreviewDeployments, now, project.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	project.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ProjectRepository) ListByOwner(ctx context.Context, ownerID string, page, perPage int, search string) ([]models.Project, int, error) {
	countQuery := `SELECT COUNT(*) FROM projects WHERE owner_id = ?`
	listQuery := projectSelectSQL + ` WHERE p.owner_id = ?`
	args := []interface{}{ownerID}

	if search != "" {
		filter := ` AND p.name LIKE ?`
		countQuery += ` AND name LIKE ?`
		listQuery += filter
		args = append(args, "%"+search+"%")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	offset := (page - 1) * perPage
	listQuery += ` ORDER BY p.created_at DESC LIMIT ? OFFSET ?`
	listArgs := append(args, perPage, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		p, err := scanProjectRows(rows)
		if err != nil {
			return nil, 0, err
		}
		projects = append(projects, *p)
	}
	return projects, total, rows.Err()
}

func (r *ProjectRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`).Scan(&count)
	return count, err
}

func (r *ProjectRepository) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE owner_id = ?`, ownerID).Scan(&count)
	return count, err
}

const projectSelectSQL = `SELECT p.id, p.owner_id, p.name, p.slug, p.github_repo, p.github_installation_id,
	p.production_branch, p.framework, p.build_command, p.install_command, p.output_directory,
	p.root_directory, p.node_version, p.auto_deploy, p.preview_deployments,
	p.lock_file_hash, p.detected_package_manager, p.created_at, p.updated_at
	FROM projects p`

func scanProject(s scanner) (*models.Project, error) {
	var p models.Project
	var createdAt, updatedAt string
	err := s.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Slug, &p.GitHubRepo, &p.GitHubInstallationID,
		&p.ProductionBranch, &p.Framework, &p.BuildCommand, &p.InstallCommand, &p.OutputDirectory,
		&p.RootDirectory, &p.NodeVersion, &p.AutoDeploy, &p.PreviewDeployments,
		&p.LockFileHash, &p.DetectedPackageManager, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &p, nil
}

func scanProjectRows(rows *sql.Rows) (*models.Project, error) {
	return scanProject(rows)
}

// UpdateBuildMeta updates the detected package manager and lock file hash for a project.
func (r *ProjectRepository) UpdateBuildMeta(ctx context.Context, projectID, pkgManager, lockHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET detected_package_manager = ?, lock_file_hash = ?, updated_at = ? WHERE id = ?`,
		pkgManager, lockHash, time.Now().UTC().Format(time.RFC3339), projectID,
	)
	if err != nil {
		return fmt.Errorf("update build meta: %w", err)
	}
	return nil
}

// ListAll returns all projects (used by garbage collection).
func (r *ProjectRepository) ListAll(ctx context.Context) ([]models.Project, error) {
	rows, err := r.db.QueryContext(ctx, projectSelectSQL+` ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		p, err := scanProjectRows(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

// Exists checks if a project with the given ID exists.
func (r *ProjectRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id = ?)`, id).Scan(&exists)
	return exists, err
}

// ClearInstallation clears the GitHub installation ID for all projects with the given installation.
func (r *ProjectRepository) ClearInstallation(ctx context.Context, installationID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET github_installation_id = NULL, updated_at = ? WHERE github_installation_id = ?`,
		time.Now().UTC().Format(time.RFC3339), installationID,
	)
	if err != nil {
		return fmt.Errorf("clear installation: %w", err)
	}
	return nil
}
