package deployment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
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
	activator      ProductionActivator
	reporter       worker.LifecycleReporter
	platformDomain string
	logger         *slog.Logger
}

func (s *Service) SetLifecycleReporter(reporter worker.LifecycleReporter) {
	s.reporter = reporter
}

func NewService(
	deployRepo *repository.DeploymentRepository,
	projectRepo *repository.ProjectRepository,
	pool *worker.Pool,
	executor *worker.BuildExecutor,
	activator ProductionActivator,
	platformDomain string,
	logger *slog.Logger,
) *Service {
	return &Service{
		deployRepo:     deployRepo,
		projectRepo:    projectRepo,
		pool:           pool,
		executor:       executor,
		activator:      activator,
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

	// Stop an active build explicitly. Queued supersession and replacement are
	// committed atomically below.
	existing, err := s.deployRepo.FindBuildingByProjectAndBranch(ctx, req.ProjectID, req.Branch)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("find active deployment: %w", err)
	}
	if existing != nil {
		if _, err := s.cancelDeployment(ctx, existing); err != nil {
			return nil, fmt.Errorf("cancel superseded deployment: %w", err)
		}
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

	cancelled, err := s.deployRepo.ReplaceQueued(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("create deployment: %w", err)
	}
	for i := range cancelled {
		s.reportLifecycle(ctx, project, &cancelled[i])
	}
	s.reportLifecycle(ctx, project, deployment)

	if s.pool != nil {
		s.pool.Offer(deployment.ID)
	}
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

	cancelled, err := s.cancelDeployment(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("cancel deployment: %w", err)
	}
	if !cancelled {
		current, getErr := s.deployRepo.GetByID(ctx, deployment.ID)
		if getErr != nil {
			return nil, fmt.Errorf("deployment state changed while cancelling")
		}
		return nil, fmt.Errorf("cannot cancel deployment in %q status", current.Status)
	}
	return deployment, nil
}

func (s *Service) cancelDeployment(ctx context.Context, deployment *models.Deployment) (bool, error) {
	expectedStatus := deployment.Status
	now := time.Now().UTC()
	deployment.Status = models.DeploymentStatusCancelled
	deployment.ErrorMessage = nil
	deployment.CompletedAt = &now
	updated, err := s.deployRepo.UpdateIfStatus(ctx, deployment, expectedStatus)
	if err != nil || !updated {
		return updated, err
	}

	if expectedStatus == models.DeploymentStatusBuilding && s.executor != nil {
		s.executor.CancelBuild(deployment.ID)
	}
	if project, projectErr := s.projectRepo.GetByID(ctx, deployment.ProjectID); projectErr != nil {
		s.logger.Warn("failed to load project for deployment feedback", "deployment_id", deployment.ID, "error", projectErr)
	} else {
		s.reportLifecycle(ctx, project, deployment)
	}
	return true, nil
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

	artifactPath, err := validateArtifact(target.ArtifactPath)
	if err != nil {
		return nil, fmt.Errorf("invalid rollback artifact: %w", err)
	}
	if s.activator == nil {
		return nil, fmt.Errorf("production activation is unavailable")
	}

	deploymentURL := fmt.Sprintf("https://%s", hostnames.ProductionHost(project.Slug, s.platformDomain))
	deployment := &models.Deployment{
		ID:                util.NewID(),
		ProjectID:         projectID,
		CommitSHA:         target.CommitSHA,
		CommitMessage:     target.CommitMessage,
		CommitAuthor:      target.CommitAuthor,
		Branch:            target.Branch,
		Status:            models.DeploymentStatusBuilding,
		IsProduction:      true,
		ArtifactPath:      &artifactPath,
		ArtifactSizeBytes: target.ArtifactSizeBytes,
		DeploymentURL:     &deploymentURL,
		IsRollback:        true,
		RollbackSourceID:  &target.ID,
		CreatedAt:         time.Now().UTC(),
	}

	if err := s.deployRepo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("create rollback deployment: %w", err)
	}

	if err := s.activateProduction(ctx, project, deployment); err != nil {
		return nil, err
	}
	if err := s.markReady(ctx, deployment); err != nil {
		return nil, fmt.Errorf("finalize rollback deployment: %w", err)
	}
	s.reportLifecycle(ctx, project, deployment)

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

	artifactPath, err := validateArtifact(source.ArtifactPath)
	if err != nil {
		return nil, fmt.Errorf("invalid promotion artifact: %w", err)
	}
	if s.activator == nil {
		return nil, fmt.Errorf("production activation is unavailable")
	}

	deploymentURL := fmt.Sprintf("https://%s", hostnames.ProductionHost(project.Slug, s.platformDomain))
	promoted := &models.Deployment{
		ID:                util.NewID(),
		ProjectID:         projectID,
		CommitSHA:         source.CommitSHA,
		CommitMessage:     source.CommitMessage,
		CommitAuthor:      source.CommitAuthor,
		Branch:            project.ProductionBranch,
		Status:            models.DeploymentStatusBuilding,
		IsProduction:      true,
		ArtifactPath:      &artifactPath,
		ArtifactSizeBytes: source.ArtifactSizeBytes,
		DeploymentURL:     &deploymentURL,
		CreatedAt:         time.Now().UTC(),
	}

	if err := s.deployRepo.Create(ctx, promoted); err != nil {
		return nil, fmt.Errorf("create promoted deployment: %w", err)
	}

	if err := s.activateProduction(ctx, project, promoted); err != nil {
		return nil, err
	}
	if err := s.markReady(ctx, promoted); err != nil {
		return nil, fmt.Errorf("finalize promoted deployment: %w", err)
	}
	s.reportLifecycle(ctx, project, promoted)

	s.logger.Info("deployment promoted", "id", promoted.ID, "source", source.ID)
	return promoted, nil
}

func (s *Service) reportLifecycle(ctx context.Context, project *models.Project, deployment *models.Deployment) {
	if s.reporter == nil {
		return
	}
	if err := s.reporter.Report(ctx, project, deployment); err != nil {
		s.logger.Warn("github lifecycle feedback failed", "deployment_id", deployment.ID, "status", deployment.Status, "error", err)
	}
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

func (s *Service) FindByCommitSHAAndBranch(ctx context.Context, projectID, commitSHA, branch string) (*models.Deployment, error) {
	return s.deployRepo.FindByCommitSHAAndBranch(ctx, projectID, commitSHA, branch)
}

// AssociatePullRequest attaches PR metadata when GitHub delivered the branch
// push before the pull_request event, then immediately publishes the current
// lifecycle state so the marker comment is not missed.
func (s *Service) AssociatePullRequest(ctx context.Context, deployment *models.Deployment, prNumber int) error {
	storedPR, err := s.deployRepo.SetGitHubPRNumberIfUnset(ctx, deployment.ID, prNumber)
	if err != nil {
		return err
	}
	if storedPR != prNumber {
		return fmt.Errorf("deployment is already associated with pull request %d", storedPR)
	}
	current, err := s.deployRepo.GetByID(ctx, deployment.ID)
	if err != nil {
		return fmt.Errorf("reload deployment for pull request association: %w", err)
	}
	project, err := s.projectRepo.GetByID(ctx, deployment.ProjectID)
	if err != nil {
		return fmt.Errorf("load project for pull request association: %w", err)
	}
	s.reportLifecycle(ctx, project, current)
	*deployment = *current
	return nil
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

// DeactivateBranchDeployments cancels active previews and returns all preview
// deployments whose routes may need idempotent cleanup.
func (s *Service) DeactivateBranchDeployments(ctx context.Context, projectID, branch string) ([]models.Deployment, error) {
	deployments, err := s.deployRepo.DeactivateBranchDeployments(ctx, projectID, branch)
	if err != nil {
		return nil, err
	}
	if s.executor != nil {
		for i := range deployments {
			if deployments[i].Status == models.DeploymentStatusBuilding {
				s.executor.CancelBuild(deployments[i].ID)
			}
		}
	}
	if s.reporter != nil && len(deployments) > 0 {
		project, projectErr := s.projectRepo.GetByID(ctx, projectID)
		if projectErr != nil {
			s.logger.Warn("failed to load project for branch cleanup feedback", "project_id", projectID, "error", projectErr)
		} else {
			for i := range deployments {
				if deployments[i].Status == models.DeploymentStatusCancelled {
					continue
				}
				deployments[i].Status = models.DeploymentStatusCancelled
				if deployments[i].CompletedAt == nil {
					now := time.Now().UTC()
					deployments[i].CompletedAt = &now
				}
				s.reportLifecycle(ctx, project, &deployments[i])
			}
		}
	}
	return deployments, nil
}

func (s *Service) activateProduction(ctx context.Context, project *models.Project, deployment *models.Deployment) error {
	framework := ""
	if project.Framework != nil {
		framework = *project.Framework
	}

	err := s.activator.ActivateProduction(ctx, ProductionActivation{
		ProjectID:    project.ID,
		ProjectSlug:  project.Slug,
		ArtifactPath: *deployment.ArtifactPath,
		Framework:    framework,
	})
	if err == nil {
		return nil
	}

	activationErr := fmt.Errorf("activate production deployment: %w", err)
	errorMessage := activationErr.Error()
	now := time.Now().UTC()
	deployment.Status = models.DeploymentStatusFailed
	deployment.ErrorMessage = &errorMessage
	deployment.CompletedAt = &now
	updated, updateErr := s.deployRepo.UpdateIfStatus(ctx, deployment, models.DeploymentStatusBuilding)
	if updateErr != nil || !updated {
		s.logger.Error("failed to mark deployment failed after activation error",
			"deployment_id", deployment.ID,
			"activation_error", err,
			"update_error", updateErr,
		)
		return fmt.Errorf("%w; additionally failed to persist failed status: %v", activationErr, updateErr)
	}

	return activationErr
}

func (s *Service) markReady(ctx context.Context, deployment *models.Deployment) error {
	now := time.Now().UTC()
	deployment.Status = models.DeploymentStatusReady
	deployment.ErrorMessage = nil
	deployment.CompletedAt = &now
	updated, err := s.deployRepo.UpdateIfStatus(ctx, deployment, models.DeploymentStatusBuilding)
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("deployment is no longer building")
	}
	return nil
}

func validateArtifact(artifactPath *string) (string, error) {
	if artifactPath == nil || strings.TrimSpace(*artifactPath) == "" {
		return "", fmt.Errorf("artifact path is empty")
	}

	path := strings.TrimSpace(*artifactPath)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat artifact path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("artifact path is not a directory")
	}

	dir, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open artifact directory: %w", err)
	}
	defer dir.Close()

	if _, err := dir.Readdirnames(1); err != nil {
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("artifact directory is empty")
		}
		return "", fmt.Errorf("read artifact directory: %w", err)
	}

	return path, nil
}
