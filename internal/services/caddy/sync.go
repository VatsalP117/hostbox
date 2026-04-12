package caddy

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// DeploymentQuerier queries active deployments for sync.
type DeploymentQuerier interface {
	ListActiveWithProject(ctx context.Context) ([]ActiveDeployment, error)
}

// DomainQuerier queries verified domains for sync.
type DomainQuerier interface {
	ListVerifiedWithProject(ctx context.Context) ([]VerifiedDomain, error)
}

// SyncService handles full configuration synchronization.
type SyncService struct {
	client     *CaddyClient
	builder    *ConfigBuilder
	deployRepo DeploymentQuerier
	domainRepo DomainQuerier
	logger     *slog.Logger
}

func NewSyncService(
	client *CaddyClient,
	builder *ConfigBuilder,
	deployRepo DeploymentQuerier,
	domainRepo DomainQuerier,
	logger *slog.Logger,
) *SyncService {
	return &SyncService{
		client:     client,
		builder:    builder,
		deployRepo: deployRepo,
		domainRepo: domainRepo,
		logger:     logger,
	}
}

// SyncAll builds the complete config from DB and loads it into Caddy.
func (s *SyncService) SyncAll(ctx context.Context) error {
	s.logger.Info("syncing caddy configuration from database")

	deployments, err := s.deployRepo.ListActiveWithProject(ctx)
	if err != nil {
		return fmt.Errorf("list active deployments: %w", err)
	}

	domains, err := s.domainRepo.ListVerifiedWithProject(ctx)
	if err != nil {
		return fmt.Errorf("list verified domains: %w", err)
	}

	config := s.builder.BuildFullConfig(deployments, domains)

	if err := s.client.LoadConfig(ctx, config); err != nil {
		return fmt.Errorf("load caddy config: %w", err)
	}

	s.logger.Info("caddy config synced",
		"deployment_routes", len(deployments),
		"domain_routes", len(domains),
	)
	return nil
}

// WaitForCaddy blocks until the Caddy admin API is reachable.
func (s *SyncService) WaitForCaddy(ctx context.Context, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("caddy admin API not reachable after %s", timeout)
		case <-ticker.C:
			if s.client.Healthy(ctx) {
				s.logger.Info("caddy admin API is reachable")
				return nil
			}
		}
	}
}

// StartPeriodicSync starts a background goroutine that re-syncs periodically.
func (s *SyncService) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.SyncAll(ctx); err != nil {
					s.logger.Error("periodic caddy sync failed", "error", err)
				}
			}
		}
	}()
}
