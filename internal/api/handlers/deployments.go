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
)

type DeploymentHandler struct {
	deploymentRepo *repository.DeploymentRepository
	projectRepo    *repository.ProjectRepository
	activityRepo   *repository.ActivityRepository
	logger         *slog.Logger
}

func NewDeploymentHandler(
	deploymentRepo *repository.DeploymentRepository,
	projectRepo *repository.ProjectRepository,
	activityRepo *repository.ActivityRepository,
	logger *slog.Logger,
) *DeploymentHandler {
	return &DeploymentHandler{
		deploymentRepo: deploymentRepo,
		projectRepo:    projectRepo,
		activityRepo:   activityRepo,
		logger:         logger,
	}
}

func (h *DeploymentHandler) ListByProject(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var pq dto.PaginationQuery
	if err := c.Bind(&pq); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	var status, branch *string
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}
	if b := c.QueryParam("branch"); b != "" {
		branch = &b
	}

	page := pq.PageOrDefault()
	perPage := pq.PerPageOrDefault()
	deployments, total, err := h.deploymentRepo.ListByProject(c.Request().Context(), project.ID, page, perPage, status, branch)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.DeploymentResponse, len(deployments))
	for i, d := range deployments {
		data[i] = toDeploymentResponse(&d)
	}

	return c.JSON(http.StatusOK, dto.DeploymentListResponse{
		Deployments: data,
		Pagination:  dto.NewPaginationResponse(total, page, perPage),
	})
}

func (h *DeploymentHandler) Create(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.CreateDeploymentRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	branch := project.ProductionBranch
	if req.Branch != nil && *req.Branch != "" {
		branch = *req.Branch
	}

	commitSHA := "manual"
	if req.CommitSHA != nil && *req.CommitSHA != "" {
		commitSHA = *req.CommitSHA
	}

	isProduction := branch == project.ProductionBranch

	deployment := &models.Deployment{
		ProjectID:    project.ID,
		CommitSHA:    commitSHA,
		Branch:       branch,
		Status:       models.DeploymentStatusQueued,
		IsProduction: isProduction,
	}

	if err := h.deploymentRepo.Create(c.Request().Context(), deployment); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "deployment.created",
		ResourceType: "deployment",
		ResourceID:   &deployment.ID,
	})

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"deployment": toDeploymentResponse(deployment),
	})
}

func (h *DeploymentHandler) Get(c echo.Context) error {
	deployment, err := h.deploymentRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Deployment")
		}
		return apperrors.NewInternal(err)
	}

	// Verify ownership via project
	_, err = h.getOwnedProject(c, deployment.ProjectID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"deployment": toDeploymentResponse(deployment),
	})
}

func (h *DeploymentHandler) getOwnedProject(c echo.Context, projectID string) (*models.Project, error) {
	user := middleware.GetUser(c)
	project, err := h.projectRepo.GetByID(c.Request().Context(), projectID)
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

func toDeploymentResponse(d *models.Deployment) dto.DeploymentResponse {
	resp := dto.DeploymentResponse{
		ID:           d.ID,
		ProjectID:    d.ProjectID,
		CommitSHA:    d.CommitSHA,
		CommitMessage: d.CommitMessage,
		CommitAuthor: d.CommitAuthor,
		Branch:       d.Branch,
		Status:       string(d.Status),
		IsProduction: d.IsProduction,
		DeploymentURL: d.DeploymentURL,
		ArtifactSizeBytes: d.ArtifactSizeBytes,
		ErrorMessage: d.ErrorMessage,
		BuildDurationMs: d.BuildDurationMs,
		CreatedAt:    d.CreatedAt.Format(time.RFC3339),
	}
	if d.StartedAt != nil {
		s := d.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if d.CompletedAt != nil {
		s := d.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &s
	}
	return resp
}
