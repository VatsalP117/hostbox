package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const commentMarker = "<!-- hostbox-preview-deployment -->"

// PRCommentManager handles creating and updating Hostbox's PR comments.
type PRCommentManager struct {
	client         *Client
	platformDomain string
	logger         *slog.Logger
}

func NewPRCommentManager(client *Client, platformDomain string, logger *slog.Logger) *PRCommentManager {
	return &PRCommentManager{
		client:         client,
		platformDomain: platformDomain,
		logger:         logger,
	}
}

// DeploymentInfo contains the data needed to render a PR comment.
type DeploymentInfo struct {
	ProjectName   string
	ProjectSlug   string
	DeploymentID  string
	CommitSHA     string
	CommitMessage string
	Branch        string
	Status        string // "building", "ready", "failed"
	DeploymentURL string
	BuildDuration string
	LogURL        string
	ErrorMessage  string
}

// PostOrUpdate creates or updates the Hostbox preview deployment comment on a PR.
func (m *PRCommentManager) PostOrUpdate(
	ctx context.Context,
	installationID int64,
	owner, repo string,
	prNumber int,
	deployment DeploymentInfo,
) error {
	body := m.buildCommentBody(deployment)

	comments, err := m.client.ListPRComments(ctx, installationID, owner, repo, prNumber)
	if err != nil {
		return fmt.Errorf("list PR comments: %w", err)
	}

	for _, c := range comments {
		if strings.Contains(c.Body, commentMarker) {
			m.logger.Debug("updating existing PR comment",
				"comment_id", c.ID,
				"pr_number", prNumber,
			)
			return m.client.UpdateComment(ctx, installationID, owner, repo, c.ID, body)
		}
	}

	m.logger.Debug("creating new PR comment", "pr_number", prNumber)
	_, err = m.client.CreatePRComment(ctx, installationID, owner, repo, prNumber, body)
	return err
}

func (m *PRCommentManager) buildCommentBody(d DeploymentInfo) string {
	var sb strings.Builder

	sb.WriteString(commentMarker)
	sb.WriteString("\n")

	switch d.Status {
	case "ready":
		sb.WriteString("## 🚀 Preview Deployment Ready\n\n")
		sb.WriteString("| Name | Status | Preview |\n")
		sb.WriteString("|------|--------|------|\n")
		sb.WriteString(fmt.Sprintf("| **%s** | ✅ Ready | [Visit Preview](%s) |\n\n", d.ProjectName, d.DeploymentURL))
		sb.WriteString(fmt.Sprintf("**Commit**: `%.7s` — %s\n", d.CommitSHA, firstLine(d.CommitMessage)))
		sb.WriteString(fmt.Sprintf("**Built in**: %s\n", d.BuildDuration))
	case "building":
		sb.WriteString("## ⏳ Preview Deployment Building\n\n")
		sb.WriteString("| Name | Status |\n")
		sb.WriteString("|------|--------|\n")
		sb.WriteString(fmt.Sprintf("| **%s** | 🔨 Building... |\n\n", d.ProjectName))
		sb.WriteString(fmt.Sprintf("**Commit**: `%.7s` — %s\n", d.CommitSHA, firstLine(d.CommitMessage)))
		if d.LogURL != "" {
			sb.WriteString(fmt.Sprintf("\n[View Build Logs](%s)\n", d.LogURL))
		}
	case "failed":
		sb.WriteString("## ❌ Preview Deployment Failed\n\n")
		sb.WriteString("| Name | Status |\n")
		sb.WriteString("|------|--------|\n")
		sb.WriteString(fmt.Sprintf("| **%s** | ❌ Failed |\n\n", d.ProjectName))
		sb.WriteString(fmt.Sprintf("**Commit**: `%.7s` — %s\n", d.CommitSHA, firstLine(d.CommitMessage)))
		if d.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("\n**Error**: %s\n", d.ErrorMessage))
		}
		if d.LogURL != "" {
			sb.WriteString(fmt.Sprintf("\n[View Build Logs](%s)\n", d.LogURL))
		}
	}

	sb.WriteString("\n---\n")
	sb.WriteString(fmt.Sprintf("*Deployed with [Hostbox](https://%s)*\n", m.platformDomain))

	return sb.String()
}

func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx != -1 {
		return s[:idx]
	}
	return s
}
