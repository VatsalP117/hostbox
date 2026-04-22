package handlers

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/VatsalP117/hostbox/internal/services/github"
	"github.com/labstack/echo/v4"
)

type GitHubHandler struct {
	githubClient *github.Client
	appSlug      string
	logger       *slog.Logger
}

func NewGitHubHandler(client *github.Client, appSlug string, logger *slog.Logger) *GitHubHandler {
	return &GitHubHandler{
		githubClient: client,
		appSlug:      appSlug,
		logger:       logger,
	}
}

type githubStatusDTO struct {
	Configured bool   `json:"configured"`
	AppSlug    string `json:"app_slug,omitempty"`
	InstallURL string `json:"install_url,omitempty"`
}

// Status returns GitHub App connection metadata for the dashboard.
func (h *GitHubHandler) Status(c echo.Context) error {
	status := githubStatusDTO{
		Configured: h.githubClient != nil,
		AppSlug:    h.appSlug,
	}

	if status.Configured && h.appSlug != "" {
		status.InstallURL = "https://github.com/apps/" + url.PathEscape(h.appSlug) + "/installations/new"
	}

	return c.JSON(http.StatusOK, status)
}

type installationDTO struct {
	ID         int64  `json:"id"`
	Account    string `json:"account"`
	AvatarURL  string `json:"avatar_url"`
	TargetType string `json:"target_type"`
}

// ListInstallations returns GitHub App installations.
func (h *GitHubHandler) ListInstallations(c echo.Context) error {
	if h.githubClient == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{
				"code":    "GITHUB_NOT_CONFIGURED",
				"message": "GitHub App integration is not configured",
			},
		})
	}

	ctx := c.Request().Context()

	installations, err := h.githubClient.ListInstallations(ctx)
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
	if h.githubClient == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{
				"code":    "GITHUB_NOT_CONFIGURED",
				"message": "GitHub App integration is not configured",
			},
		})
	}

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

	repos, total, err := h.githubClient.ListRepos(ctx, installationID, page, perPage)
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
