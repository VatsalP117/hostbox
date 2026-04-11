package api

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/go-playground/validator/v10"

	appmiddleware "github.com/vatsalpatel/hostbox/internal/api/middleware"
	"github.com/vatsalpatel/hostbox/internal/config"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/repository"
)

// Server holds the Echo instance and dependencies.
type Server struct {
	Echo      *echo.Echo
	Config    *config.Config
	DB        *sql.DB
	Repos     *repository.Repositories
	Validator *validator.Validate
	Logger    *slog.Logger
	startTime time.Time
}

// echoValidator wraps go-playground/validator to satisfy echo.Validator interface.
type echoValidator struct {
	validator *validator.Validate
}

func (ev *echoValidator) Validate(i interface{}) error {
	return ev.validator.Struct(i)
}

// NewServer creates and configures the Echo HTTP server.
func NewServer(cfg *config.Config, db *sql.DB, repos *repository.Repositories, logger *slog.Logger) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	v := validator.New()
	e.Validator = &echoValidator{validator: v}

	s := &Server{
		Echo:      e,
		Config:    cfg,
		DB:        db,
		Repos:     repos,
		Validator: v,
		Logger:    logger,
		startTime: time.Now(),
	}

	// Middleware stack
	e.Use(appmiddleware.RequestID())
	e.Use(appmiddleware.Logger(logger))
	e.Use(appmiddleware.Recovery(logger))
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: []string{cfg.BaseURL()},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: "default-src 'self'",
	}))

	// Custom error handler
	e.HTTPErrorHandler = s.customErrorHandler

	return s
}

// StartTime returns when the server was created.
func (s *Server) StartTime() time.Time {
	return s.startTime
}

// Start begins listening for HTTP requests. Blocks until shutdown signal.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	s.Logger.Info("server starting",
		"host", s.Config.Host,
		"port", s.Config.Port,
		"version", "dev",
	)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := s.Echo.Start(addr); err != nil && err != http.ErrServerClosed {
			s.Logger.Error("server error", "error", err)
		}
	}()

	<-quit
	s.Logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.Echo.Shutdown(ctx)
}

// Shutdown gracefully stops the server with a timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Echo.Shutdown(ctx)
}

// customErrorHandler converts Echo errors and AppErrors into consistent JSON responses.
func (s *Server) customErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var appErr *apperrors.AppError
	switch e := err.(type) {
	case *apperrors.AppError:
		appErr = e
	case *echo.HTTPError:
		msg := "An error occurred"
		if m, ok := e.Message.(string); ok {
			msg = m
		}
		appErr = &apperrors.AppError{
			Code:    http.StatusText(e.Code),
			Message: msg,
			Status:  e.Code,
		}
	default:
		s.Logger.Error("unhandled error", "error", err, "path", c.Request().URL.Path)
		appErr = apperrors.NewInternal(err)
	}

	c.JSON(appErr.Status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
			"details": appErr.Details,
		},
	})
}
