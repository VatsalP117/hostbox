package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

const (
	maxRetries     = 5
	baseBackoff    = 500 * time.Millisecond
	requestTimeout = 10 * time.Second
)

// CaddyClient communicates with the Caddy Admin API.
type CaddyClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewCaddyClient creates a client for the Caddy Admin API.
func NewCaddyClient(baseURL string, logger *slog.Logger) *CaddyClient {
	return &CaddyClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// LoadConfig replaces the entire Caddy configuration.
func (c *CaddyClient) LoadConfig(ctx context.Context, config *CaddyConfig) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal caddy config: %w", err)
	}
	return c.doWithRetry(ctx, "POST", "/load", body)
}

// GetConfig reads the current Caddy configuration.
func (c *CaddyClient) GetConfig(ctx context.Context) (*CaddyConfig, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/config/", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("caddy GET /config/: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caddy GET /config/ returned %d: %s", resp.StatusCode, string(respBody))
	}

	var config CaddyConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode caddy config: %w", err)
	}
	return &config, nil
}

// AddRoute appends a route to the specified server.
func (c *CaddyClient) AddRoute(ctx context.Context, serverName string, route CaddyRoute) error {
	body, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("marshal route: %w", err)
	}
	path := fmt.Sprintf("/config/apps/http/servers/%s/routes", serverName)
	return c.doWithRetry(ctx, "POST", path, body)
}

// PatchRoute replaces a route at a specific index.
func (c *CaddyClient) PatchRoute(ctx context.Context, serverName string, index int, route CaddyRoute) error {
	body, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("marshal route: %w", err)
	}
	path := fmt.Sprintf("/config/apps/http/servers/%s/routes/%d", serverName, index)
	return c.doWithRetry(ctx, "PUT", path, body)
}

// DeleteRoute removes a route by its @id.
func (c *CaddyClient) DeleteRoute(ctx context.Context, routeID string) error {
	path := fmt.Sprintf("/id/%s", routeID)
	return c.doWithRetry(ctx, "DELETE", path, nil)
}

// Healthy returns true if Caddy's admin API is reachable.
func (c *CaddyClient) Healthy(ctx context.Context) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", c.baseURL+"/config/", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *CaddyClient) doWithRetry(ctx context.Context, method, path string, body []byte) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(float64(baseBackoff) * math.Pow(2, float64(attempt-1)))
			c.logger.Debug("caddy retry",
				"attempt", attempt,
				"backoff", backoff,
				"method", method,
				"path", path,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(reqCtx, method, c.baseURL+path, bodyReader)
		if err != nil {
			cancel()
			return fmt.Errorf("create request: %w", err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("caddy %s %s: %w", method, path, err)
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("caddy %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))

		// Only retry on 5xx
		if resp.StatusCode < 500 {
			return lastErr
		}
	}

	return fmt.Errorf("caddy %s %s failed after %d retries: %w", method, path, maxRetries, lastErr)
}
