package handlers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/api/middleware"
	"github.com/vatsalpatel/hostbox/internal/dto"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
	deploysvc "github.com/vatsalpatel/hostbox/internal/services/deployment"
	"github.com/vatsalpatel/hostbox/internal/worker"
)

type DeploymentHandler struct {
	deploymentRepo *repository.DeploymentRepository
	projectRepo    *repository.ProjectRepository
	activityRepo   *repository.ActivityRepository
	service        *deploysvc.Service
	sseHub         *worker.SSEHub
	logDir         string
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

// SetBuildDeps injects the deployment service, SSE hub, and log directory.
// Called after the worker pool is initialized during startup.
func (h *DeploymentHandler) SetBuildDeps(service *deploysvc.Service, sseHub *worker.SSEHub, logDir string) {
	h.service = service
	h.sseHub = sseHub
	h.logDir = logDir
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

// TriggerDeploy creates a deployment via the build service and enqueues it.
func (h *DeploymentHandler) TriggerDeploy(c echo.Context) error {
	if h.service == nil {
		return apperrors.NewInternal(fmt.Errorf("build service not initialized"))
	}

	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.TriggerDeployRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if appErr := dto.ValidateStruct(req); appErr != nil {
		return appErr
	}

	deployment, err := h.service.TriggerDeployment(c.Request().Context(), deploysvc.TriggerRequest{
		ProjectID:     project.ID,
		Branch:        req.Branch,
		CommitSHA:     req.CommitSHA,
		CommitMessage: req.CommitMessage,
		CommitAuthor:  req.CommitAuthor,
	})
	if err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "deployment.triggered",
		ResourceType: "deployment",
		ResourceID:   &deployment.ID,
	})

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"deployment": toDeploymentResponse(deployment),
	})
}

// CancelDeploy cancels a queued or building deployment.
func (h *DeploymentHandler) CancelDeploy(c echo.Context) error {
	if h.service == nil {
		return apperrors.NewInternal(fmt.Errorf("build service not initialized"))
	}

	deployment, err := h.service.GetDeployment(c.Request().Context(), c.Param("id"))
	if err != nil {
		return apperrors.NewNotFound("Deployment")
	}

	// Verify ownership
	if _, err := h.getOwnedProject(c, deployment.ProjectID); err != nil {
		return err
	}

	cancelled, err := h.service.CancelDeployment(c.Request().Context(), deployment.ID)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"deployment": toDeploymentResponse(cancelled),
	})
}

// Rollback creates an instant deployment pointing to a previous deployment's artifacts.
func (h *DeploymentHandler) Rollback(c echo.Context) error {
	if h.service == nil {
		return apperrors.NewInternal(fmt.Errorf("build service not initialized"))
	}

	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.RollbackRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if appErr := dto.ValidateStruct(req); appErr != nil {
		return appErr
	}

	deployment, err := h.service.Rollback(c.Request().Context(), project.ID, req.DeploymentID)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "deployment.rollback",
		ResourceType: "deployment",
		ResourceID:   &deployment.ID,
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"deployment": toDeploymentResponse(deployment),
	})
}

// Promote elevates a preview deployment to production.
func (h *DeploymentHandler) Promote(c echo.Context) error {
	if h.service == nil {
		return apperrors.NewInternal(fmt.Errorf("build service not initialized"))
	}

	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	deploymentID := c.Param("deploymentId")
	deployment, err := h.service.Promote(c.Request().Context(), project.ID, deploymentID)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "deployment.promoted",
		ResourceType: "deployment",
		ResourceID:   &deployment.ID,
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"deployment": toDeploymentResponse(deployment),
	})
}

// Redeploy triggers a new build using the latest production deployment's config.
func (h *DeploymentHandler) Redeploy(c echo.Context) error {
	if h.service == nil {
		return apperrors.NewInternal(fmt.Errorf("build service not initialized"))
	}

	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	deployment, err := h.service.Redeploy(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"deployment": toDeploymentResponse(deployment),
	})
}

// GetLogs returns the build log lines for a deployment.
func (h *DeploymentHandler) GetLogs(c echo.Context) error {
	deployment, err := h.deploymentRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Deployment")
		}
		return apperrors.NewInternal(err)
	}

	if _, err := h.getOwnedProject(c, deployment.ProjectID); err != nil {
		return err
	}

	logPath := ""
	if deployment.LogPath != nil {
		logPath = *deployment.LogPath
	}
	if logPath == "" && h.logDir != "" {
		logPath = filepath.Join(h.logDir, deployment.ID+".log")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusOK, dto.LogResponse{Lines: []string{}, TotalLines: 0})
		}
		return apperrors.NewInternal(err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	total := len(lines)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset >= total {
		offset = total
	}

	return c.JSON(http.StatusOK, dto.LogResponse{
		Lines:      lines[offset:end],
		TotalLines: total,
		HasMore:    end < total,
	})
}

// StreamLogs streams build logs via Server-Sent Events.
func (h *DeploymentHandler) StreamLogs(c echo.Context) error {
	if h.sseHub == nil {
		return apperrors.NewInternal(fmt.Errorf("SSE not initialized"))
	}

	deploymentID := c.Param("id")
	deployment, err := h.deploymentRepo.GetByID(c.Request().Context(), deploymentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Deployment")
		}
		return apperrors.NewInternal(err)
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")
	c.Response().WriteHeader(http.StatusOK)

	lastEventIDStr := c.Request().Header.Get("Last-Event-ID")
	lastEventID, _ := strconv.ParseInt(lastEventIDStr, 10, 64)

	logPath := ""
	if deployment.LogPath != nil {
		logPath = *deployment.LogPath
	}
	if logPath == "" && h.logDir != "" {
		logPath = filepath.Join(h.logDir, deploymentID+".log")
	}

	if data, readErr := os.ReadFile(logPath); readErr == nil {
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		for i, line := range lines {
			lineID := int64(i + 1)
			if lineID <= lastEventID {
				continue
			}
			fmt.Fprintf(c.Response(), "id: %d\nevent: log\ndata: %s\n\n", lineID, line)
		}
		c.Response().Flush()
	}

	// If the build is already completed, send done event and close
	if deployment.Status == models.DeploymentStatusReady ||
		deployment.Status == models.DeploymentStatusFailed ||
		deployment.Status == models.DeploymentStatusCancelled {
		fmt.Fprintf(c.Response(), "event: done\ndata: {\"status\":\"%s\"}\n\n", deployment.Status)
		c.Response().Flush()
		return nil
	}

	events, unsubscribe := h.sseHub.Subscribe(deploymentID)
	defer unsubscribe()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			fmt.Fprintf(c.Response(), "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, event.Data)
			c.Response().Flush()

			if event.Type == worker.SSEEventDone {
				return nil
			}
		}
	}
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
