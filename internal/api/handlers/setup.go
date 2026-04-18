package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/VatsalP117/hostbox/internal/dto"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/services"
)

type SetupHandler struct {
	authService  *services.AuthService
	userRepo     *repository.UserRepository
	settingsRepo *repository.SettingsRepository
	activityRepo *repository.ActivityRepository
	logger       *slog.Logger
	useHTTPS     bool
}

func NewSetupHandler(
	authService *services.AuthService,
	userRepo *repository.UserRepository,
	settingsRepo *repository.SettingsRepository,
	activityRepo *repository.ActivityRepository,
	useHTTPS bool,
	logger *slog.Logger,
) *SetupHandler {
	return &SetupHandler{
		authService:  authService,
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
		activityRepo: activityRepo,
		useHTTPS:     useHTTPS,
		logger:       logger,
	}
}

func (h *SetupHandler) Status(c echo.Context) error {
	complete, err := h.settingsRepo.GetBool(c.Request().Context(), "setup_complete")
	if err != nil {
		return apperrors.NewInternal(err)
	}
	return c.JSON(http.StatusOK, dto.SetupStatusResponse{
		SetupRequired: !complete,
	})
}

func (h *SetupHandler) Setup(c echo.Context) error {
	ctx := c.Request().Context()

	// Check if already set up
	complete, err := h.settingsRepo.GetBool(ctx, "setup_complete")
	if err != nil {
		return apperrors.NewInternal(err)
	}
	if complete {
		return apperrors.NewForbidden("Setup already complete")
	}

	var req dto.SetupRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	// Check email uniqueness
	_, err = h.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return apperrors.NewConflict("Email already registered")
	}
	if err != sql.ErrNoRows {
		return apperrors.NewInternal(err)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	// Create admin user
	var displayName *string
	if req.DisplayName != "" {
		displayName = &req.DisplayName
	}
	user := &models.User{
		Email:         req.Email,
		PasswordHash:  string(hash),
		DisplayName:   displayName,
		IsAdmin:       true,
		EmailVerified: true,
	}
	if err := h.userRepo.Create(ctx, user); err != nil {
		return apperrors.NewInternal(err)
	}

	// Mark setup complete
	if err := h.settingsRepo.Set(ctx, "setup_complete", "true"); err != nil {
		return apperrors.NewInternal(err)
	}

	accessToken, rawRefresh, err := h.authService.CreateSession(
		ctx,
		user,
		c.Request().UserAgent(),
		c.RealIP(),
	)
	if err != nil {
		return apperrors.NewInternal(err)
	}
	c.SetCookie(&http.Cookie{
		Name:     "hostbox_refresh",
		Value:    rawRefresh,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.useHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60,
	})

	// Log activity
	action := "system.setup_complete"
	resourceType := "system"
	h.activityRepo.Create(ctx, &models.ActivityLog{
		UserID:       &user.ID,
		Action:       action,
		ResourceType: resourceType,
	})

	return c.JSON(http.StatusCreated, dto.AuthResponse{
		User:        toUserResponse(user),
		AccessToken: accessToken,
	})
}
