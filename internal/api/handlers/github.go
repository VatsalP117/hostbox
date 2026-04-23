package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

type GitHubHandler struct {
	runtime          *github.Runtime
	configStore      *github.AppConfigStore
	dashboardBaseURL string
	platformName     string
	logger           *slog.Logger
}

func NewGitHubHandler(
	runtime *github.Runtime,
	configStore *github.AppConfigStore,
	dashboardBaseURL string,
	platformName string,
	logger *slog.Logger,
) *GitHubHandler {
	return &GitHubHandler{
		runtime:          runtime,
		configStore:      configStore,
		dashboardBaseURL: strings.TrimRight(dashboardBaseURL, "/"),
		platformName:     platformName,
		logger:           logger,
	}
}

type githubStatusDTO struct {
	Configured bool   `json:"configured"`
	AppSlug    string `json:"app_slug,omitempty"`
	InstallURL string `json:"install_url,omitempty"`
}

// Status returns GitHub App connection metadata for the dashboard.
func (h *GitHubHandler) Status(c echo.Context) error {
	configured, appSlug := h.runtime.Status()
	status := githubStatusDTO{
		Configured: configured,
		AppSlug:    appSlug,
	}

	if status.Configured && appSlug != "" {
		status.InstallURL = githubInstallURL(appSlug)
	}

	return c.JSON(http.StatusOK, status)
}

type githubManifestDTO struct {
	ActionURL string         `json:"action_url"`
	Manifest  map[string]any `json:"manifest"`
}

// Manifest creates a GitHub App manifest payload for one-click app registration.
func (h *GitHubHandler) Manifest(c echo.Context) error {
	state, err := randomState()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to start GitHub connection",
			},
		})
	}
	if err := h.configStore.SetManifestState(c.Request().Context(), state); err != nil {
		h.logger.Error("failed to store github manifest state", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to start GitHub connection",
			},
		})
	}

	baseURL := h.dashboardBaseURL
	manifest := map[string]any{
		"name":        h.defaultAppName(),
		"url":         baseURL,
		"description": "Deploy static sites from GitHub repositories with Hostbox.",
		"hook_attributes": map[string]any{
			"url":    baseURL + "/api/v1/github/webhook",
			"active": true,
		},
		"redirect_url":  baseURL + "/github/manifest",
		"callback_urls": []string{baseURL + "/github/setup"},
		"setup_url":     baseURL + "/github/setup",
		"public":        false,
		"default_events": []string{
			"push",
			"pull_request",
		},
		"default_permissions": map[string]string{
			"contents":      "read",
			"deployments":   "write",
			"pull_requests": "write",
			"statuses":      "write",
		},
		"setup_on_update": true,
	}

	return c.JSON(http.StatusOK, githubManifestDTO{
		ActionURL: "https://github.com/settings/apps/new?state=" + url.QueryEscape(state),
		Manifest:  manifest,
	})
}

type githubManifestCompleteRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// CompleteManifest exchanges GitHub's manifest code for app credentials.
func (h *GitHubHandler) CompleteManifest(c echo.Context) error {
	var req githubManifestCompleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{"code": "VALIDATION_ERROR", "message": "Invalid request body"},
		})
	}

	expectedState, err := h.configStore.GetManifestState(c.Request().Context())
	if err != nil {
		h.logger.Error("failed to load github manifest state", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"code": "INTERNAL_ERROR", "message": "Failed to finish GitHub connection"},
		})
	}
	if req.Code == "" || req.State == "" || expectedState == "" || req.State != expectedState {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{"code": "VALIDATION_ERROR", "message": "Invalid GitHub connection state"},
		})
	}

	conversion, err := github.ConvertManifest(c.Request().Context(), req.Code)
	if err != nil {
		h.logger.Error("failed to convert github manifest", "error", err)
		return c.JSON(http.StatusBadGateway, map[string]any{
			"error": map[string]string{"code": "GITHUB_ERROR", "message": "Failed to finish GitHub connection"},
		})
	}

	appConfig := github.AppConfig{
		AppID:         conversion.ID,
		AppSlug:       conversion.Slug,
		PrivateKeyPEM: []byte(conversion.PEM),
		WebhookSecret: conversion.WebhookSecret,
	}
	if err := h.configStore.Save(c.Request().Context(), appConfig); err != nil {
		h.logger.Error("failed to save github app config", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"code": "INTERNAL_ERROR", "message": "Failed to save GitHub connection"},
		})
	}
	if err := h.runtime.Configure(appConfig); err != nil {
		h.logger.Error("failed to configure github runtime", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"code": "INTERNAL_ERROR", "message": "Failed to activate GitHub connection"},
		})
	}
	_ = h.configStore.SetManifestState(c.Request().Context(), "")

	return h.Status(c)
}

type installationDTO struct {
	ID         int64  `json:"id"`
	Account    string `json:"account"`
	AvatarURL  string `json:"avatar_url"`
	TargetType string `json:"target_type"`
}

// ListInstallations returns GitHub App installations.
func (h *GitHubHandler) ListInstallations(c echo.Context) error {
	ctx := c.Request().Context()

	installations, err := h.runtime.ListInstallations(ctx)
	if err != nil {
		h.logger.Error("failed to list github installations", "error", err)
		return c.JSON(http.StatusBadGateway, map[string]any{
			"error": map[string]string{
				"code":    "GITHUB_ERROR",
				"message": "Failed to fetch GitHub installations",
			},
		})
	}

	dtos := make([]installationDTO, len(installations))
	for i, inst := range installations {
		dtos[i] = installationDTO{
			ID:         inst.ID,
			Account:    inst.Account.Login,
			AvatarURL:  inst.Account.AvatarURL,
			TargetType: inst.TargetType,
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"installations": dtos,
	})
}

type repoDTO struct {
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	Language      string `json:"language"`
	Description   string `json:"description"`
}

// ListRepos returns repositories for a GitHub App installation.
func (h *GitHubHandler) ListRepos(c echo.Context) error {
	ctx := c.Request().Context()

	installationIDStr := c.QueryParam("installation_id")
	if installationIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "VALIDATION_ERROR",
				"message": "installation_id query parameter is required",
			},
		})
	}
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"code":    "VALIDATION_ERROR",
				"message": "installation_id must be a valid integer",
			},
		})
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	repos, total, err := h.runtime.ListRepos(ctx, installationID, page, perPage)
	if err != nil {
		h.logger.Error("failed to list repos", "installation_id", installationID, "error", err)
		return c.JSON(http.StatusBadGateway, map[string]any{
			"error": map[string]string{
				"code":    "GITHUB_ERROR",
				"message": "Failed to fetch repositories from GitHub",
			},
		})
	}

	dtos := make([]repoDTO, len(repos))
	for i, r := range repos {
		dtos[i] = repoDTO{
			FullName:      r.FullName,
			Name:          r.Name,
			Private:       r.Private,
			DefaultBranch: r.DefaultBranch,
			HTMLURL:       r.HTMLURL,
			Language:      r.Language,
			Description:   r.Description,
		}
	}

	totalPages := (total + perPage - 1) / perPage

	return c.JSON(http.StatusOK, map[string]any{
		"repos": dtos,
		"pagination": map[string]any{
			"total":       total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	})
}

func (h *GitHubHandler) defaultAppName() string {
	host := "self-hosted"
	if parsed, err := url.Parse(h.dashboardBaseURL); err == nil && parsed.Hostname() != "" {
		host = strings.ReplaceAll(parsed.Hostname(), ".", "-")
	}
	name := strings.TrimSpace(h.platformName)
	if name == "" {
		name = "Hostbox"
	}
	return name + " " + host
}

func githubInstallURL(appSlug string) string {
	return "https://github.com/apps/" + url.PathEscape(appSlug) + "/installations/new"
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
