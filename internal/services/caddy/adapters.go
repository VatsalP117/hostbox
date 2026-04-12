package caddy

import (
	"context"

	"github.com/vatsalpatel/hostbox/internal/repository"
)

// DeploymentRepoAdapter adapts the DeploymentRepository to the DeploymentQuerier interface.
type DeploymentRepoAdapter struct {
	Repo *repository.DeploymentRepository
}

func (a *DeploymentRepoAdapter) ListActiveWithProject(ctx context.Context) ([]ActiveDeployment, error) {
	rows, err := a.Repo.ListActiveWithProject(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ActiveDeployment, len(rows))
	for i, r := range rows {
		framework := ""
		if r.Framework != nil {
			framework = *r.Framework
		}
		artifactPath := ""
		if r.ArtifactPath != nil {
			artifactPath = *r.ArtifactPath
		}
		result[i] = ActiveDeployment{
			DeploymentID: r.DeploymentID,
			ProjectID:    r.ProjectID,
			ProjectSlug:  r.ProjectSlug,
			Branch:       r.Branch,
			BranchSlug:   Slugify(r.Branch),
			CommitSHA:    r.CommitSHA,
			IsProduction: r.IsProduction,
			ArtifactPath: artifactPath,
			Framework:    framework,
		}
	}
	return result, nil
}

// DomainRepoAdapter adapts the DomainRepository to the DomainQuerier interface.
type DomainRepoAdapter struct {
	Repo *repository.DomainRepository
}

func (a *DomainRepoAdapter) ListVerifiedWithProject(ctx context.Context) ([]VerifiedDomain, error) {
	rows, err := a.Repo.ListVerifiedWithProject(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]VerifiedDomain, len(rows))
	for i, r := range rows {
		framework := ""
		if r.Framework != nil {
			framework = *r.Framework
		}
		productionArtifact := ""
		if r.ProductionArtifact != nil {
			productionArtifact = *r.ProductionArtifact
		}
		result[i] = VerifiedDomain{
			DomainID:           r.DomainID,
			Domain:             r.Domain,
			ProjectID:          r.ProjectID,
			ProjectSlug:        r.ProjectSlug,
			ProductionArtifact: productionArtifact,
			Framework:          framework,
		}
	}
	return result, nil
}
