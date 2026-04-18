package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/VatsalP117/hostbox/internal/dto"
	"github.com/VatsalP117/hostbox/internal/version"
)

// HealthHandler handles the GET /api/v1/health endpoint.
type HealthHandler struct {
	startTime time.Time
	db        *sql.DB
}

func NewHealthHandler(startTime time.Time, db *sql.DB) *HealthHandler {
	return &HealthHandler{
		startTime: startTime,
		db:        db,
	}
}

// Health returns server health status.
func (h *HealthHandler) Health(c echo.Context) error {
	status := "ok"

	if err := h.db.PingContext(c.Request().Context()); err != nil {
		status = "degraded"
	}

	return c.JSON(http.StatusOK, dto.HealthResponse{
		Status:        status,
		Version:       version.Version,
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
	})
}
