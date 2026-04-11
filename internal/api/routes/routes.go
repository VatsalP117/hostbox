package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/vatsalpatel/hostbox/internal/api/handlers"
)

// Register sets up all API routes on the Echo instance.
func Register(e *echo.Echo, health *handlers.HealthHandler) {
	api := e.Group("/api/v1")

	// Public routes (no auth)
	api.GET("/health", health.Health)
}
