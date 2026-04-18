package caddy

import (
	"context"
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

	// Add preview route
	if err := h.manager.AddDeploymentRoute(ctx, activeDeploy); err != nil {
		h.logger.Error("failed to add deployment route", "error", err, "deployment_id", deployment.ID)
	}

	// If production, update production route
	if deployment.IsProduction {
		if err := h.manager.UpdateProductionRoute(ctx, project.Slug, project.ID, artifactPath, framework); err != nil {
			h.logger.Error("failed to update production route", "error", err, "project_id", project.ID)
		}
	}

	// Update branch-stable route
	branchSlug := Slugify(deployment.Branch)
	if err := h.manager.UpdateBranchRoute(ctx, project.Slug, project.ID, branchSlug, artifactPath, framework); err != nil {
		h.logger.Error("failed to update branch route", "error", err, "project_id", project.ID, "branch", deployment.Branch)
	}

	return nil
}

// OnBuildFailure is a no-op for route management (failed builds don't get routes).
func (h *PostBuildRouteHook) OnBuildFailure(ctx context.Context, project *models.Project, deployment *models.Deployment, buildErr error) error {
	return nil
}
