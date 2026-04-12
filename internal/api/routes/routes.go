package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/api/handlers"
	"github.com/vatsalpatel/hostbox/internal/api/middleware"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/internal/services"
)

// Deps holds all handler and service dependencies needed for route registration.
type Deps struct {
	AuthService  *services.AuthService
	SettingsRepo *repository.SettingsRepository

	HealthHandler        *handlers.HealthHandler
	SetupHandler         *handlers.SetupHandler
	AuthHandler          *handlers.AuthHandler
	ProjectHandler       *handlers.ProjectHandler
	DeploymentHandler    *handlers.DeploymentHandler
	DomainHandler        *handlers.DomainHandler
	EnvVarHandler        *handlers.EnvVarHandler
	AdminHandler         *handlers.AdminHandler
	GitHubWebhookHandler *handlers.GitHubWebhookHandler
	GitHubHandler        *handlers.GitHubHandler
}

// Register sets up all API routes on the Echo instance.
func Register(e *echo.Echo, deps Deps) {
	// Rate limiters
	authLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{Rate: 10, Burst: 10})
	apiLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{Rate: 100, Burst: 100})

	api := e.Group("/api/v1")

	// --- Public routes (rate limited by IP) ---
	pub := api.Group("")
	pub.Use(middleware.RateLimit(authLimiter, middleware.IPKeyFunc))

	pub.GET("/health", deps.HealthHandler.Health)
	pub.GET("/setup/status", deps.SetupHandler.Status)
	pub.POST("/setup", deps.SetupHandler.Setup)

	// Auth public routes (require setup complete)
	pub.POST("/auth/register", deps.AuthHandler.Register, middleware.RequireSetupComplete(deps.SettingsRepo))
	pub.POST("/auth/login", deps.AuthHandler.Login, middleware.RequireSetupComplete(deps.SettingsRepo))
	pub.POST("/auth/refresh", deps.AuthHandler.Refresh)
	pub.POST("/auth/forgot-password", deps.AuthHandler.ForgotPassword, middleware.RequireSetupComplete(deps.SettingsRepo))
	pub.POST("/auth/reset-password", deps.AuthHandler.ResetPassword, middleware.RequireSetupComplete(deps.SettingsRepo))
	pub.POST("/auth/verify-email", deps.AuthHandler.VerifyEmail, middleware.RequireSetupComplete(deps.SettingsRepo))

	// --- Authenticated routes (JWT + user rate limit) ---
	authed := api.Group("")
	authed.Use(middleware.JWTAuth(deps.AuthService))
	authed.Use(middleware.RateLimit(apiLimiter, middleware.UserKeyFunc))

	// Auth management
	authed.GET("/auth/me", deps.AuthHandler.Me)
	authed.PATCH("/auth/me", deps.AuthHandler.UpdateProfile)
	authed.PUT("/auth/me/password", deps.AuthHandler.ChangePassword)
	authed.POST("/auth/logout", deps.AuthHandler.Logout)
	authed.POST("/auth/logout-all", deps.AuthHandler.LogoutAll)

	// Projects
	authed.POST("/projects", deps.ProjectHandler.Create)
	authed.GET("/projects", deps.ProjectHandler.List)
	authed.GET("/projects/:id", deps.ProjectHandler.Get)
	authed.PATCH("/projects/:id", deps.ProjectHandler.Update)
	authed.DELETE("/projects/:id", deps.ProjectHandler.Delete)

	// Deployments
	authed.GET("/projects/:projectId/deployments", deps.DeploymentHandler.ListByProject)
	authed.POST("/projects/:projectId/deployments", deps.DeploymentHandler.Create)
	authed.POST("/projects/:projectId/deployments/trigger", deps.DeploymentHandler.TriggerDeploy)
	authed.POST("/projects/:projectId/redeploy", deps.DeploymentHandler.Redeploy)
	authed.POST("/projects/:projectId/rollback", deps.DeploymentHandler.Rollback)
	authed.POST("/projects/:projectId/promote/:deploymentId", deps.DeploymentHandler.Promote)
	authed.GET("/deployments/:id", deps.DeploymentHandler.Get)
	authed.GET("/deployments/:id/logs", deps.DeploymentHandler.GetLogs)
	authed.GET("/deployments/:id/logs/stream", deps.DeploymentHandler.StreamLogs)
	authed.POST("/deployments/:id/cancel", deps.DeploymentHandler.CancelDeploy)

	// Domains
	authed.POST("/projects/:projectId/domains", deps.DomainHandler.Create)
	authed.GET("/projects/:projectId/domains", deps.DomainHandler.ListByProject)
	authed.DELETE("/domains/:id", deps.DomainHandler.Delete)
	authed.POST("/domains/:id/verify", deps.DomainHandler.Verify)

	// Environment variables
	authed.POST("/projects/:projectId/env-vars", deps.EnvVarHandler.Create)
	authed.GET("/projects/:projectId/env-vars", deps.EnvVarHandler.ListByProject)
	authed.POST("/projects/:projectId/env-vars/bulk", deps.EnvVarHandler.BulkCreate)
	authed.PATCH("/env-vars/:id", deps.EnvVarHandler.Update)
	authed.DELETE("/env-vars/:id", deps.EnvVarHandler.Delete)

	// --- Admin routes (JWT + admin + user rate limit) ---
	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth(deps.AuthService))
	admin.Use(middleware.RequireAdmin())
	admin.Use(middleware.RateLimit(apiLimiter, middleware.UserKeyFunc))

	admin.GET("/stats", deps.AdminHandler.Stats)
	admin.GET("/activity", deps.AdminHandler.Activity)
	admin.GET("/users", deps.AdminHandler.Users)
	admin.GET("/settings", deps.AdminHandler.GetSettings)
	admin.PUT("/settings", deps.AdminHandler.UpdateSettings)

	// --- GitHub webhook (public, signature-verified) ---
	if deps.GitHubWebhookHandler != nil {
		api.POST("/github/webhook", deps.GitHubWebhookHandler.HandleWebhook)
	}

	// --- GitHub authenticated endpoints ---
	if deps.GitHubHandler != nil {
		gh := authed.Group("/github")
		gh.GET("/installations", deps.GitHubHandler.ListInstallations)
		gh.GET("/repos", deps.GitHubHandler.ListRepos)
	}
}
