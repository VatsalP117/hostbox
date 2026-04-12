package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// PushPayload is the GitHub push webhook payload.
type PushPayload struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
	Deleted bool `json:"deleted"`
	Created bool `json:"created"`
}

type PushHandler struct {
	projectRepo   ProjectRepository
	deploymentSvc DeploymentCreator
	logger        *slog.Logger
}

func NewPushHandler(
	projectRepo ProjectRepository,
	deploymentSvc DeploymentCreator,
	logger *slog.Logger,
) *PushHandler {
	return &PushHandler{
		projectRepo:   projectRepo,
		deploymentSvc: deploymentSvc,
		logger:        logger,
	}
}

func (h *PushHandler) Handle(ctx context.Context, payload []byte, deliveryID string) error {
	var event PushPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal push payload: %w", err)
	}

	if event.Deleted {
		h.logger.Debug("ignoring branch deletion push", "ref", event.Ref)
		return nil
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if branch == event.Ref {
		return nil
	}

	repoFullName := event.Repository.FullName
	installationID := event.Installation.ID

	project, err := h.projectRepo.GetByGitHubRepo(ctx, repoFullName)
	if err != nil {
		h.logger.Debug("no project found for repo", "repo", repoFullName)
		return nil
	}

	if !project.AutoDeploy {
		h.logger.Debug("auto_deploy disabled, skipping",
			"project_id", project.ID,
			"repo", repoFullName,
		)
		return nil
	}

	isProduction := branch == project.ProductionBranch

	if !isProduction && !project.PreviewDeployments {
		h.logger.Debug("preview_deployments disabled, skipping non-production push",
			"project_id", project.ID,
			"branch", branch,
		)
		return nil
	}

	existing, _ := h.deploymentSvc.FindByCommitSHA(ctx, project.ID, event.After)
	if existing != nil {
		h.logger.Debug("deployment already exists for commit",
			"project_id", project.ID,
			"commit", event.After,
		)
		return nil
	}

	h.logger.Info("creating deployment from push",
		"project_id", project.ID,
		"branch", branch,
		"commit", event.After,
		"is_production", isProduction,
	)

	_, err = h.deploymentSvc.CreateFromWebhook(ctx, WebhookTriggerParams{
		ProjectID:      project.ID,
		Branch:         branch,
		CommitSHA:      event.After,
		CommitMessage:  event.HeadCommit.Message,
		CommitAuthor:   event.HeadCommit.Author.Name,
		IsProduction:   isProduction,
		InstallationID: installationID,
	})
	return err
}
