package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/VatsalP117/hostbox/internal/models"
)

// PullRequestPayload is the GitHub pull_request webhook payload.
type PullRequestPayload struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Head   struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Merged bool `json:"merged"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

type PullRequestHandler struct {
	projectRepo   ProjectRepository
	deploymentSvc DeploymentCreator
	routeManager  RouteRemover
	logger        *slog.Logger
}

func NewPullRequestHandler(
	projectRepo ProjectRepository,
	deploymentSvc DeploymentCreator,
	routeManager RouteRemover,
	logger *slog.Logger,
) *PullRequestHandler {
	return &PullRequestHandler{
		projectRepo:   projectRepo,
		deploymentSvc: deploymentSvc,
		routeManager:  routeManager,
		logger:        logger,
	}
}

func (h *PullRequestHandler) Handle(ctx context.Context, payload []byte, deliveryID string) error {
	var event PullRequestPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal pull_request payload: %w", err)
	}

	repoFullName := event.Repository.FullName

	project, err := h.projectRepo.GetByGitHubRepo(ctx, repoFullName)
	if err != nil {
		return nil
	}

	switch event.Action {
	case "opened", "synchronize", "reopened":
		return h.handleOpenedOrSync(ctx, project, event)
	case "closed":
		return h.handleClosed(ctx, project, event)
	default:
		h.logger.Debug("ignoring pull_request action", "action", event.Action)
		return nil
	}
}

func (h *PullRequestHandler) handleOpenedOrSync(ctx context.Context, project *models.Project, event PullRequestPayload) error {
	if !project.PreviewDeployments {
		return nil
	}

	branch := event.PullRequest.Head.Ref
	commitSHA := event.PullRequest.Head.SHA

	existing, _ := h.deploymentSvc.FindByCommitSHA(ctx, project.ID, commitSHA)
	if existing != nil {
		return nil
	}

	h.logger.Info("creating preview deployment from PR",
		"project_id", project.ID,
		"pr_number", event.Number,
		"branch", branch,
		"commit", commitSHA,
	)

	_, err := h.deploymentSvc.CreateFromWebhook(ctx, WebhookTriggerParams{
		ProjectID:      project.ID,
		Branch:         branch,
		CommitSHA:      commitSHA,
		CommitMessage:  event.PullRequest.Title,
		CommitAuthor:   "",
		IsProduction:   false,
		GitHubPRNumber: event.Number,
		InstallationID: event.Installation.ID,
	})
	return err
}

func (h *PullRequestHandler) handleClosed(ctx context.Context, project *models.Project, event PullRequestPayload) error {
	branch := event.PullRequest.Head.Ref

	h.logger.Info("PR closed, deactivating preview deployments",
		"project_id", project.ID,
		"pr_number", event.Number,
		"branch", branch,
	)

	deployments, err := h.deploymentSvc.DeactivateBranchDeployments(ctx, project.ID, branch)
	if err != nil {
		return fmt.Errorf("deactivate branch deployments: %w", err)
	}

	for _, d := range deployments {
		if err := h.routeManager.RemoveDeploymentRoute(ctx, d.ID); err != nil {
			h.logger.Error("failed to remove deployment route", "deployment_id", d.ID, "error", err)
		}
	}

	return nil
}
