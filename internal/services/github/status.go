package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// StatusReporter posts GitHub Deployment Statuses.
type StatusReporter struct {
	client         DeploymentStatusClient
	platformDomain string
	logger         *slog.Logger
}

// DeploymentStatusClient is the subset of the GitHub API used for deployment
// lifecycle reporting. Keeping this boundary small makes lifecycle reporting
// independently testable and allows callers to obtain clients dynamically.
type DeploymentStatusClient interface {
	CreateDeployment(context.Context, int64, string, string, CreateDeploymentRequest) (*DeploymentResponse, error)
	CreateDeploymentStatus(context.Context, int64, string, string, int64, CreateDeploymentStatusRequest) error
}

func NewStatusReporter(client DeploymentStatusClient, platformDomain string, logger *slog.Logger) *StatusReporter {
	return &StatusReporter{
		client:         client,
		platformDomain: platformDomain,
		logger:         logger,
	}
}

// DeploymentStatusInfo contains info needed to report to GitHub.
type DeploymentStatusInfo struct {
	InstallationID int64
	Owner          string
	Repo           string
	CommitSHA      string
	Environment    string // "production" or "preview"
	Status         string // Hostbox status: "queued", "building", "ready", "failed", "cancelled"
	DeploymentURL  string
	LogURL         string
	Description    string
	GitHubDeployID int64 // 0 = create new, >0 = update existing
}

// mapStatus converts Hostbox deployment status to GitHub Deployment Status state.
func mapStatus(hostboxStatus string) string {
	switch hostboxStatus {
	case "queued":
		return "pending"
	case "building":
		return "in_progress"
	case "ready":
		return "success"
	case "failed":
		return "failure"
	case "cancelled":
		return "error"
	default:
		return "error"
	}
}

// ReportStatus creates a GitHub Deployment and posts a status update.
// If info.GitHubDeployID is 0, creates a new GitHub Deployment first.
// Returns the GitHub Deployment ID (to be stored for subsequent updates).
func (r *StatusReporter) ReportStatus(ctx context.Context, info DeploymentStatusInfo) (int64, error) {
	owner, repo, err := parseRepository(info.Owner + "/" + info.Repo)
	if err != nil {
		return 0, err
	}
	if info.InstallationID <= 0 {
		return 0, fmt.Errorf("github installation id is required")
	}
	if strings.TrimSpace(info.CommitSHA) == "" {
		return 0, fmt.Errorf("deployment commit sha is required")
	}

	deployID := info.GitHubDeployID
	if deployID == 0 {
		shortSHA := info.CommitSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		resp, err := r.client.CreateDeployment(ctx, info.InstallationID, owner, repo, CreateDeploymentRequest{
			Ref:              info.CommitSHA,
			Task:             "deploy",
			AutoMerge:        false,
			RequiredContexts: []string{},
			Environment:      info.Environment,
			Description:      fmt.Sprintf("Hostbox deployment for %s", shortSHA),
		})
		if err != nil {
			return 0, fmt.Errorf("create github deployment: %w", err)
		}
		deployID = resp.ID
	}

	statusReq := CreateDeploymentStatusRequest{
		State:        mapStatus(info.Status),
		Description:  info.Description,
		AutoInactive: true,
	}
	if info.DeploymentURL != "" {
		statusReq.EnvironmentURL = info.DeploymentURL
	}
	if info.LogURL != "" {
		statusReq.LogURL = info.LogURL
	}

	if err := r.client.CreateDeploymentStatus(ctx, info.InstallationID, owner, repo, deployID, statusReq); err != nil {
		return deployID, fmt.Errorf("create github deployment status: %w", err)
	}

	r.logger.Info("reported github deployment status",
		"github_deploy_id", deployID,
		"status", info.Status,
		"environment", info.Environment,
	)

	return deployID, nil
}

func parseRepository(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid github repository %q: expected owner/repo", fullName)
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" || owner == "." || owner == ".." || repo == "." || repo == ".." {
		return "", "", fmt.Errorf("invalid github repository %q: expected owner/repo", fullName)
	}
	return owner, repo, nil
}
