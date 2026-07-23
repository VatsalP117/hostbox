package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/VatsalP117/hostbox/internal/models"
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
	routeManager  RouteRemover
	logger        *slog.Logger
}

func NewPushHandler(
	projectRepo ProjectRepository,
	deploymentSvc DeploymentCreator,
	routeManager RouteRemover,
	logger *slog.Logger,
) *PushHandler {
	return &PushHandler{
		projectRepo:   projectRepo,
		deploymentSvc: deploymentSvc,
		routeManager:  routeManager,
		logger:        logger,
	}
}

func (h *PushHandler) Handle(ctx context.Context, payload []byte, deliveryID string) error {
	var event PushPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal push payload: %w", err)
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if branch == event.Ref {
		return nil
	}

	repoFullName := event.Repository.FullName
	installationID := event.Installation.ID

	projects, err := h.projectRepo.ListByGitHubSource(ctx, installationID, repoFullName)
	if err != nil {
		return fmt.Errorf("list projects for github source: %w", err)
	}
	if len(projects) == 0 {
		h.logger.Debug("no active project found for github source",
			"repo", repoFullName,
			"installation_id", installationID,
		)
		return nil
	}

	var eventErrors []error
	for i := range projects {
		project := &projects[i]
		if event.Deleted {
			if err := h.cleanupDeletedBranch(ctx, project.ID, branch); err != nil {
				eventErrors = append(eventErrors, fmt.Errorf("project %s: %w", project.ID, err))
			}
			continue
		}
		if err := h.deployPush(ctx, project, event, branch, installationID); err != nil {
			eventErrors = append(eventErrors, fmt.Errorf("project %s: %w", project.ID, err))
		}
	}
	return errors.Join(eventErrors...)
}

func (h *PushHandler) deployPush(
	ctx context.Context,
	project *models.Project,
	event PushPayload,
	branch string,
	installationID int64,
) error {
	if !project.AutoDeploy {
		h.logger.Debug("auto_deploy disabled, skipping",
			"project_id", project.ID,
			"repo", event.Repository.FullName,
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

	existing, _ := h.deploymentSvc.FindByCommitSHAAndBranch(ctx, project.ID, event.After, branch)
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

	_, err := h.deploymentSvc.CreateFromWebhook(ctx, WebhookTriggerParams{
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

func (h *PushHandler) cleanupDeletedBranch(ctx context.Context, projectID, branch string) error {
	h.logger.Info("branch deleted, deactivating preview deployments",
		"project_id", projectID,
		"branch", branch,
	)

	var cleanupErrors []error
	deployments, err := h.deploymentSvc.DeactivateBranchDeployments(ctx, projectID, branch)
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("deactivate branch deployments: %w", err))
	} else {
		for _, deployment := range deployments {
			if err := h.routeManager.RemoveDeploymentRoute(ctx, deployment.ID); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("remove deployment route %s: %w", deployment.ID, err))
			}
		}
	}

	if err := h.routeManager.RemoveBranchRoute(ctx, projectID, branch); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("remove branch route: %w", err))
	}

	return errors.Join(cleanupErrors...)
}
