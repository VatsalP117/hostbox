package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// StatusReporter posts GitHub Deployment Statuses.
type StatusReporter struct {
	client         *Client
	platformDomain string
	logger         *slog.Logger
}

func NewStatusReporter(client *Client, platformDomain string, logger *slog.Logger) *StatusReporter {
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
	parts := strings.SplitN(info.Owner+"/"+info.Repo, "/", 2)
	owner, repo := parts[0], parts[1]

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
