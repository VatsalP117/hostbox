package github

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Runtime holds the currently active GitHub App client and webhook router.
// It can be configured at startup from env/settings or later from the manifest flow.
type Runtime struct {
	mu sync.RWMutex

	appConfig     AppConfig
	tokenProvider *TokenProvider
	client        *Client
	eventRouter   *GitHubEventRouter
	logger        *slog.Logger
}

func NewRuntime(logger *slog.Logger) *Runtime {
	return &Runtime{logger: logger}
}

func (r *Runtime) Configure(cfg AppConfig) error {
	if cfg.AppID <= 0 || len(cfg.PrivateKeyPEM) == 0 {
		return fmt.Errorf("github app id and private key are required")
	}

	tokenProvider, err := NewTokenProvider(cfg, r.logger)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.appConfig = cfg
	r.tokenProvider = tokenProvider
	r.client = NewClient(tokenProvider, r.logger)
	r.mu.Unlock()

	return nil
}

func (r *Runtime) SetEventRouter(router *GitHubEventRouter) {
	r.mu.Lock()
	r.eventRouter = router
	r.mu.Unlock()
}

func (r *Runtime) Status() (configured bool, appSlug string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client != nil, r.appConfig.AppSlug
}

func (r *Runtime) WebhookSecretAndRouter() (string, *GitHubEventRouter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.client == nil || r.appConfig.WebhookSecret == "" || r.eventRouter == nil {
		return "", nil, false
	}
	return r.appConfig.WebhookSecret, r.eventRouter, true
}

func (r *Runtime) ListInstallations(ctx context.Context) ([]Installation, error) {
	client, err := r.clientOrError()
	if err != nil {
		return nil, err
	}
	return client.ListInstallations(ctx)
}

func (r *Runtime) ListRepos(ctx context.Context, installationID int64, page, perPage int) ([]Repository, int, error) {
	client, err := r.clientOrError()
	if err != nil {
		return nil, 0, err
	}
	return client.ListRepos(ctx, installationID, page, perPage)
}

func (r *Runtime) GetRepo(ctx context.Context, installationID int64, owner, repo string) (*Repository, error) {
	client, err := r.clientOrError()
	if err != nil {
		return nil, err
	}
	return client.GetRepo(ctx, installationID, owner, repo)
}

func (r *Runtime) GetInstallationToken(installationID int64) (string, error) {
	r.mu.RLock()
	tokenProvider := r.tokenProvider
	r.mu.RUnlock()

	if tokenProvider == nil {
		return "", fmt.Errorf("github app integration is not configured")
	}
	return tokenProvider.GetInstallationToken(installationID)
}

func (r *Runtime) clientOrError() (*Client, error) {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("github app integration is not configured")
	}
	return client, nil
}
