package github

import (
	"context"

	"github.com/VatsalP117/hostbox/internal/models"
)

// ProjectRepository defines the project data access methods needed by GitHub handlers.
type ProjectRepository interface {
	ListByGitHubSource(ctx context.Context, installationID int64, repoFullName string) ([]models.Project, error)
	ClearInstallation(ctx context.Context, installationID int64) error
	SetInstallationStatus(ctx context.Context, installationID int64, status string) error
	SetRepositoryAccess(ctx context.Context, installationID int64, repos []string, granted bool) error
	UpdateGitHubRepositoryIdentity(ctx context.Context, installationID, repositoryID int64, oldFullName, newFullName string) error
	RenameInstallationOwner(ctx context.Context, installationID int64, oldOwner, newOwner string) error
}

// DeploymentCreator defines the deployment creation methods needed by webhook handlers.
type DeploymentCreator interface {
	FindByCommitSHAAndBranch(ctx context.Context, projectID, commitSHA, branch string) (*models.Deployment, error)
	AssociatePullRequest(ctx context.Context, deployment *models.Deployment, prNumber int) error
	CreateFromWebhook(ctx context.Context, params WebhookTriggerParams) (*models.Deployment, error)
	DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error)
}

// RouteRemover defines the Caddy route removal methods needed by PR close handler.
type RouteRemover interface {
	RemoveDeploymentRoute(ctx context.Context, deploymentID string) error
	RemoveBranchRoute(ctx context.Context, projectID, branch string) error
}

type InstallationTokenInvalidator interface {
	InvalidateInstallationToken(installationID int64)
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
