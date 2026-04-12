package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	caddysvc "github.com/vatsalpatel/hostbox/internal/services/caddy"
	deploysvc "github.com/vatsalpatel/hostbox/internal/services/deployment"
	ghsvc "github.com/vatsalpatel/hostbox/internal/services/github"
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

	// 9a. Initialize Caddy services
	caddyClient := caddysvc.NewCaddyClient(cfg.CaddyAdminURL, l)

	var dnsProviderConf json.RawMessage
	if cfg.DNSProvider != "" && cfg.DNSProviderConfig != "" {
		dnsProviderConf = json.RawMessage(cfg.DNSProviderConfig)
	}

	configBuilder := caddysvc.NewConfigBuilder(caddysvc.BuilderConfig{
		PlatformDomain:  cfg.PlatformDomain,
		PlatformHTTPS:   cfg.PlatformHTTPS,
		ACMEEmail:       cfg.ACMEEmail,
		APIUpstream:     fmt.Sprintf("localhost:%d", cfg.Port),
		DeploymentRoot:  cfg.DeploymentsDir,
		DNSProvider:     cfg.DNSProvider,
		DNSProviderConf: dnsProviderConf,
	})

	routeManager := caddysvc.NewRouteManager(caddyClient, configBuilder, l)

	deployAdapter := &caddysvc.DeploymentRepoAdapter{Repo: repos.Deployment}
	domainAdapter := &caddysvc.DomainRepoAdapter{Repo: repos.Domain}
	caddySyncSvc := caddysvc.NewSyncService(caddyClient, configBuilder, deployAdapter, domainAdapter, l)

	// Try to sync Caddy on startup (non-fatal if Caddy not available)
	ctx := context.Background()
	if err := caddySyncSvc.WaitForCaddy(ctx, 5*time.Second); err != nil {
		l.Warn("caddy not reachable on startup, routes will sync later", "error", err)
	} else {
		if err := caddySyncSvc.SyncAll(ctx); err != nil {
			l.Error("initial caddy sync failed", "error", err)
		}
	}

	// Start periodic re-sync every 5 minutes
	syncCtx, syncCancel := context.WithCancel(context.Background())
	caddySyncSvc.StartPeriodicSync(syncCtx, 5*time.Minute)

	// 9b. Initialize GitHub services (optional, only if configured)
	var ghClient *ghsvc.Client
	var ghEventRouter *ghsvc.GitHubEventRouter
	var ghWebhookHandler *handlers.GitHubWebhookHandler
	var ghHandler *handlers.GitHubHandler

	if cfg.GitHubAppID > 0 && len(cfg.GitHubAppPEM) > 0 {
		tokenProvider, err := ghsvc.NewTokenProvider(ghsvc.AppConfig{
			AppID:         cfg.GitHubAppID,
			AppSlug:       cfg.GitHubAppSlug,
			PrivateKeyPEM: []byte(cfg.GitHubAppPEM),
			WebhookSecret: cfg.GitHubWebhookSecret,
		}, l)
		if err != nil {
			l.Error("failed to initialize GitHub token provider", "error", err)
			os.Exit(1)
		}

		ghClient = ghsvc.NewClient(tokenProvider, l)
		ghHandler = handlers.NewGitHubHandler(ghClient, l)

		l.Info("github app integration initialized", "app_id", cfg.GitHubAppID)
	}

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

		// Wire Caddy route updates into the build pipeline
		executor.SetPostBuildHook(caddysvc.NewPostBuildRouteHook(routeManager, l))

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

		// Wire GitHub event handlers if GitHub integration is configured
		if ghClient != nil {
			pushHandler := ghsvc.NewPushHandler(repos.Project, deploymentService, l)
			prHandler := ghsvc.NewPullRequestHandler(repos.Project, deploymentService, routeManager, l)
			installHandler := ghsvc.NewInstallationHandler(repos.Project, l)
			ghEventRouter = ghsvc.NewGitHubEventRouter(pushHandler, prHandler, installHandler, l)
			ghWebhookHandler = handlers.NewGitHubWebhookHandler(cfg.GitHubWebhookSecret, ghEventRouter, l)
		}

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

	srv.OnShutdown(func() {
		l.Info("stopping caddy periodic sync")
		syncCancel()
	})

	// 13. Register routes
	routes.Register(srv.Echo, routes.Deps{
		AuthService:          authService,
		SettingsRepo:         repos.Settings,
		HealthHandler:        healthHandler,
		SetupHandler:         setupHandler,
		AuthHandler:          authHandler,
		ProjectHandler:       projectHandler,
		DeploymentHandler:    deploymentHandler,
		DomainHandler:        domainHandler,
		EnvVarHandler:        envVarHandler,
		AdminHandler:         adminHandler,
		GitHubWebhookHandler: ghWebhookHandler,
		GitHubHandler:        ghHandler,
	})

	// 14. Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		l.Error("server error", "error", err)
		os.Exit(1)
	}
}
