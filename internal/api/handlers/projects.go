package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/api/middleware"
	"github.com/vatsalpatel/hostbox/internal/dto"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/internal/util"
)

type ProjectHandler struct {
	projectRepo    *repository.ProjectRepository
	deploymentRepo *repository.DeploymentRepository
	domainRepo     *repository.DomainRepository
	activityRepo   *repository.ActivityRepository
	logger         *slog.Logger
}

func NewProjectHandler(
	projectRepo *repository.ProjectRepository,
	deploymentRepo *repository.DeploymentRepository,
	domainRepo *repository.DomainRepository,
	activityRepo *repository.ActivityRepository,
	logger *slog.Logger,
) *ProjectHandler {
	return &ProjectHandler{
		projectRepo:    projectRepo,
		deploymentRepo: deploymentRepo,
		domainRepo:     domainRepo,
		activityRepo:   activityRepo,
		logger:         logger,
	}
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
	slug := util.Slugify(req.Name)

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
		"project": toProjectResponse(project),
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
		data[i] = toProjectResponse(&p)
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

	domainData := make([]dto.DomainResponse, len(domains))
	for i, d := range domains {
		domainData[i] = toDomainResponse(&d)
	}

	var latestResp interface{}
	if latestDeployment != nil {
		latestResp = toDeploymentResponse(latestDeployment)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"project":           toProjectResponse(project),
		"latest_deployment": latestResp,
		"domains":           domainData,
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
		project.Slug = util.Slugify(*req.Name)
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

	return c.JSON(http.StatusOK, map[string]interface{}{
		"project": toProjectResponse(project),
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

func toProjectResponse(p *models.Project) dto.ProjectResponse {
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
		CreatedAt:            p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            p.UpdatedAt.Format(time.RFC3339),
	}
	return resp
}
