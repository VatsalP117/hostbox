package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/platform/hostnames"
)

// FeedbackClient is the GitHub API surface needed to publish deployment and
// pull-request lifecycle feedback.
type FeedbackClient interface {
	DeploymentStatusClient
	PRCommentClient
}

// FeedbackClientProvider returns the currently configured GitHub client. It is
// deliberately dynamic because the GitHub App can be configured after startup.
type FeedbackClientProvider interface {
	FeedbackClient() (FeedbackClient, error)
}

// FeedbackClientProviderFunc adapts a function into a client provider.
type FeedbackClientProviderFunc func() (FeedbackClient, error)

func (f FeedbackClientProviderFunc) FeedbackClient() (FeedbackClient, error) {
	return f()
}

// GitHubDeploymentStore persists the stable GitHub Deployment ID reused by all
// lifecycle status updates for one Hostbox deployment.
type GitHubDeploymentStore interface {
	SetGitHubDeployIDIfUnset(context.Context, string, int64) (int64, error)
}

type githubDeploymentReader interface {
	GetByID(context.Context, string) (*models.Deployment, error)
}

// LifecycleReporter synchronously publishes GitHub feedback with bounded
// retries. Callers decide whether an exhausted reporting failure affects the
// primary deployment operation.
type LifecycleReporter struct {
	clients          FeedbackClientProvider
	deployments      GitHubDeploymentStore
	dashboardBaseURL string
	dashboardDomain  string
	platformDomain   string
	logger           *slog.Logger
	mu               sync.Mutex
}

func NewLifecycleReporter(
	clients FeedbackClientProvider,
	deployments GitHubDeploymentStore,
	dashboardBaseURL string,
	logger *slog.Logger,
	platformDomain ...string,
) (*LifecycleReporter, error) {
	if clients == nil {
		return nil, fmt.Errorf("github feedback client provider is required")
	}
	if deployments == nil {
		return nil, fmt.Errorf("github deployment store is required")
	}
	base, err := url.Parse(strings.TrimRight(dashboardBaseURL, "/"))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("invalid dashboard base url %q", dashboardBaseURL)
	}
	if logger == nil {
		logger = slog.Default()
	}
	reporter := &LifecycleReporter{
		clients:          clients,
		deployments:      deployments,
		dashboardBaseURL: strings.TrimRight(base.String(), "/"),
		dashboardDomain:  base.Host,
		logger:           logger,
	}
	if len(platformDomain) > 0 {
		reporter.platformDomain = strings.Trim(strings.TrimSpace(platformDomain[0]), ".")
	}
	return reporter, nil
}

// Report publishes the deployment's current state. Projects without GitHub
// metadata are intentionally ignored so manual and disconnected projects keep
// working. Partial GitHub metadata is rejected rather than sent to an ambiguous
// repository or installation.
func (r *LifecycleReporter) Report(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<(attempt-1)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return errors.Join(lastErr, ctx.Err())
			case <-time.After(delay):
			}
		}
		if err := r.reportOnce(ctx, project, deployment); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("github lifecycle feedback failed after 3 attempts: %w", lastErr)
}

func (r *LifecycleReporter) reportOnce(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	if project == nil || deployment == nil {
		return fmt.Errorf("project and deployment are required for github feedback")
	}
	if project.GitHubRepo == nil && project.GitHubInstallationID == nil {
		return nil
	}
	if project.GitHubRepo == nil || strings.TrimSpace(*project.GitHubRepo) == "" {
		return fmt.Errorf("github repository is required for feedback")
	}
	if project.GitHubInstallationID == nil || *project.GitHubInstallationID <= 0 {
		return fmt.Errorf("github installation id is required for feedback")
	}
	owner, repo, err := parseRepository(*project.GitHubRepo)
	if err != nil {
		return err
	}
	if strings.TrimSpace(deployment.CommitSHA) == "" {
		return fmt.Errorf("deployment commit sha is required for github feedback")
	}

	// Serialize the reload and publication together. PR association and build
	// finalization use different in-memory snapshots, so reloading only in the
	// callers could otherwise let an older building report overwrite ready.
	r.mu.Lock()
	defer r.mu.Unlock()
	callerDeployment := deployment
	if reader, ok := r.deployments.(githubDeploymentReader); ok {
		current, reloadErr := reader.GetByID(ctx, deployment.ID)
		if reloadErr != nil {
			return fmt.Errorf("reload deployment for github feedback: %w", reloadErr)
		}
		deployment = current
		defer func() { *callerDeployment = *deployment }()
	}

	client, err := r.clients.FeedbackClient()
	if err != nil {
		return fmt.Errorf("get github feedback client: %w", err)
	}
	if client == nil {
		return fmt.Errorf("get github feedback client: provider returned nil client")
	}

	dashboardURL := r.deploymentDashboardURL(project.ID, deployment.ID)
	deployURL := stringValue(deployment.DeploymentURL)
	environment := "preview"
	if deployment.IsProduction {
		environment = "production"
	}

	githubDeployID := int64(0)
	if deployment.GitHubDeployID != nil {
		githubDeployID = *deployment.GitHubDeployID
	}
	statusReporter := NewStatusReporter(client, "", r.logger)
	createdID, statusErr := statusReporter.ReportStatus(ctx, DeploymentStatusInfo{
		InstallationID: *project.GitHubInstallationID,
		Owner:          owner,
		Repo:           repo,
		CommitSHA:      deployment.CommitSHA,
		Environment:    environment,
		Status:         string(deployment.Status),
		DeploymentURL:  deployURL,
		LogURL:         dashboardURL,
		Description:    statusDescription(deployment.Status),
		GitHubDeployID: githubDeployID,
	})
	if githubDeployID == 0 && createdID > 0 {
		storedID, persistErr := r.deployments.SetGitHubDeployIDIfUnset(ctx, deployment.ID, createdID)
		if persistErr != nil {
			return errors.Join(statusErr, fmt.Errorf("persist github deployment id: %w", persistErr))
		}
		deployment.GitHubDeployID = &storedID
	}
	if statusErr != nil {
		return statusErr
	}

	if deployment.IsProduction || deployment.GitHubPRNumber == nil {
		return nil
	}
	comment := NewPRCommentManager(client, r.dashboardDomain, r.logger)
	branchURL := ""
	if r.platformDomain != "" {
		branchURL = "https://" + hostnames.BranchHost(project.Slug, deployment.Branch, r.platformDomain)
	}
	return comment.PostOrUpdate(ctx, *project.GitHubInstallationID, owner, repo, *deployment.GitHubPRNumber, DeploymentInfo{
		ProjectName:   project.Name,
		ProjectSlug:   project.Slug,
		DeploymentID:  deployment.ID,
		CommitSHA:     deployment.CommitSHA,
		CommitMessage: stringValue(deployment.CommitMessage),
		Branch:        deployment.Branch,
		Status:        string(deployment.Status),
		DeploymentURL: deployURL,
		BranchURL:     branchURL,
		BuildDuration: formatBuildDuration(deployment.BuildDurationMs),
		LogURL:        dashboardURL,
		ErrorMessage:  stringValue(deployment.ErrorMessage),
	})
}

func (r *LifecycleReporter) deploymentDashboardURL(projectID, deploymentID string) string {
	base, _ := url.Parse(r.dashboardBaseURL)
	base.Path = strings.TrimRight(base.Path, "/") + "/projects/" + url.PathEscape(projectID) + "/deployments/" + url.PathEscape(deploymentID)
	return base.String()
}

func statusDescription(status models.DeploymentStatus) string {
	switch status {
	case models.DeploymentStatusQueued:
		return "Deployment queued"
	case models.DeploymentStatusBuilding:
		return "Deployment building"
	case models.DeploymentStatusReady:
		return "Deployment ready"
	case models.DeploymentStatusFailed:
		return "Deployment failed"
	case models.DeploymentStatusCancelled:
		return "Deployment cancelled"
	default:
		return "Deployment status unknown"
	}
}

func formatBuildDuration(milliseconds *int64) string {
	if milliseconds == nil || *milliseconds < 0 {
		return ""
	}
	return (time.Duration(*milliseconds) * time.Millisecond).Round(time.Millisecond).String()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
