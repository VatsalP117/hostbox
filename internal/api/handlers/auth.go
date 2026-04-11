package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/api/middleware"
	"github.com/vatsalpatel/hostbox/internal/dto"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
	useHTTPS    bool
	logger      *slog.Logger
}

func NewAuthHandler(authService *services.AuthService, useHTTPS bool, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{authService: authService, useHTTPS: useHTTPS, logger: logger}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	user, accessToken, rawRefresh, err := h.authService.Register(
		c.Request().Context(), &req, c.Request().UserAgent(), c.RealIP(),
	)
	if err != nil {
		return err
	}

	h.setRefreshCookie(c, rawRefresh)
	return c.JSON(http.StatusCreated, dto.AuthResponse{
		User:        toUserResponse(user),
		AccessToken: accessToken,
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	user, accessToken, rawRefresh, err := h.authService.Login(
		c.Request().Context(), &req, c.Request().UserAgent(), c.RealIP(),
	)
	if err != nil {
		return err
	}

	h.setRefreshCookie(c, rawRefresh)
	return c.JSON(http.StatusOK, dto.AuthResponse{
		User:        toUserResponse(user),
		AccessToken: accessToken,
	})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("hostbox_refresh")
	if err != nil || cookie.Value == "" {
		return apperrors.NewUnauthorized("Missing refresh token")
	}

	accessToken, err := h.authService.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dto.TokenResponse{AccessToken: accessToken})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie("hostbox_refresh")
	if err != nil || cookie.Value == "" {
		return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
	}

	user := middleware.GetUser(c)
	var userID *string
	if user != nil {
		userID = &user.ID
	}

	if err := h.authService.Logout(c.Request().Context(), cookie.Value, userID); err != nil {
		return err
	}

	h.clearRefreshCookie(c)
	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *AuthHandler) LogoutAll(c echo.Context) error {
	user := middleware.GetUser(c)
	count, err := h.authService.LogoutAll(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}

	h.clearRefreshCookie(c)
	return c.JSON(http.StatusOK, dto.LogoutAllResponse{
		Success:         true,
		SessionsRevoked: count,
	})
}

func (h *AuthHandler) Me(c echo.Context) error {
	user := middleware.GetUser(c)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": toUserResponse(user),
	})
}

func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	var req dto.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	user := middleware.GetUser(c)
	updated, err := h.authService.UpdateProfile(c.Request().Context(), user.ID, &req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": toUserResponse(updated),
	})
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	var req dto.ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	user := middleware.GetUser(c)
	if err := h.authService.ChangePassword(c.Request().Context(), user.ID, &req); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	_ = h.authService.ForgotPassword(c.Request().Context(), req.Email)
	return c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "If an account with that email exists, a password reset link has been sent.",
	})
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	if err := h.authService.ResetPassword(c.Request().Context(), &req); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req dto.VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	if err := h.authService.VerifyEmail(c.Request().Context(), req.Token); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

// --- Cookie helpers ---

func (h *AuthHandler) setRefreshCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "hostbox_refresh",
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.useHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})
}

func (h *AuthHandler) clearRefreshCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "hostbox_refresh",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.useHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   0,
	})
}

// --- Model → DTO conversions ---

func toUserResponse(u *models.User) dto.UserResponse {
	resp := dto.UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		IsAdmin:       u.IsAdmin,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
	}
	if !u.UpdatedAt.IsZero() {
		resp.UpdatedAt = u.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
