package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/VatsalP117/hostbox/internal/api/middleware"
	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/dto"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/util"
)

const secretMask = "••••••••"

type EnvVarHandler struct {
	envVarRepo   *repository.EnvVarRepository
	projectRepo  *repository.ProjectRepository
	activityRepo *repository.ActivityRepository
	config       *config.Config
	logger       *slog.Logger
}

func NewEnvVarHandler(
	envVarRepo *repository.EnvVarRepository,
	projectRepo *repository.ProjectRepository,
	activityRepo *repository.ActivityRepository,
	cfg *config.Config,
	logger *slog.Logger,
) *EnvVarHandler {
	return &EnvVarHandler{
		envVarRepo:   envVarRepo,
		projectRepo:  projectRepo,
		activityRepo: activityRepo,
		config:       cfg,
		logger:       logger,
	}
}

func (h *EnvVarHandler) Create(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.CreateEnvVarRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	isSecret := false
	if req.IsSecret != nil {
		isSecret = *req.IsSecret
	}
	scope := "all"
	if req.Scope != nil {
		scope = *req.Scope
	}

	// Encrypt the value
	encryptedValue, err := util.Encrypt(req.Value, h.config.EncryptionKey)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	envVar := &models.EnvVar{
		ProjectID:      project.ID,
		Key:            req.Key,
		EncryptedValue: encryptedValue,
		IsSecret:       isSecret,
		Scope:          scope,
	}

	if err := h.envVarRepo.Create(c.Request().Context(), envVar); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "envvar.created",
		ResourceType: "envvar",
		ResourceID:   &envVar.ID,
	})

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"env_var": h.toEnvVarResponse(envVar),
	})
}

func (h *EnvVarHandler) ListByProject(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	envVars, err := h.envVarRepo.ListByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.EnvVarResponse, len(envVars))
	for i, ev := range envVars {
		data[i] = h.toEnvVarResponse(&ev)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"env_vars": data,
	})
}

func (h *EnvVarHandler) BulkCreate(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.BulkEnvVarRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	var created, updated int
	var responses []dto.EnvVarResponse
	ctx := c.Request().Context()

	for _, ev := range req.EnvVars {
		isSecret := false
		if ev.IsSecret != nil {
			isSecret = *ev.IsSecret
		}
		scope := "all"
		if ev.Scope != nil {
			scope = *ev.Scope
		}

		encryptedValue, err := util.Encrypt(ev.Value, h.config.EncryptionKey)
		if err != nil {
			return apperrors.NewInternal(err)
		}

		envVar := &models.EnvVar{
			ProjectID:      project.ID,
			Key:            ev.Key,
			EncryptedValue: encryptedValue,
			IsSecret:       isSecret,
			Scope:          scope,
		}

		// Check if exists
		existing, existErr := h.envVarRepo.GetByProjectKeyScope(ctx, project.ID, ev.Key, scope)
		if existErr == nil && existing != nil {
			existing.EncryptedValue = encryptedValue
			existing.IsSecret = isSecret
			if err := h.envVarRepo.Update(ctx, existing); err != nil {
				return apperrors.NewInternal(err)
			}
			envVar = existing
			updated++
		} else {
			if err := h.envVarRepo.Create(ctx, envVar); err != nil {
				return apperrors.NewInternal(err)
			}
			created++
		}

		responses = append(responses, h.toEnvVarResponse(envVar))
	}

	return c.JSON(http.StatusOK, dto.BulkCreateEnvVarResponse{
		EnvVars: responses,
		Created: created,
		Updated: updated,
	})
}

func (h *EnvVarHandler) Update(c echo.Context) error {
	envVar, err := h.envVarRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Environment variable")
		}
		return apperrors.NewInternal(err)
	}

	// Verify ownership via project
	_, err = h.getOwnedProject(c, envVar.ProjectID)
	if err != nil {
		return err
	}

	var req dto.UpdateEnvVarRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	if req.Value != nil {
		encryptedValue, err := util.Encrypt(*req.Value, h.config.EncryptionKey)
		if err != nil {
			return apperrors.NewInternal(err)
		}
		envVar.EncryptedValue = encryptedValue
	}
	if req.IsSecret != nil {
		envVar.IsSecret = *req.IsSecret
	}
	if req.Scope != nil {
		envVar.Scope = *req.Scope
	}

	if err := h.envVarRepo.Update(c.Request().Context(), envVar); err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"env_var": h.toEnvVarResponse(envVar),
	})
}

func (h *EnvVarHandler) Delete(c echo.Context) error {
	envVar, err := h.envVarRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Environment variable")
		}
		return apperrors.NewInternal(err)
	}

	_, err = h.getOwnedProject(c, envVar.ProjectID)
	if err != nil {
		return err
	}

	if err := h.envVarRepo.Delete(c.Request().Context(), envVar.ID); err != nil {
		return apperrors.NewInternal(err)
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *EnvVarHandler) getOwnedProject(c echo.Context, projectID string) (*models.Project, error) {
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

func (h *EnvVarHandler) toEnvVarResponse(ev *models.EnvVar) dto.EnvVarResponse {
	value := secretMask
	if !ev.IsSecret {
		decrypted, err := util.Decrypt(ev.EncryptedValue, h.config.EncryptionKey)
		if err != nil {
			h.logger.Error("failed to decrypt env var", "error", err, "id", ev.ID)
			value = "[decryption error]"
		} else {
			value = decrypted
		}
	}

	return dto.EnvVarResponse{
		ID:        ev.ID,
		ProjectID: ev.ProjectID,
		Key:       ev.Key,
		Value:     value,
		IsSecret:  ev.IsSecret,
		Scope:     ev.Scope,
		CreatedAt: ev.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ev.UpdatedAt.Format(time.RFC3339),
	}
}
