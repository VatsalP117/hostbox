package caddy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/VatsalP117/hostbox/internal/models"
)

// PostBuildRouteHook implements worker.PostBuildHook to update Caddy routes after builds.
type PostBuildRouteHook struct {
	manager *RouteManager
	logger  *slog.Logger
}

func NewPostBuildRouteHook(manager *RouteManager, logger *slog.Logger) *PostBuildRouteHook {
	return &PostBuildRouteHook{
		manager: manager,
		logger:  logger,
	}
}

// OnBuildSuccess adds/updates Caddy routes for a successful deployment.
func (h *PostBuildRouteHook) OnBuildSuccess(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	framework := ""
	if project.Framework != nil {
		framework = *project.Framework
	}

	artifactPath := ""
	if deployment.ArtifactPath != nil {
		artifactPath = *deployment.ArtifactPath
	}

	activeDeploy := ActiveDeployment{
		DeploymentID: deployment.ID,
		ProjectID:    project.ID,
		ProjectSlug:  project.Slug,
		Branch:       deployment.Branch,
		BranchSlug:   Slugify(deployment.Branch),
		CommitSHA:    deployment.CommitSHA,
		IsProduction: deployment.IsProduction,
		ArtifactPath: artifactPath,
		Framework:    framework,
	}

	var routeErrs []error
	if err := h.manager.AddDeploymentRoute(ctx, activeDeploy); err != nil {
		routeErrs = append(routeErrs, fmt.Errorf("add deployment route: %w", err))
	}

	// If production, update production route
	if deployment.IsProduction {
		if err := h.manager.UpdateProductionRoute(ctx, project.Slug, project.ID, artifactPath, framework); err != nil {
			routeErrs = append(routeErrs, fmt.Errorf("update production route: %w", err))
		}
	}

	// Update branch-stable route
	branchSlug := Slugify(deployment.Branch)
	if err := h.manager.UpdateBranchRoute(ctx, project.Slug, project.ID, branchSlug, artifactPath, framework); err != nil {
		routeErrs = append(routeErrs, fmt.Errorf("update branch route: %w", err))
	}

	return errors.Join(routeErrs...)
}

// OnBuildFailure is a no-op for route management (failed builds don't get routes).
func (h *PostBuildRouteHook) OnBuildFailure(ctx context.Context, project *models.Project, deployment *models.Deployment, buildErr error) error {
	return nil
}

// OnBuildCancelled removes routes that may have been added immediately before
// a concurrent branch/PR cleanup marked the deployment cancelled.
func (h *PostBuildRouteHook) OnBuildCancelled(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	return errors.Join(
		h.manager.RemoveDeploymentRoute(ctx, deployment.ID),
		h.manager.RemoveBranchRoute(ctx, project.ID, deployment.Branch),
	)
}
