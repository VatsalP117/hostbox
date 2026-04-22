package handlers

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/VatsalP117/hostbox/internal/api/middleware"
	"github.com/VatsalP117/hostbox/internal/dto"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/platform/hostnames"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/services/github"
)

type ProjectHandler struct {
	projectRepo      *repository.ProjectRepository
	deploymentRepo   *repository.DeploymentRepository
	domainRepo       *repository.DomainRepository
	envVarRepo       *repository.EnvVarRepository
	notificationRepo *repository.NotificationRepository
	activityRepo     *repository.ActivityRepository
	config           struct {
		PlatformDomain  string
		DashboardDomain string
	}
	githubRuntime *github.Runtime
	logger        *slog.Logger
}

func NewProjectHandler(
	projectRepo *repository.ProjectRepository,
	deploymentRepo *repository.DeploymentRepository,
	domainRepo *repository.DomainRepository,
	envVarRepo *repository.EnvVarRepository,
	notificationRepo *repository.NotificationRepository,
	activityRepo *repository.ActivityRepository,
	platformDomain string,
	dashboardDomain string,
	logger *slog.Logger,
) *ProjectHandler {
	return &ProjectHandler{
		projectRepo:      projectRepo,
		deploymentRepo:   deploymentRepo,
		domainRepo:       domainRepo,
		envVarRepo:       envVarRepo,
		notificationRepo: notificationRepo,
		activityRepo:     activityRepo,
		config: struct {
			PlatformDomain  string
			DashboardDomain string
		}{
			PlatformDomain:  platformDomain,
			DashboardDomain: dashboardDomain,
		},
		logger: logger,
	}
}

func (h *ProjectHandler) SetGitHubRuntime(runtime *github.Runtime) {
	h.githubRuntime = runtime
}

func (h *ProjectHandler) Create(c echo.Context) error {
	var req dto.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	user := middleware.GetUser(c)
	slug := hostnames.NormalizeProjectSlug(req.Name)
	if err := h.validateProjectSlug(slug); err != nil {
		return err
	}
	if err := h.validateGitHubRepository(c.Request().Context(), req.GitHubRepo, req.GitHubInstallationID); err != nil {
		return err
	}

	project := &models.Project{
		OwnerID:              user.ID,
		Name:                 req.Name,
		Slug:                 slug,
		GitHubRepo:           req.GitHubRepo,
		GitHubInstallationID: req.GitHubInstallationID,
		ProductionBranch:     "main",
		BuildCommand:         req.BuildCommand,
		InstallCommand:       req.InstallCommand,
		OutputDirectory:      req.OutputDirectory,
		RootDirectory:        "/",
		NodeVersion:          "20",
		AutoDeploy:           true,
		PreviewDeployments:   true,
	}

	if req.RootDirectory != nil {
		project.RootDirectory = *req.RootDirectory
	}
	if req.NodeVersion != nil {
		project.NodeVersion = *req.NodeVersion
	}

	if err := h.projectRepo.Create(c.Request().Context(), project); err != nil {
		return apperrors.NewInternal(err)
	}

	h.logActivity(c, &user.ID, "project.created", "project", &project.ID)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"project": toProjectResponse(project, projectStatus(nil)),
	})
}

func (h *ProjectHandler) List(c echo.Context) error {
	var pq dto.PaginationQuery
	if err := c.Bind(&pq); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	search := c.QueryParam("search")

	user := middleware.GetUser(c)
	page := pq.PageOrDefault()
	perPage := pq.PerPageOrDefault()

	projects, total, err := h.projectRepo.ListByOwner(c.Request().Context(), user.ID, page, perPage, search)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.ProjectResponse, len(projects))
	for i, p := range projects {
		latestDeployment, err := h.deploymentRepo.GetLatestByProjectAndBranch(c.Request().Context(), p.ID, p.ProductionBranch)
		if err != nil && err != sql.ErrNoRows {
			return apperrors.NewInternal(err)
		}

		data[i] = toProjectResponse(&p, projectStatus(latestDeployment))
	}

	return c.JSON(http.StatusOK, dto.ProjectListResponse{
		Projects:   data,
		Pagination: dto.NewPaginationResponse(total, page, perPage),
	})
}

func (h *ProjectHandler) Get(c echo.Context) error {
	project, err := h.getOwnedProject(c)
	if err != nil {
		return err
	}

	latestDeployment, err := h.deploymentRepo.GetLatestByProjectAndBranch(c.Request().Context(), project.ID, project.ProductionBranch)
	if err != nil && err != sql.ErrNoRows {
		return apperrors.NewInternal(err)
	}

	domains, err := h.domainRepo.ListByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	stats, err := h.deploymentRepo.SummarizeByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	envVarsCount, err := h.envVarRepo.CountByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	notificationsCount, err := h.notificationRepo.CountByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	domainData := make([]dto.DomainResponse, len(domains))
	for i, d := range domains {
		domainData[i] = toDomainResponse(&d)
	}

	var latestResp *dto.DeploymentResponse
	if latestDeployment != nil {
		resp := toDeploymentResponse(latestDeployment)
		latestResp = &resp
	}

	return c.JSON(http.StatusOK, dto.ProjectDetailResponse{
		Project:            toProjectResponse(project, projectStatus(latestDeployment)),
		LatestDeployment:   latestResp,
		Domains:            domainData,
		Stats:              toProjectStatsResponse(stats),
		EnvVarsCount:       envVarsCount,
		NotificationsCount: notificationsCount,
	})
}

func (h *ProjectHandler) Update(c echo.Context) error {
	project, err := h.getOwnedProject(c)
	if err != nil {
		return err
	}

	var req dto.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	if req.Name != nil {
		project.Name = *req.Name
		project.Slug = hostnames.NormalizeProjectSlug(*req.Name)
		if err := h.validateProjectSlug(project.Slug); err != nil {
			return err
		}
	}
	if req.BuildCommand != nil {
		project.BuildCommand = req.BuildCommand
	}
	if req.InstallCommand != nil {
		project.InstallCommand = req.InstallCommand
	}
	if req.OutputDirectory != nil {
		project.OutputDirectory = req.OutputDirectory
	}
	if req.RootDirectory != nil {
		project.RootDirectory = *req.RootDirectory
	}
	if req.NodeVersion != nil {
		project.NodeVersion = *req.NodeVersion
	}
	if req.ProductionBranch != nil {
		project.ProductionBranch = *req.ProductionBranch
	}
	if req.AutoDeploy != nil {
		project.AutoDeploy = *req.AutoDeploy
	}
	if req.PreviewDeployments != nil {
		project.PreviewDeployments = *req.PreviewDeployments
	}

	if err := h.projectRepo.Update(c.Request().Context(), project); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.logActivity(c, &user.ID, "project.updated", "project", &project.ID)

	latestDeployment, err := h.deploymentRepo.GetLatestByProjectAndBranch(c.Request().Context(), project.ID, project.ProductionBranch)
	if err != nil && err != sql.ErrNoRows {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"project": toProjectResponse(project, projectStatus(latestDeployment)),
	})
}

func (h *ProjectHandler) Delete(c echo.Context) error {
	project, err := h.getOwnedProject(c)
	if err != nil {
		return err
	}

	if err := h.projectRepo.Delete(c.Request().Context(), project.ID); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.logActivity(c, &user.ID, "project.deleted", "project", &project.ID)

	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

// --- Helpers ---

func (h *ProjectHandler) getOwnedProject(c echo.Context) (*models.Project, error) {
	user := middleware.GetUser(c)
	project, err := h.projectRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NewNotFound("Project")
		}
		return nil, apperrors.NewInternal(err)
	}
	if project.OwnerID != user.ID && !user.IsAdmin {
		return nil, apperrors.NewForbidden("Access denied")
	}
	return project, nil
}

func (h *ProjectHandler) logActivity(c echo.Context, userID *string, action, resourceType string, resourceID *string) {
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}

func (h *ProjectHandler) validateProjectSlug(slug string) error {
	if hostnames.CollidesWithDashboard(slug, h.config.PlatformDomain, h.config.DashboardDomain) {
		return apperrors.NewValidationError("Validation failed", []apperrors.FieldError{{
			Field:   "name",
			Message: "resolves to a reserved platform subdomain",
		}})
	}
	return nil
}

func (h *ProjectHandler) validateGitHubRepository(ctx context.Context, repo *string, installationID *int64) error {
	if installationID == nil {
		return nil
	}

	if repo == nil || strings.TrimSpace(*repo) == "" {
		return apperrors.NewBadRequest("github_repo is required when github_installation_id is provided")
	}
	if h.githubRuntime == nil {
		return apperrors.NewBadRequest("GitHub App integration is not configured")
	}

	owner, name, ok := splitGitHubRepo(*repo)
	if !ok {
		return apperrors.NewBadRequest("github_repo must be in owner/repository format")
	}

	if _, err := h.githubRuntime.GetRepo(ctx, *installationID, owner, name); err != nil {
		h.logger.Warn("selected github repository is not accessible through installation",
			"repo", strings.TrimSpace(*repo),
			"installation_id", *installationID,
			"error", err,
		)
		return apperrors.NewBadRequest("Selected repository is not accessible by this GitHub installation")
	}

	normalized := owner + "/" + name
	*repo = normalized
	return nil
}

func splitGitHubRepo(repo string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func toProjectResponse(p *models.Project, status string) dto.ProjectResponse {
	resp := dto.ProjectResponse{
		ID:                   p.ID,
		OwnerID:              p.OwnerID,
		Name:                 p.Name,
		Slug:                 p.Slug,
		GitHubRepo:           p.GitHubRepo,
		GitHubInstallationID: p.GitHubInstallationID,
		ProductionBranch:     p.ProductionBranch,
		Framework:            p.Framework,
		BuildCommand:         p.BuildCommand,
		InstallCommand:       p.InstallCommand,
		OutputDirectory:      p.OutputDirectory,
		RootDirectory:        p.RootDirectory,
		NodeVersion:          p.NodeVersion,
		AutoDeploy:           p.AutoDeploy,
		PreviewDeployments:   p.PreviewDeployments,
		Status:               status,
		CreatedAt:            p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            p.UpdatedAt.Format(time.RFC3339),
	}
	return resp
}

func toProjectStatsResponse(summary repository.ProjectDeploymentSummary) dto.ProjectStatsResponse {
	resp := dto.ProjectStatsResponse{
		TotalDeployments:   summary.TotalDeployments,
		SuccessfulBuilds:   summary.SuccessfulBuilds,
		FailedBuilds:       summary.FailedBuilds,
		AverageBuildTimeMs: summary.AverageBuildDurationMs,
	}
	if summary.LastDeployAt != nil {
		lastDeployAt := summary.LastDeployAt.Format(time.RFC3339)
		resp.LastDeployAt = &lastDeployAt
	}
	return resp
}

func projectStatus(latestDeployment *models.Deployment) string {
	if latestDeployment == nil {
		return "stopped"
	}

	switch latestDeployment.Status {
	case models.DeploymentStatusReady:
		return "healthy"
	case models.DeploymentStatusQueued, models.DeploymentStatusBuilding:
		return "building"
	case models.DeploymentStatusFailed, models.DeploymentStatusCancelled:
		return "failed"
	default:
		return "stopped"
	}
}
