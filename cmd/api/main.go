package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/vatsalpatel/hostbox/internal/api"
	"github.com/vatsalpatel/hostbox/internal/api/handlers"
	"github.com/vatsalpatel/hostbox/internal/api/routes"
	"github.com/vatsalpatel/hostbox/internal/config"
	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/logger"
	dockerpkg "github.com/vatsalpatel/hostbox/internal/platform/docker"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/internal/services"
	deploysvc "github.com/vatsalpatel/hostbox/internal/services/deployment"
	"github.com/vatsalpatel/hostbox/internal/worker"
	"github.com/vatsalpatel/hostbox/migrations"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Setup logger
	l := logger.Setup(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(l)

	// 3. Open database
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		l.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close(db)

	// 4. Run migrations
	if err := database.Migrate(db, migrations.FS); err != nil {
		l.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// 5. Initialize repositories
	repos := repository.New(db)

	// 6. Create auth service
	authService := services.NewAuthService(
		repos.User, repos.Session, repos.Settings, repos.Activity,
		cfg, l,
	)

	// 7. Start session cleanup
	go func() {
		authService.CleanupExpiredSessions(context.Background())
	}()

	// 8. Initialize Docker client
	dockerClient, err := dockerpkg.NewClient()
	if err != nil {
		l.Warn("docker client not available, build pipeline disabled", "error", err)
	}

	// 9. Initialize SSE hub
	sseHub := worker.NewSSEHub()

	// 10. Create server
	srv := api.NewServer(cfg, db, repos, l)

	// 11. Create handlers
	healthHandler := handlers.NewHealthHandler(srv.StartTime(), db)
	setupHandler := handlers.NewSetupHandler(authService, repos.User, repos.Settings, repos.Activity, cfg.PlatformHTTPS, l)
	authHandler := handlers.NewAuthHandler(authService, cfg.PlatformHTTPS, l)
	projectHandler := handlers.NewProjectHandler(repos.Project, repos.Activity, l)
	deploymentHandler := handlers.NewDeploymentHandler(repos.Deployment, repos.Project, repos.Activity, l)
	domainHandler := handlers.NewDomainHandler(repos.Domain, repos.Project, repos.Activity, cfg.PlatformDomain, l)
	envVarHandler := handlers.NewEnvVarHandler(repos.EnvVar, repos.Project, repos.Activity, cfg, l)
	adminHandler := handlers.NewAdminHandler(repos.User, repos.Project, repos.Deployment, repos.Activity, repos.Settings, cfg, l)

	// 12. Initialize build pipeline (executor, pool, service) if Docker is available
	if dockerClient != nil {
		executor := worker.NewBuildExecutor(
			&cfg.Build,
			cfg.EncryptionKey,
			dockerClient,
			repos.Deployment,
			repos.Project,
			repos.EnvVar,
			sseHub,
			cfg.PlatformDomain,
		)

		pool := worker.NewPool(
			cfg.MaxConcurrentBuilds,
			cfg.Build.JobChannelBuffer,
			executor,
			repos.Deployment,
			dockerClient,
		)

		deploymentService := deploysvc.NewService(
			repos.Deployment,
			repos.Project,
			pool,
			executor,
			cfg.PlatformDomain,
			l,
		)

		deploymentHandler.SetBuildDeps(deploymentService, sseHub, cfg.Build.LogBaseDir)

		if err := pool.Start(); err != nil {
			l.Error("failed to start worker pool", "error", err)
			os.Exit(1)
		}

		srv.OnShutdown(func() {
			l.Info("shutting down worker pool")
			pool.Shutdown(time.Duration(cfg.Build.ShutdownTimeoutSec) * time.Second)
		})
		srv.OnShutdown(func() {
			l.Info("closing docker client")
			dockerClient.Close()
		})

		l.Info("build pipeline initialized", "workers", cfg.MaxConcurrentBuilds)
	}

	// 13. Register routes
	routes.Register(srv.Echo, routes.Deps{
		AuthService:       authService,
		SettingsRepo:      repos.Settings,
		HealthHandler:     healthHandler,
		SetupHandler:      setupHandler,
		AuthHandler:       authHandler,
		ProjectHandler:    projectHandler,
		DeploymentHandler: deploymentHandler,
		DomainHandler:     domainHandler,
		EnvVarHandler:     envVarHandler,
		AdminHandler:      adminHandler,
	})

	// 14. Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		l.Error("server error", "error", err)
		os.Exit(1)
	}
}
