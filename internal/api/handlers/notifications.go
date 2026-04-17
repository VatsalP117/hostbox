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
	notificationsvc "github.com/vatsalpatel/hostbox/internal/services/notification"
)

type NotificationHandler struct {
	notificationRepo *repository.NotificationRepository
	projectRepo      *repository.ProjectRepository
	activityRepo     *repository.ActivityRepository
	service          *notificationsvc.Service
	logger           *slog.Logger
}

func NewNotificationHandler(
	notificationRepo *repository.NotificationRepository,
	projectRepo *repository.ProjectRepository,
	activityRepo *repository.ActivityRepository,
	service *notificationsvc.Service,
	logger *slog.Logger,
) *NotificationHandler {
	return &NotificationHandler{
		notificationRepo: notificationRepo,
		projectRepo:      projectRepo,
		activityRepo:     activityRepo,
		service:          service,
		logger:           logger,
	}
}

func (h *NotificationHandler) ListByProject(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	configs, err := h.notificationRepo.ListByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.NotificationConfigResponse, len(configs))
	for i, cfg := range configs {
		data[i] = toNotificationResponse(&cfg)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"notifications": data})
}

func (h *NotificationHandler) Create(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.CreateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	events := "all"
	if req.Events != nil && *req.Events != "" {
		events = *req.Events
	}

	config := &models.NotificationConfig{
		ProjectID:  &project.ID,
		Channel:    req.Channel,
		WebhookURL: req.WebhookURL,
		Events:     events,
		Enabled:    true,
	}

	if err := h.notificationRepo.Create(c.Request().Context(), config); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "notification.created",
		ResourceType: "notification",
		ResourceID:   &config.ID,
	})

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"notification": toNotificationResponse(config),
	})
}

func (h *NotificationHandler) Update(c echo.Context) error {
	config, err := h.notificationRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Notification")
		}
		return apperrors.NewInternal(err)
	}

	if config.ProjectID == nil {
		return apperrors.NewForbidden("Global notification configs cannot be modified here")
	}

	if _, err := h.getOwnedProject(c, *config.ProjectID); err != nil {
		return err
	}

	var req dto.UpdateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	if req.WebhookURL != nil {
		config.WebhookURL = *req.WebhookURL
	}
	if req.Events != nil {
		config.Events = *req.Events
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.notificationRepo.Update(c.Request().Context(), config); err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"notification": toNotificationResponse(config),
	})
}

func (h *NotificationHandler) Delete(c echo.Context) error {
	config, err := h.notificationRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Notification")
		}
		return apperrors.NewInternal(err)
	}

	if config.ProjectID == nil {
		return apperrors.NewForbidden("Global notification configs cannot be deleted here")
	}

	if _, err := h.getOwnedProject(c, *config.ProjectID); err != nil {
		return err
	}

	if err := h.notificationRepo.Delete(c.Request().Context(), config.ID); err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *NotificationHandler) Test(c echo.Context) error {
	config, err := h.notificationRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Notification")
		}
		return apperrors.NewInternal(err)
	}

	if config.ProjectID == nil {
		return apperrors.NewForbidden("Global notification configs cannot be tested here")
	}

	project, err := h.getOwnedProject(c, *config.ProjectID)
	if err != nil {
		return err
	}

	payload := notificationsvc.NotificationPayload{
		Event: notificationsvc.EventDeploySuccess,
		Project: notificationsvc.ProjectInfo{
			ID:   project.ID,
			Name: project.Name,
			Slug: project.Slug,
		},
		Deployment: &notificationsvc.DeploymentInfo{
			ID:            "test-deployment",
			Status:        "ready",
			Branch:        project.ProductionBranch,
			CommitSHA:     "manual",
			DeploymentURL: "https://" + project.Slug + ".localhost",
			IsProduction:  true,
		},
		ServerURL: "http://localhost",
	}

	if err := h.service.SendTest(c.Request().Context(), config, payload); err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *NotificationHandler) getOwnedProject(c echo.Context, projectID string) (*models.Project, error) {
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

func toNotificationResponse(cfg *models.NotificationConfig) dto.NotificationConfigResponse {
	return dto.NotificationConfigResponse{
		ID:         cfg.ID,
		ProjectID:  cfg.ProjectID,
		Channel:    cfg.Channel,
		WebhookURL: cfg.WebhookURL,
		Events:     cfg.Events,
		Enabled:    cfg.Enabled,
		CreatedAt:  cfg.CreatedAt.Format(time.RFC3339),
	}
}
