package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/config"
	"github.com/vatsalpatel/hostbox/internal/dto"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
	adminsvc "github.com/vatsalpatel/hostbox/internal/services/admin"
	"github.com/vatsalpatel/hostbox/internal/services/backup"
)

type AdminHandler struct {
	userRepo       *repository.UserRepository
	projectRepo    *repository.ProjectRepository
	deploymentRepo *repository.DeploymentRepository
	activityRepo   *repository.ActivityRepository
	settingsRepo   *repository.SettingsRepository
	config         *config.Config
	startTime      time.Time
	logger         *slog.Logger
	backupService  *backup.Service
	updateService  *adminsvc.UpdateService
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	projectRepo *repository.ProjectRepository,
	deploymentRepo *repository.DeploymentRepository,
	activityRepo *repository.ActivityRepository,
	settingsRepo *repository.SettingsRepository,
	cfg *config.Config,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		projectRepo:    projectRepo,
		deploymentRepo: deploymentRepo,
		activityRepo:   activityRepo,
		settingsRepo:   settingsRepo,
		config:         cfg,
		startTime:      time.Now(),
		logger:         logger,
	}
}

func (h *AdminHandler) Stats(c echo.Context) error {
	ctx := c.Request().Context()

	projectCount, err := h.projectRepo.Count(ctx)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	deploymentCount, err := h.deploymentRepo.Count(ctx)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	activeBuilds, err := h.deploymentRepo.CountByStatuses(ctx, string(models.DeploymentStatusQueued), string(models.DeploymentStatusBuilding))
	if err != nil {
		return apperrors.NewInternal(err)
	}

	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	diskUsage := h.getDiskUsage()

	return c.JSON(http.StatusOK, dto.AdminStatsResponse{
		ProjectCount:    int64(projectCount),
		DeploymentCount: int64(deploymentCount),
		ActiveBuilds:    int64(activeBuilds),
		UserCount:       int64(userCount),
		DiskUsage:       diskUsage,
		UptimeSeconds:   int64(time.Since(h.startTime).Seconds()),
	})
}

func (h *AdminHandler) Activity(c echo.Context) error {
	var pq dto.PaginationQuery
	if err := c.Bind(&pq); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	var action, resourceType *string
	if a := c.QueryParam("action"); a != "" {
		action = &a
	}
	if rt := c.QueryParam("resource_type"); rt != "" {
		resourceType = &rt
	}

	page := pq.PageOrDefault()
	perPage := pq.PerPageOrDefault()
	entries, total, err := h.activityRepo.List(c.Request().Context(), page, perPage, action, resourceType)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.ActivityLogResponse, len(entries))
	for i, e := range entries {
		data[i] = toActivityResponse(&e)
	}

	return c.JSON(http.StatusOK, dto.ActivityListResponse{
		Activities: data,
		Pagination: dto.NewPaginationResponse(total, page, perPage),
	})
}

func (h *AdminHandler) Users(c echo.Context) error {
	var pq dto.PaginationQuery
	if err := c.Bind(&pq); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	page := pq.PageOrDefault()
	perPage := pq.PerPageOrDefault()
	users, total, err := h.userRepo.List(c.Request().Context(), page, perPage)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.UserResponse, len(users))
	for i, u := range users {
		data[i] = toUserResponse(&u)
	}

	return c.JSON(http.StatusOK, dto.UserListResponse{
		Users:      data,
		Pagination: dto.NewPaginationResponse(total, page, perPage),
	})
}

func (h *AdminHandler) GetSettings(c echo.Context) error {
	settings, err := h.loadSettings(c.Request().Context())
	if err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"settings": settings})
}

func (h *AdminHandler) UpdateSettings(c echo.Context) error {
	var req dto.UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	if req.RegistrationEnabled != nil {
		val := "false"
		if *req.RegistrationEnabled {
			val = "true"
		}
		if err := h.settingsRepo.Set(ctx, "registration_enabled", val); err != nil {
			return apperrors.NewInternal(err)
		}
	}
	if req.MaxProjects != nil {
		if err := h.settingsRepo.Set(ctx, "max_projects", fmt.Sprintf("%d", *req.MaxProjects)); err != nil {
			return apperrors.NewInternal(err)
		}
	}
	if req.MaxConcurrentBuilds != nil {
		if err := h.settingsRepo.Set(ctx, "max_concurrent_builds", fmt.Sprintf("%d", *req.MaxConcurrentBuilds)); err != nil {
			return apperrors.NewInternal(err)
		}
	}
	if req.ArtifactRetentionDays != nil {
		if err := h.settingsRepo.Set(ctx, "artifact_retention_days", fmt.Sprintf("%d", *req.ArtifactRetentionDays)); err != nil {
			return apperrors.NewInternal(err)
		}
	}

	// Return updated settings
	return h.GetSettings(c)
}

func (h *AdminHandler) getDiskUsage() dto.DiskUsageResponse {
	deploymentsSize := dirSize(h.config.DeploymentsDir)
	logsSize := dirSize(h.config.LogsDir)
	dbSize := fileSize(h.config.DatabasePath)
	totalSize := deploymentsSize + logsSize + dbSize

	return dto.DiskUsageResponse{
		DeploymentsBytes: deploymentsSize,
		DeploymentBytes:  deploymentsSize,
		LogsBytes:        logsSize,
		DatabaseBytes:    dbSize,
		TotalBytes:       totalSize,
		UsedBytes:        totalSize,
		AvailableBytes:   0,
	}
}

func (h *AdminHandler) ListDeployments(c echo.Context) error {
	var pq dto.PaginationQuery
	if err := c.Bind(&pq); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	page := pq.PageOrDefault()
	perPage := pq.PerPageOrDefault()

	deployments, total, err := h.deploymentRepo.ListRecent(c.Request().Context(), page, perPage)
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

func (h *AdminHandler) loadSettings(ctx context.Context) (map[string]interface{}, error) {
	settings, err := h.settingsRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	parseBool := func(key string, defaultValue bool) bool {
		value, ok := settings[key]
		if !ok {
			return defaultValue
		}
		return value == "true"
	}

	parseInt := func(key string, defaultValue int) int {
		value, ok := settings[key]
		if !ok {
			return defaultValue
		}
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
			return defaultValue
		}
		return parsed
	}

	return map[string]interface{}{
		"registration_enabled":    parseBool("registration_enabled", true),
		"max_projects":            parseInt("max_projects", 10),
		"max_concurrent_builds":   parseInt("max_concurrent_builds", 3),
		"artifact_retention_days": parseInt("artifact_retention_days", 30),
	}, nil
}

func toActivityResponse(a *models.ActivityLog) dto.ActivityLogResponse {
	resp := dto.ActivityLogResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		Metadata:     a.Metadata,
		CreatedAt:    a.CreatedAt.Format(time.RFC3339),
	}
	return resp
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// SetBackupService injects the backup service for admin endpoints.
func (h *AdminHandler) SetBackupService(svc *backup.Service) {
	h.backupService = svc
}

// SetUpdateService injects the update service for admin endpoints.
func (h *AdminHandler) SetUpdateService(svc *adminsvc.UpdateService) {
	h.updateService = svc
}

func (h *AdminHandler) CreateBackup(c echo.Context) error {
	if h.backupService == nil {
		return apperrors.NewInternal(fmt.Errorf("backup service not configured"))
	}

	compress := true
	if c.QueryParam("compress") == "false" {
		compress = false
	}

	result, err := h.backupService.CreateBackup(c.Request().Context(), compress)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"backup": result,
	})
}

func (h *AdminHandler) ListBackups(c echo.Context) error {
	if h.backupService == nil {
		return apperrors.NewInternal(fmt.Errorf("backup service not configured"))
	}

	backups, err := h.backupService.ListBackups()
	if err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"backups": backups,
	})
}

func (h *AdminHandler) RestoreBackup(c echo.Context) error {
	if h.backupService == nil {
		return apperrors.NewInternal(fmt.Errorf("backup service not configured"))
	}

	var req struct {
		Path string `json:"path" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	if err := h.backupService.Restore(c.Request().Context(), req.Path); err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Database restored successfully. Server will restart.",
	})
}

func (h *AdminHandler) CheckUpdate(c echo.Context) error {
	if h.updateService == nil {
		return apperrors.NewInternal(fmt.Errorf("update service not configured"))
	}

	check, err := h.updateService.Check(c.Request().Context())
	if err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"update": check,
	})
}
