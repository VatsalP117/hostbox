package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client is a typed GitHub REST API client.
type Client struct {
	tokens     *TokenProvider
	httpClient *http.Client
	logger     *slog.Logger
	baseURL    string
}

func NewClient(tokens *TokenProvider, logger *slog.Logger) *Client {
	return NewClientWithBaseURL(tokens, logger, "https://api.github.com")
}

func NewClientWithBaseURL(tokens *TokenProvider, logger *slog.Logger, baseURL string) *Client {
	return &Client{
		tokens:     tokens,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		baseURL:    baseURL,
	}
}

func (c *Client) doInstallationRequest(ctx context.Context, installationID int64, method, url string, body interface{}, result interface{}) error {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.tokens.GetInstallationToken(installationID)
		if err != nil {
			return fmt.Errorf("get installation token: %w", err)
		}

		var bodyReader io.Reader
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(b)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("github %s %s: %w", method, url, err)
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			c.tokens.InvalidateToken(installationID)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("github %s %s returned %d: %s", method, url, resp.StatusCode, string(respBody))
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("github request failed after token refresh")
}

// --- Installations ---

// Installation represents a GitHub App installation.
type Installation struct {
	ID      int64 `json:"id"`
	Account struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"account"`
	AppID      int64             `json:"app_id"`
	TargetType string            `json:"target_type"`
	Permissions map[string]string `json:"permissions"`
	Events     []string          `json:"events"`
}

// ListInstallations returns all installations of the GitHub App.
func (c *Client) ListInstallations(ctx context.Context) ([]Installation, error) {
	appJWT, err := c.tokens.GenerateAppJWT()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/app/installations", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list installations returned %d: %s", resp.StatusCode, string(body))
	}

	var installations []Installation
	if err := json.NewDecoder(resp.Body).Decode(&installations); err != nil {
		return nil, fmt.Errorf("decode installations: %w", err)
	}
	return installations, nil
}

// --- Repositories ---

// Repository represents a GitHub repository.
type Repository struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	Language      string `json:"language"`
	Description   string `json:"description"`
}

type listReposResponse struct {
	TotalCount   int          `json:"total_count"`
	Repositories []Repository `json:"repositories"`
}

// ListRepos lists repositories accessible to an installation.
func (c *Client) ListRepos(ctx context.Context, installationID int64, page, perPage int) ([]Repository, int, error) {
	url := fmt.Sprintf("%s/installation/repositories?per_page=%d&page=%d", c.baseURL, perPage, page)

	var result listReposResponse
	if err := c.doInstallationRequest(ctx, installationID, "GET", url, nil, &result); err != nil {
		return nil, 0, err
	}
	return result.Repositories, result.TotalCount, nil
}

// GetRepo fetches a single repository's info.
func (c *Client) GetRepo(ctx context.Context, installationID int64, owner, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	var result Repository
	if err := c.doInstallationRequest(ctx, installationID, "GET", url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Deployment Statuses ---

// CreateDeploymentRequest creates a GitHub deployment.
type CreateDeploymentRequest struct {
	Ref              string   `json:"ref"`
	Task             string   `json:"task"`
	AutoMerge        bool     `json:"auto_merge"`
	RequiredContexts []string `json:"required_contexts"`
	Environment      string   `json:"environment"`
	Description      string   `json:"description"`
}

// DeploymentResponse is the response from creating a GitHub deployment.
type DeploymentResponse struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// CreateDeploymentStatusRequest updates a deployment status.
type CreateDeploymentStatusRequest struct {
	State          string `json:"state"`
	Description    string `json:"description"`
	EnvironmentURL string `json:"environment_url,omitempty"`
	LogURL         string `json:"log_url,omitempty"`
	AutoInactive   bool   `json:"auto_inactive"`
}

// CreateDeployment creates a GitHub Deployment for a ref.
func (c *Client) CreateDeployment(ctx context.Context, installationID int64, owner, repo string, req CreateDeploymentRequest) (*DeploymentResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/deployments", c.baseURL, owner, repo)

	var result DeploymentResponse
	if err := c.doInstallationRequest(ctx, installationID, "POST", url, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDeploymentStatus updates the status of a GitHub Deployment.
func (c *Client) CreateDeploymentStatus(ctx context.Context, installationID int64, owner, repo string, deploymentID int64, req CreateDeploymentStatusRequest) error {
	url := fmt.Sprintf("%s/repos/%s/%s/deployments/%d/statuses", c.baseURL, owner, repo, deploymentID)
	return c.doInstallationRequest(ctx, installationID, "POST", url, req, nil)
}

// --- PR Comments ---

// IssueComment represents a GitHub issue/PR comment.
type IssueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}

// ListPRComments lists comments on a pull request.
func (c *Client) ListPRComments(ctx context.Context, installationID int64, owner, repo string, prNumber int) ([]IssueComment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=100", c.baseURL, owner, repo, prNumber)

	var comments []IssueComment
	if err := c.doInstallationRequest(ctx, installationID, "GET", url, nil, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// CreatePRComment creates a new comment on a PR.
func (c *Client) CreatePRComment(ctx context.Context, installationID int64, owner, repo string, prNumber int, body string) (*IssueComment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, prNumber)

	reqBody := map[string]string{"body": body}
	var result IssueComment
	if err := c.doInstallationRequest(ctx, installationID, "POST", url, reqBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateComment updates an existing comment.
func (c *Client) UpdateComment(ctx context.Context, installationID int64, owner, repo string, commentID int64, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", c.baseURL, owner, repo, commentID)

	reqBody := map[string]string{"body": body}
	return c.doInstallationRequest(ctx, installationID, "PATCH", url, reqBody, nil)
}
