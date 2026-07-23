package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/util"
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
	if project.ProductionBranch == "" {
		project.ProductionBranch = "main"
	}
	if project.RootDirectory == "" {
		project.RootDirectory = "/"
	}
	if project.NodeVersion == "" {
		project.NodeVersion = "20"
	}
	if !project.AutoDeploy {
		project.AutoDeploy = true
	}
	if !project.PreviewDeployments {
		project.PreviewDeployments = true
	}
	if project.GitHubConnectionStatus == "" {
		if project.GitHubRepo != nil && project.GitHubInstallationID != nil {
			project.GitHubConnectionStatus = models.GitHubConnectionActive
		} else {
			project.GitHubConnectionStatus = models.GitHubConnectionDisconnected
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id, owner_id, name, slug, github_repo, github_installation_id,
		  github_repository_id, github_connection_status,
		  production_branch, framework, build_command, install_command, output_directory,
		  root_directory, node_version, auto_deploy, preview_deployments, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		project.ID, project.OwnerID, project.Name, project.Slug,
		project.GitHubRepo, project.GitHubInstallationID,
		project.GitHubRepositoryID, project.GitHubConnectionStatus,
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

// ListByGitHubSource returns every active project associated with an exact
// installation/repository pair. Multiple projects may intentionally map to one
// repository, so webhook handlers must fan out deterministically.
func (r *ProjectRepository) ListByGitHubSource(ctx context.Context, installationID int64, repo string) ([]models.Project, error) {
	rows, err := r.db.QueryContext(ctx, projectSelectSQL+`
		WHERE p.github_installation_id = ?
		  AND LOWER(p.github_repo) = LOWER(?)
		  AND p.github_connection_status = ?
		ORDER BY p.created_at ASC, p.id ASC`,
		installationID, repo, models.GitHubConnectionActive)
	if err != nil {
		return nil, fmt.Errorf("list projects by github source: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		project, err := scanProjectRows(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *project)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) Update(ctx context.Context, project *models.Project) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, slug = ?, github_repo = ?, github_installation_id = ?,
		  github_repository_id = ?, github_connection_status = ?,
		  production_branch = ?, framework = ?, build_command = ?, install_command = ?,
		  output_directory = ?, root_directory = ?, node_version = ?,
		  auto_deploy = ?, preview_deployments = ?, updated_at = ?
		 WHERE id = ?`,
		project.Name, project.Slug, project.GitHubRepo, project.GitHubInstallationID,
		project.GitHubRepositoryID, project.GitHubConnectionStatus,
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
	p.github_repository_id, p.github_connection_status,
	p.production_branch, p.framework, p.build_command, p.install_command, p.output_directory,
	p.root_directory, p.node_version, p.auto_deploy, p.preview_deployments,
	p.lock_file_hash, p.detected_package_manager, p.created_at, p.updated_at
	FROM projects p`

func scanProject(s scanner) (*models.Project, error) {
	var p models.Project
	var createdAt, updatedAt string
	err := s.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Slug, &p.GitHubRepo, &p.GitHubInstallationID,
		&p.GitHubRepositoryID, &p.GitHubConnectionStatus,
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

// UpdateBuildMeta updates metadata detected from the exact source revision being built.
func (r *ProjectRepository) UpdateBuildMeta(ctx context.Context, projectID, framework, pkgManager, lockHash string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE projects SET framework = ?, detected_package_manager = ?, lock_file_hash = ?, updated_at = ? WHERE id = ?`,
		framework, pkgManager, lockHash, time.Now().UTC().Format(time.RFC3339), projectID,
	)
	if err != nil {
		return fmt.Errorf("update build meta: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
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
		`UPDATE projects
		 SET github_installation_id = NULL, github_connection_status = ?, updated_at = ?
		 WHERE github_installation_id = ?`,
		models.GitHubConnectionDisconnected, time.Now().UTC().Format(time.RFC3339), installationID,
	)
	if err != nil {
		return fmt.Errorf("clear installation: %w", err)
	}
	return nil
}

// SetInstallationStatus updates every project still associated with an
// installation. Suspension is reversible and deliberately retains source
// identity so an unsuspend event can reactivate it.
func (r *ProjectRepository) SetInstallationStatus(ctx context.Context, installationID int64, status string) error {
	if status != models.GitHubConnectionActive && status != models.GitHubConnectionSuspended {
		return fmt.Errorf("invalid github installation status %q", status)
	}
	fromStatus := models.GitHubConnectionSuspended
	if status == models.GitHubConnectionSuspended {
		fromStatus = models.GitHubConnectionActive
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects
		 SET github_connection_status = ?, updated_at = ?
		 WHERE github_installation_id = ? AND github_connection_status = ?`,
		status, time.Now().UTC().Format(time.RFC3339), installationID, fromStatus)
	if err != nil {
		return fmt.Errorf("set installation status: %w", err)
	}
	return nil
}

// SetRepositoryAccess marks matching projects as active or access-removed.
// The installation ID is retained so a later "added" event can reconnect the
// project without guessing which installation previously owned the mapping.
func (r *ProjectRepository) SetRepositoryAccess(ctx context.Context, installationID int64, repos []string, granted bool) error {
	if len(repos) == 0 {
		return nil
	}
	status := models.GitHubConnectionAccessRemoved
	if granted {
		status = models.GitHubConnectionActive
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, repo := range repos {
		query := `UPDATE projects
			 SET github_connection_status = ?, updated_at = ?
			 WHERE github_installation_id = ? AND LOWER(github_repo) = LOWER(?)`
		if granted {
			query += ` AND github_connection_status <> 'suspended'`
		}
		if _, err := r.db.ExecContext(ctx,
			query,
			status, now, installationID, repo); err != nil {
			return fmt.Errorf("set repository access for %q: %w", repo, err)
		}
	}
	return nil
}

// UpdateGitHubRepositoryIdentity follows a repository across rename or
// transfer. Repository ID is preferred because the full name changes; the old
// full name is retained as a migration fallback for projects created before ID
// persistence existed.
func (r *ProjectRepository) UpdateGitHubRepositoryIdentity(
	ctx context.Context,
	installationID, repositoryID int64,
	oldFullName, newFullName string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin github repository identity update: %w", err)
	}
	defer tx.Rollback()

	// Repository names are mutable locators. Update historical source snapshots
	// for the same stable repository identity so exact rebuild remains possible
	// after rename/transfer; commit and build recipe fields remain unchanged.
	if _, err := tx.ExecContext(ctx,
		`UPDATE deployments
		 SET source_repository = ?, source_installation_id = ?
		 WHERE project_id IN (
		     SELECT id FROM projects
		     WHERE github_repository_id = ?
		        OR (github_repository_id IS NULL AND LOWER(github_repo) = LOWER(?))
		 )
		   AND LOWER(source_repository) = LOWER(?)`,
		newFullName, installationID, repositoryID, oldFullName, oldFullName); err != nil {
		return fmt.Errorf("update deployment github source identity: %w", err)
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE projects
		 SET github_repo = ?, github_installation_id = ?, github_repository_id = ?,
		     github_connection_status = ?, updated_at = ?
		 WHERE (github_repository_id = ? OR (github_repository_id IS NULL AND LOWER(github_repo) = LOWER(?)))`,
		newFullName, installationID, repositoryID, models.GitHubConnectionActive,
		time.Now().UTC().Format(time.RFC3339), repositoryID, oldFullName)
	if err != nil {
		return fmt.Errorf("update github repository identity: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("read github repository identity update result: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit github repository identity update: %w", err)
	}
	return nil
}

// RenameInstallationOwner updates legacy owner/repository strings after the
// account hosting an installation is renamed.
func (r *ProjectRepository) RenameInstallationOwner(ctx context.Context, installationID int64, oldOwner, newOwner string) error {
	oldPrefix := oldOwner + "/"
	newPrefix := newOwner + "/"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin github installation owner rename: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE deployments
		 SET source_repository = ? || SUBSTR(source_repository, LENGTH(?) + 1)
		 WHERE project_id IN (
		     SELECT id FROM projects WHERE github_installation_id = ?
		 )
		   AND LOWER(source_repository) LIKE LOWER(?) || '%'`,
		newPrefix, oldPrefix, installationID, oldPrefix); err != nil {
		return fmt.Errorf("rename deployment github source owner: %w", err)
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE projects
		 SET github_repo = ? || SUBSTR(github_repo, LENGTH(?) + 1), updated_at = ?
		 WHERE github_installation_id = ? AND LOWER(github_repo) LIKE LOWER(?) || '%'`,
		newPrefix, oldPrefix, time.Now().UTC().Format(time.RFC3339), installationID, oldPrefix)
	if err != nil {
		return fmt.Errorf("rename github installation owner: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit github installation owner rename: %w", err)
	}
	return nil
}
