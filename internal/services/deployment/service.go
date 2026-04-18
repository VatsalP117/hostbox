package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/platform/hostnames"
	"github.com/VatsalP117/hostbox/internal/repository"
	ghsvc "github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/VatsalP117/hostbox/internal/util"
	"github.com/VatsalP117/hostbox/internal/worker"
)

type Service struct {
	deployRepo     *repository.DeploymentRepository
	projectRepo    *repository.ProjectRepository
	pool           *worker.Pool
	executor       *worker.BuildExecutor
	platformDomain string
	logger         *slog.Logger
}

func NewService(
	deployRepo *repository.DeploymentRepository,
	projectRepo *repository.ProjectRepository,
	pool *worker.Pool,
	executor *worker.BuildExecutor,
	platformDomain string,
	logger *slog.Logger,
) *Service {
	return &Service{
		deployRepo:     deployRepo,
		projectRepo:    projectRepo,
		pool:           pool,
		executor:       executor,
		platformDomain: platformDomain,
		logger:         logger,
	}
}

// TriggerDeployment creates a new deployment and enqueues it for building.
func (s *Service) TriggerDeployment(ctx context.Context, req TriggerRequest) (*models.Deployment, error) {
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	isProduction := req.Branch == project.ProductionBranch

	// Deduplication: cancel any existing queued/building deploy for this project+branch
	existing, _ := s.deployRepo.FindQueuedOrBuilding(ctx, req.ProjectID, req.Branch)
	if existing != nil {
		s.cancelDeployment(ctx, existing)
	}

	deployment := &models.Deployment{
		ID:             util.NewID(),
		ProjectID:      req.ProjectID,
		CommitSHA:      req.CommitSHA,
		CommitMessage:  req.CommitMessage,
		CommitAuthor:   req.CommitAuthor,
		Branch:         req.Branch,
		Status:         models.DeploymentStatusQueued,
		IsProduction:   isProduction,
		GitHubPRNumber: req.PRNumber,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.deployRepo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("create deployment: %w", err)
	}

	s.pool.Enqueue(deployment.ID)
	s.logger.Info("deployment triggered", "id", deployment.ID, "project", req.ProjectID, "branch", req.Branch)

	return deployment, nil
}

// CancelDeployment cancels a queued or building deployment.
func (s *Service) CancelDeployment(ctx context.Context, deploymentID string) (*models.Deployment, error) {
	deployment, err := s.deployRepo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("deployment not found: %w", err)
	}

	if deployment.Status != models.DeploymentStatusQueued && deployment.Status != models.DeploymentStatusBuilding {
		return nil, fmt.Errorf("cannot cancel deployment in %q status", deployment.Status)
	}

	s.cancelDeployment(ctx, deployment)
	return deployment, nil
}

func (s *Service) cancelDeployment(ctx context.Context, deployment *models.Deployment) {
	if deployment.Status == models.DeploymentStatusBuilding {
		s.executor.CancelBuild(deployment.ID)
	}

	now := time.Now().UTC()
	deployment.Status = models.DeploymentStatusCancelled
	deployment.CompletedAt = &now
	if err := s.deployRepo.Update(ctx, deployment); err != nil {
		s.logger.Error("failed to cancel deployment", "id", deployment.ID, "err", err)
	}
}

// GetDeployment returns a single deployment by ID.
func (s *Service) GetDeployment(ctx context.Context, id string) (*models.Deployment, error) {
	return s.deployRepo.GetByID(ctx, id)
}

// ListDeployments returns paginated deployments for a project.
func (s *Service) ListDeployments(ctx context.Context, projectID string, opts ListOpts) ([]models.Deployment, int, error) {
	return s.deployRepo.ListByProject(ctx, projectID, opts.Page, opts.PerPage, opts.Status, opts.Branch)
}

// Rollback creates a new deployment that points to a previous deployment's artifacts.
func (s *Service) Rollback(ctx context.Context, projectID, targetDeploymentID string) (*models.Deployment, error) {
	target, err := s.deployRepo.GetByID(ctx, targetDeploymentID)
	if err != nil {
		return nil, fmt.Errorf("target deployment not found: %w", err)
	}
	if target.Status != models.DeploymentStatusReady {
		return nil, fmt.Errorf("cannot rollback to deployment in %q status", target.Status)
	}
	if target.ProjectID != projectID {
		return nil, fmt.Errorf("deployment does not belong to this project")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	now := time.Now().UTC()
	deploymentURL := fmt.Sprintf("https://%s", hostnames.ProductionHost(project.Slug, s.platformDomain))
	deployment := &models.Deployment{
		ID:                util.NewID(),
		ProjectID:         projectID,
		CommitSHA:         target.CommitSHA,
		CommitMessage:     target.CommitMessage,
		CommitAuthor:      target.CommitAuthor,
		Branch:            target.Branch,
		Status:            models.DeploymentStatusReady,
		IsProduction:      true,
		ArtifactPath:      target.ArtifactPath,
		ArtifactSizeBytes: target.ArtifactSizeBytes,
		DeploymentURL:     &deploymentURL,
		IsRollback:        true,
		RollbackSourceID:  &target.ID,
		CompletedAt:       &now,
		CreatedAt:         now,
	}

	if err := s.deployRepo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("create rollback deployment: %w", err)
	}

	// TODO (Phase 5): Update Caddy routes to point to this deployment
	s.logger.Info("rollback created", "id", deployment.ID, "source", target.ID)
	return deployment, nil
}

// Promote makes a preview deployment the new production deployment.
func (s *Service) Promote(ctx context.Context, projectID, deploymentID string) (*models.Deployment, error) {
	source, err := s.deployRepo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("deployment not found: %w", err)
	}
	if source.Status != models.DeploymentStatusReady {
		return nil, fmt.Errorf("cannot promote deployment in %q status", source.Status)
	}
	if source.ProjectID != projectID {
		return nil, fmt.Errorf("deployment does not belong to this project")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	now := time.Now().UTC()
	deploymentURL := fmt.Sprintf("https://%s", hostnames.ProductionHost(project.Slug, s.platformDomain))
	promoted := &models.Deployment{
		ID:                util.NewID(),
		ProjectID:         projectID,
		CommitSHA:         source.CommitSHA,
		CommitMessage:     source.CommitMessage,
		CommitAuthor:      source.CommitAuthor,
		Branch:            project.ProductionBranch,
		Status:            models.DeploymentStatusReady,
		IsProduction:      true,
		ArtifactPath:      source.ArtifactPath,
		ArtifactSizeBytes: source.ArtifactSizeBytes,
		DeploymentURL:     &deploymentURL,
		CompletedAt:       &now,
		CreatedAt:         now,
	}

	if err := s.deployRepo.Create(ctx, promoted); err != nil {
		return nil, fmt.Errorf("create promoted deployment: %w", err)
	}

	// TODO (Phase 5): Update Caddy routes
	s.logger.Info("deployment promoted", "id", promoted.ID, "source", source.ID)
	return promoted, nil
}

// Redeploy triggers a new build using the same branch and latest commit.
func (s *Service) Redeploy(ctx context.Context, projectID string) (*models.Deployment, error) {
	latest, err := s.deployRepo.FindLatestReady(ctx, projectID, true)
	if err != nil {
		return nil, fmt.Errorf("no previous production deployment found: %w", err)
	}

	return s.TriggerDeployment(ctx, TriggerRequest{
		ProjectID:     projectID,
		Branch:        latest.Branch,
		CommitSHA:     latest.CommitSHA,
		CommitMessage: latest.CommitMessage,
		CommitAuthor:  latest.CommitAuthor,
	})
}

// FindByCommitSHA finds a deployment by project and commit SHA.
func (s *Service) FindByCommitSHA(ctx context.Context, projectID, commitSHA string) (*models.Deployment, error) {
	return s.deployRepo.FindByCommitSHA(ctx, projectID, commitSHA)
}

// CreateFromWebhook creates a deployment triggered by a GitHub webhook.
func (s *Service) CreateFromWebhook(ctx context.Context, params ghsvc.WebhookTriggerParams) (*models.Deployment, error) {
	var prNumber *int
	if params.GitHubPRNumber > 0 {
		prNumber = &params.GitHubPRNumber
	}
	var commitMsg *string
	if params.CommitMessage != "" {
		commitMsg = &params.CommitMessage
	}
	var commitAuthor *string
	if params.CommitAuthor != "" {
		commitAuthor = &params.CommitAuthor
	}

	return s.TriggerDeployment(ctx, TriggerRequest{
		ProjectID:     params.ProjectID,
		Branch:        params.Branch,
		CommitSHA:     params.CommitSHA,
		CommitMessage: commitMsg,
		CommitAuthor:  commitAuthor,
		PRNumber:      prNumber,
	})
}

// DeactivateBranchDeployments marks all ready deployments for a branch as cancelled.
func (s *Service) DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error) {
	return s.deployRepo.DeactivateBranchDeployments(ctx, projectID, branch)
}
