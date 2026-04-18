package api

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	appmiddleware "github.com/VatsalP117/hostbox/internal/api/middleware"
	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/dto"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/repository"
)

// Server holds the Echo instance and dependencies.
type Server struct {
	Echo          *echo.Echo
	Config        *config.Config
	DB            *sql.DB
	Repos         *repository.Repositories
	Logger        *slog.Logger
	startTime     time.Time
	shutdownHooks []func()
}

// echoValidator wraps go-playground/validator to satisfy echo.Validator interface.
type echoValidator struct{}

func (ev *echoValidator) Validate(i interface{}) error {
	return dto.ValidateStruct(i)
}

// NewServer creates and configures the Echo HTTP server.
func NewServer(cfg *config.Config, db *sql.DB, repos *repository.Repositories, logger *slog.Logger) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Validator = &echoValidator{}

	s := &Server{
		Echo:      e,
		Config:    cfg,
		DB:        db,
		Repos:     repos,
		Logger:    logger,
		startTime: time.Now(),
	}

	// Middleware stack (order matters)
	e.Use(appmiddleware.RequestID())
	e.Use(appmiddleware.Logger(logger))
	e.Use(appmiddleware.Recovery(logger))
	e.Use(appmiddleware.CORS([]string{cfg.DashboardBaseURL()}))
	e.Use(appmiddleware.SecurityHeaders())

	// Custom error handler
	e.HTTPErrorHandler = s.customErrorHandler

	return s
}

// StartTime returns when the server was created.
func (s *Server) StartTime() time.Time {
	return s.startTime
}

// Start begins listening for HTTP requests. Blocks until shutdown signal.
// OnShutdown registers a function to be called during graceful shutdown.
func (s *Server) OnShutdown(fn func()) {
	s.shutdownHooks = append(s.shutdownHooks, fn)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	s.Logger.Info("server starting",
		"host", s.Config.Host,
		"port", s.Config.Port,
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

	// Stop accepting new connections
	if err := s.Echo.Shutdown(ctx); err != nil {
		s.Logger.Error("echo shutdown error", "error", err)
	}

	// Run registered shutdown hooks (worker pool, docker, etc.)
	for _, hook := range s.shutdownHooks {
		hook()
	}

	return nil
}

// Shutdown gracefully stops the server with a timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Echo.Shutdown(ctx)
}

func (s *Server) ServeDashboard(distDir string) {
	stat, err := os.Stat(distDir)
	if err != nil || !stat.IsDir() {
		s.Logger.Warn("dashboard dist directory not found; API-only mode enabled", "path", distDir)
		return
	}

	s.Echo.GET("/*", func(c echo.Context) error {
		requestPath := c.Request().URL.Path
		if requestPath == "" || requestPath == "/" {
			return c.File(filepath.Join(distDir, "index.html"))
		}
		if len(requestPath) >= 5 && requestPath[:5] == "/api/" {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		relativePath := requestPath[1:]
		if info, err := os.Stat(filepath.Join(distDir, relativePath)); err == nil && !info.IsDir() {
			return c.File(filepath.Join(distDir, relativePath))
		}
		if _, err := fs.Stat(os.DirFS(distDir), relativePath); err == nil {
			return c.File(filepath.Join(distDir, relativePath))
		}
		return c.File(filepath.Join(distDir, "index.html"))
	})
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
