package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type DomainRepository struct {
	db *sql.DB
}

func NewDomainRepository(db *sql.DB) *DomainRepository {
	return &DomainRepository{db: db}
}

func (r *DomainRepository) Create(ctx context.Context, domain *models.Domain) error {
	if domain.ID == "" {
		domain.ID = util.NewID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO domains (id, project_id, domain, verified, verified_at, last_checked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		domain.ID, domain.ProjectID, domain.Domain, domain.Verified,
		formatNullableTime(domain.VerifiedAt),
		formatNullableTime(domain.LastCheckedAt),
		now,
	)
	if err != nil {
		return fmt.Errorf("create domain: %w", err)
	}
	domain.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *DomainRepository) GetByID(ctx context.Context, id string) (*models.Domain, error) {
	row := r.db.QueryRowContext(ctx, domainSelectSQL+` WHERE id = ?`, id)
	return scanDomain(row)
}

func (r *DomainRepository) GetByDomain(ctx context.Context, domain string) (*models.Domain, error) {
	row := r.db.QueryRowContext(ctx, domainSelectSQL+` WHERE domain = ?`, domain)
	return scanDomain(row)
}

func (r *DomainRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM domains WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DomainRepository) ListByProject(ctx context.Context, projectID string) ([]models.Domain, error) {
	rows, err := r.db.QueryContext(ctx, domainSelectSQL+` WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	defer rows.Close()

	var domains []models.Domain
	for rows.Next() {
		d, err := scanDomainRows(rows)
		if err != nil {
			return nil, err
		}
		domains = append(domains, *d)
	}
	return domains, rows.Err()
}

func (r *DomainRepository) UpdateVerification(ctx context.Context, id string, verified bool, verifiedAt *time.Time) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE domains SET verified = ?, verified_at = ? WHERE id = ?`,
		verified, formatNullableTime(verifiedAt), id,
	)
	if err != nil {
		return fmt.Errorf("update domain verification: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DomainRepository) UpdateLastChecked(ctx context.Context, id string, lastCheckedAt time.Time) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE domains SET last_checked_at = ? WHERE id = ?`,
		lastCheckedAt.UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("update domain last checked: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DomainRepository) ListUnverified(ctx context.Context) ([]models.Domain, error) {
	rows, err := r.db.QueryContext(ctx, domainSelectSQL+` WHERE verified = 0 ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list unverified domains: %w", err)
	}
	defer rows.Close()

	var domains []models.Domain
	for rows.Next() {
		d, err := scanDomainRows(rows)
		if err != nil {
			return nil, err
		}
		domains = append(domains, *d)
	}
	return domains, rows.Err()
}

const domainSelectSQL = `SELECT id, project_id, domain, verified, verified_at, last_checked_at, created_at FROM domains`

func scanDomain(s scanner) (*models.Domain, error) {
	var d models.Domain
	var verifiedAt, lastCheckedAt, createdAt sql.NullString
	err := s.Scan(&d.ID, &d.ProjectID, &d.Domain, &d.Verified, &verifiedAt, &lastCheckedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	if verifiedAt.Valid {
		t, _ := time.Parse(time.RFC3339, verifiedAt.String)
		d.VerifiedAt = &t
	}
	if lastCheckedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastCheckedAt.String)
		d.LastCheckedAt = &t
	}
	if createdAt.Valid {
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	return &d, nil
}

func scanDomainRows(rows *sql.Rows) (*models.Domain, error) {
	return scanDomain(rows)
}

// ListVerifiedWithProject returns verified domains joined with project info for Caddy sync.
func (r *DomainRepository) ListVerifiedWithProject(ctx context.Context) ([]VerifiedDomainRow, error) {
	query := `SELECT dom.id, dom.domain, dom.project_id, p.slug, p.framework,
		(SELECT d.artifact_path FROM deployments d
		 WHERE d.project_id = p.id AND d.status = 'ready' AND d.is_production = 1
		 ORDER BY d.created_at DESC LIMIT 1) AS production_artifact
		FROM domains dom
		JOIN projects p ON dom.project_id = p.id
		WHERE dom.verified = 1`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list verified domains: %w", err)
	}
	defer rows.Close()

	var result []VerifiedDomainRow
	for rows.Next() {
		var row VerifiedDomainRow
		if err := rows.Scan(&row.DomainID, &row.Domain, &row.ProjectID, &row.ProjectSlug,
			&row.Framework, &row.ProductionArtifact); err != nil {
			return nil, fmt.Errorf("scan verified domain: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// VerifiedDomainRow is a denormalized row for Caddy config building.
type VerifiedDomainRow struct {
	DomainID           string
	Domain             string
	ProjectID          string
	ProjectSlug        string
	Framework          *string
	ProductionArtifact *string
}
