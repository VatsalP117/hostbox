package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"

	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/internal/services"
)

type contextKey string

const UserContextKey contextKey = "user"

// GetUser extracts the authenticated user from the Echo context.
func GetUser(c echo.Context) *models.User {
	u, ok := c.Get(string(UserContextKey)).(*models.User)
	if !ok {
		return nil
	}
	return u
}

// JWTAuth validates the Bearer token and injects the user into context.
func JWTAuth(authService *services.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				return apperrors.NewUnauthorized("Missing or invalid authorization header")
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			claims, err := authService.ValidateAccessToken(tokenStr)
			if err != nil {
				return apperrors.NewUnauthorized("Invalid or expired token")
			}

			user, err := authService.GetCurrentUser(c.Request().Context(), claims.Subject)
			if err != nil {
				return apperrors.NewUnauthorized("User not found")
			}

			c.Set(string(UserContextKey), user)
			return next(c)
		}
	}
}

// RequireAdmin ensures the authenticated user is an admin.
func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := GetUser(c)
			if user == nil || !user.IsAdmin {
				return apperrors.NewForbidden("Admin access required")
			}
			return next(c)
		}
	}
}

// RequireSetupComplete checks that platform setup is done. Returns 503 if not.
func RequireSetupComplete(settingsRepo *repository.SettingsRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			complete, err := settingsRepo.GetBool(c.Request().Context(), "setup_complete")
			if err != nil || !complete {
				return &apperrors.AppError{
					Code:    "SETUP_REQUIRED",
					Message: "Platform setup required",
					Status:  503,
				}
			}
			return next(c)
		}
	}
}
