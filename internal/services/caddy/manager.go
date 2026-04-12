package caddy

import (
	"context"
	"fmt"
	"log/slog"
)

// RouteManager handles high-level route lifecycle operations.
type RouteManager struct {
	client  *CaddyClient
	builder *ConfigBuilder
	logger  *slog.Logger
}

func NewRouteManager(client *CaddyClient, builder *ConfigBuilder, logger *slog.Logger) *RouteManager {
	return &RouteManager{
		client:  client,
		builder: builder,
		logger:  logger,
	}
}

// AddDeploymentRoute adds a preview route for a newly-ready deployment.
func (m *RouteManager) AddDeploymentRoute(ctx context.Context, d ActiveDeployment) error {
	route := m.builder.buildPreviewRoute(d)
	m.logger.Info("adding deployment route",
		"deployment_id", d.DeploymentID,
		"host", route.Match[0].Host[0],
	)
	return m.client.AddRoute(ctx, "main", route)
}

// UpdateProductionRoute sets or updates the production route for a project.
func (m *RouteManager) UpdateProductionRoute(ctx context.Context, projectSlug, projectID, artifactPath, framework string) error {
	route := m.builder.buildProductionRoute(projectSlug, projectID, artifactPath, framework)

	// Remove existing first (ignore errors for non-existent)
	_ = m.client.DeleteRoute(ctx, fmt.Sprintf("route-prod-%s", projectID))

	m.logger.Info("setting production route",
		"project_id", projectID,
		"host", route.Match[0].Host[0],
	)
	return m.client.AddRoute(ctx, "main", route)
}

// UpdateBranchRoute sets or updates the branch-stable route.
func (m *RouteManager) UpdateBranchRoute(ctx context.Context, projectSlug, projectID, branchSlug, artifactPath, framework string) error {
	routeID := fmt.Sprintf("route-branch-%s-%s", projectID, branchSlug)
	_ = m.client.DeleteRoute(ctx, routeID)

	route := m.builder.buildBranchStableRoute(projectSlug, projectID, branchSlug, artifactPath, framework)
	m.logger.Info("setting branch route",
		"project_id", projectID,
		"branch", branchSlug,
		"host", route.Match[0].Host[0],
	)
	return m.client.AddRoute(ctx, "main", route)
}

// AddCustomDomainRoute adds a route for a verified custom domain.
func (m *RouteManager) AddCustomDomainRoute(ctx context.Context, d VerifiedDomain) error {
	route := m.builder.buildCustomDomainRoute(d)
	m.logger.Info("adding custom domain route",
		"domain_id", d.DomainID,
		"domain", d.Domain,
	)
	return m.client.AddRoute(ctx, "main", route)
}

// RemoveCustomDomainRoute removes a custom domain route.
func (m *RouteManager) RemoveCustomDomainRoute(ctx context.Context, domainID string) error {
	routeID := fmt.Sprintf("route-domain-%s", domainID)
	m.logger.Info("removing custom domain route", "domain_id", domainID)
	return m.client.DeleteRoute(ctx, routeID)
}

// RemoveDeploymentRoute removes a single deployment's preview route.
func (m *RouteManager) RemoveDeploymentRoute(ctx context.Context, deploymentID string) error {
	routeID := fmt.Sprintf("route-deploy-%s", deploymentID)
	m.logger.Info("removing deployment route", "deployment_id", deploymentID)
	return m.client.DeleteRoute(ctx, routeID)
}

// RemoveAllProjectRoutes removes all routes for a project.
func (m *RouteManager) RemoveAllProjectRoutes(ctx context.Context, projectID string, deploymentIDs []string, branchSlugs []string, domainIDs []string) error {
	_ = m.client.DeleteRoute(ctx, fmt.Sprintf("route-prod-%s", projectID))

	for _, slug := range branchSlugs {
		_ = m.client.DeleteRoute(ctx, fmt.Sprintf("route-branch-%s-%s", projectID, slug))
	}

	for _, dID := range deploymentIDs {
		_ = m.client.DeleteRoute(ctx, fmt.Sprintf("route-deploy-%s", dID))
	}

	for _, domID := range domainIDs {
		_ = m.client.DeleteRoute(ctx, fmt.Sprintf("route-domain-%s", domID))
	}

	m.logger.Info("removed all project routes", "project_id", projectID)
	return nil
}
