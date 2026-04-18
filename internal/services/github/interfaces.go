package github

import (
	"context"

	"github.com/VatsalP117/hostbox/internal/models"
)

// ProjectRepository defines the project data access methods needed by GitHub handlers.
type ProjectRepository interface {
	GetByGitHubRepo(ctx context.Context, repoFullName string) (*models.Project, error)
	ClearInstallation(ctx context.Context, installationID int64) error
}

// DeploymentCreator defines the deployment creation methods needed by webhook handlers.
type DeploymentCreator interface {
	FindByCommitSHA(ctx context.Context, projectID, commitSHA string) (*models.Deployment, error)
	CreateFromWebhook(ctx context.Context, params WebhookTriggerParams) (*models.Deployment, error)
	DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error)
}

// RouteRemover defines the Caddy route removal methods needed by PR close handler.
type RouteRemover interface {
	RemoveDeploymentRoute(ctx context.Context, deploymentID string) error
}

// WebhookTriggerParams contains parameters for creating a deployment from a webhook.
type WebhookTriggerParams struct {
	ProjectID      string
	Branch         string
	CommitSHA      string
	CommitMessage  string
	CommitAuthor   string
	IsProduction   bool
	GitHubPRNumber int
	InstallationID int64
}
