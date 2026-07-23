package github

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

const MaxWebhookPayloadBytes int64 = 1 << 20

type DeliveryRouter interface {
	Route(ctx context.Context, eventType string, payload []byte, deliveryID string) error
}

// RouterProvider resolves the currently configured event router. Runtime
// integration can use RouterProviderFunc without coupling the processor to the
// mutable GitHub Runtime implementation.
type RouterProvider interface {
	WebhookRouter() (DeliveryRouter, bool)
}

type RouterProviderFunc func() (DeliveryRouter, bool)

func (f RouterProviderFunc) WebhookRouter() (DeliveryRouter, bool) {
	return f()
}

type WebhookDeliveryRepository interface {
	Create(context.Context, *models.GitHubWebhookDelivery) (bool, error)
	ClaimNext(context.Context, time.Time, int) (*models.GitHubWebhookDelivery, error)
	RecoverProcessing(context.Context, time.Time, int) (int64, error)
	MarkCompleted(context.Context, string, time.Time) (bool, error)
	MarkForRetry(context.Context, string, string, time.Time, time.Time) (bool, error)
	MarkFailed(context.Context, string, string, time.Time) (bool, error)
	DeleteCompletedBefore(context.Context, time.Time, int) (int64, error)
}

type DeliveryProcessorConfig struct {
	Workers      int
	ScanBatch    int
	ScanInterval time.Duration
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
	Retention    time.Duration
	CleanupBatch int
}

func DefaultDeliveryProcessorConfig() DeliveryProcessorConfig {
	return DeliveryProcessorConfig{
		Workers:      1,
		ScanBatch:    16,
		ScanInterval: time.Second,
		MaxAttempts:  5,
		BaseBackoff:  time.Second,
		MaxBackoff:   time.Minute,
		Retention:    30 * 24 * time.Hour,
		CleanupBatch: 500,
	}
}

// DeliveryProcessor durably accepts deliveries and routes them through a fixed
// worker set. It never creates a goroutine per request or per delivery.
type DeliveryProcessor struct {
	repository WebhookDeliveryRepository
	routers    RouterProvider
	logger     *slog.Logger
	config     DeliveryProcessorConfig

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	done    chan struct{}
	wake    chan struct{}
	jobs    chan *models.GitHubWebhookDelivery
}

func NewDeliveryProcessor(
	repository WebhookDeliveryRepository,
	routers RouterProvider,
	logger *slog.Logger,
	config ...DeliveryProcessorConfig,
) *DeliveryProcessor {
	cfg := DefaultDeliveryProcessorConfig()
	if len(config) > 0 {
		cfg = normalizeDeliveryProcessorConfig(config[0])
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DeliveryProcessor{
		repository: repository,
		routers:    routers,
		logger:     logger,
		config:     cfg,
		wake:       make(chan struct{}, 1),
		jobs:       make(chan *models.GitHubWebhookDelivery, cfg.Workers),
	}
}

// Accept persists a delivery before returning. created=false indicates GitHub
// redelivered an already-known delivery ID.
func (p *DeliveryProcessor) Accept(ctx context.Context, deliveryID, eventType string, payload []byte) (bool, error) {
	if deliveryID == "" || eventType == "" {
		return false, fmt.Errorf("github delivery id and event type are required")
	}
	if int64(len(payload)) > MaxWebhookPayloadBytes {
		return false, fmt.Errorf("github webhook payload exceeds %d bytes", MaxWebhookPayloadBytes)
	}

	created, err := p.repository.Create(ctx, &models.GitHubWebhookDelivery{
		DeliveryID: deliveryID,
		EventType:  eventType,
		Payload:    append([]byte(nil), payload...),
		Status:     models.GitHubWebhookDeliveryQueued,
	})
	if err != nil {
		return false, err
	}
	if created {
		select {
		case p.wake <- struct{}{}:
		default:
		}
	}
	return created, nil
}

// Start recovers interrupted work before starting the bounded dispatcher and
// workers. A processor instance is intentionally single-use.
func (p *DeliveryProcessor) Start(parent context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return fmt.Errorf("github delivery processor already started")
	}
	if p.repository == nil || p.routers == nil {
		return fmt.Errorf("github delivery processor requires repository and router provider")
	}

	recovered, err := p.repository.RecoverProcessing(parent, time.Now().UTC(), p.config.MaxAttempts)
	if err != nil {
		return err
	}
	if recovered > 0 {
		p.logger.Info("recovered interrupted github webhook deliveries", "count", recovered)
	}

	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.started = true

	var group sync.WaitGroup
	group.Add(1 + p.config.Workers)
	go func() {
		defer group.Done()
		p.dispatch(ctx)
	}()
	for range p.config.Workers {
		go func() {
			defer group.Done()
			p.work(ctx)
		}()
	}
	go func(done chan struct{}) {
		group.Wait()
		close(done)
	}(p.done)

	select {
	case p.wake <- struct{}{}:
	default:
	}
	return nil
}

// Shutdown stops claiming new work and waits for fixed workers up to ctx's
// deadline. In-flight rows remain processing and are recovered on next start.
func (p *DeliveryProcessor) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	cancel, done := p.cancel, p.done
	p.mu.Unlock()

	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *DeliveryProcessor) dispatch(ctx context.Context) {
	ticker := time.NewTicker(p.config.ScanInterval)
	defer ticker.Stop()
	lastCleanup := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.wake:
			p.scan(ctx)
		case <-ticker.C:
			p.scan(ctx)
		}
		if lastCleanup.IsZero() || time.Since(lastCleanup) >= time.Hour {
			p.cleanup(ctx)
			lastCleanup = time.Now()
		}
	}
}

func (p *DeliveryProcessor) cleanup(ctx context.Context) {
	deleted, err := p.repository.DeleteCompletedBefore(ctx, time.Now().UTC().Add(-p.config.Retention), p.config.CleanupBatch)
	if err != nil {
		if ctx.Err() == nil {
			p.logger.Warn("failed to prune github webhook deliveries", "error", err)
		}
		return
	}
	if deleted > 0 {
		p.logger.Info("pruned completed github webhook deliveries", "count", deleted)
	}
}

func (p *DeliveryProcessor) scan(ctx context.Context) {
	for range p.config.ScanBatch {
		delivery, err := p.repository.ClaimNext(ctx, time.Now().UTC(), p.config.MaxAttempts)
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			if ctx.Err() == nil {
				p.logger.Error("failed to claim github webhook delivery", "error", err)
			}
			return
		}

		select {
		case p.jobs <- delivery:
		case <-ctx.Done():
			return
		}
	}
}

func (p *DeliveryProcessor) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery := <-p.jobs:
			p.process(ctx, delivery)
		}
	}
}

func (p *DeliveryProcessor) process(ctx context.Context, delivery *models.GitHubWebhookDelivery) {
	router, ok := p.routers.WebhookRouter()
	var routeErr error
	if !ok || router == nil {
		routeErr = fmt.Errorf("github event router is not ready")
	} else {
		routeErr = routeWebhookDelivery(ctx, router, delivery)
	}

	now := time.Now().UTC()
	// The route may have completed its external side effects at the same moment
	// shutdown cancelled the worker context. Persist the outcome with a bounded,
	// independent context so a successful delivery is not unnecessarily replayed.
	persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if routeErr == nil {
		changed, err := p.repository.MarkCompleted(persistCtx, delivery.DeliveryID, now)
		if err != nil {
			p.logger.Error("failed to complete github webhook delivery", "delivery_id", delivery.DeliveryID, "error", err)
		} else if !changed {
			p.logger.Warn("github webhook completion lost state race", "delivery_id", delivery.DeliveryID)
		}
		return
	}

	lastError := boundedError(routeErr)
	var permanentErr *PermanentWebhookError
	if errors.As(routeErr, &permanentErr) || delivery.Attempts >= p.config.MaxAttempts {
		changed, err := p.repository.MarkFailed(persistCtx, delivery.DeliveryID, lastError, now)
		if err != nil {
			p.logger.Error("failed to mark github webhook delivery terminal", "delivery_id", delivery.DeliveryID, "error", err)
		} else if changed {
			p.logger.Error("github webhook delivery failed terminally", "delivery_id", delivery.DeliveryID, "attempts", delivery.Attempts, "error", routeErr)
		}
		return
	}

	nextAttempt := now.Add(p.backoff(delivery.Attempts))
	changed, err := p.repository.MarkForRetry(persistCtx, delivery.DeliveryID, lastError, nextAttempt, now)
	if err != nil {
		p.logger.Error("failed to schedule github webhook delivery retry", "delivery_id", delivery.DeliveryID, "error", err)
	} else if changed {
		p.logger.Warn("github webhook delivery scheduled for retry", "delivery_id", delivery.DeliveryID, "attempt", delivery.Attempts, "error", routeErr)
	}
}

func routeWebhookDelivery(ctx context.Context, router DeliveryRouter, delivery *models.GitHubWebhookDelivery) (routeErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			routeErr = fmt.Errorf("github webhook handler panic: %v", recovered)
		}
	}()
	return router.Route(ctx, delivery.EventType, delivery.Payload, delivery.DeliveryID)
}

func (p *DeliveryProcessor) backoff(attempt int) time.Duration {
	delay := p.config.BaseBackoff
	for i := 1; i < attempt && delay < p.config.MaxBackoff; i++ {
		if delay > p.config.MaxBackoff/2 {
			return p.config.MaxBackoff
		}
		delay *= 2
	}
	if delay > p.config.MaxBackoff {
		return p.config.MaxBackoff
	}
	return delay
}

func normalizeDeliveryProcessorConfig(cfg DeliveryProcessorConfig) DeliveryProcessorConfig {
	defaults := DefaultDeliveryProcessorConfig()
	if cfg.Workers <= 0 {
		cfg.Workers = defaults.Workers
	}
	if cfg.ScanBatch <= 0 {
		cfg.ScanBatch = defaults.ScanBatch
	}
	if cfg.ScanInterval <= 0 {
		cfg.ScanInterval = defaults.ScanInterval
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaults.MaxAttempts
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = defaults.BaseBackoff
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		cfg.MaxBackoff = cfg.BaseBackoff
	}
	if cfg.Retention <= 0 {
		cfg.Retention = defaults.Retention
	}
	if cfg.CleanupBatch <= 0 {
		cfg.CleanupBatch = defaults.CleanupBatch
	}
	return cfg
}

func boundedError(err error) string {
	const maxErrorBytes = 2048
	message := err.Error()
	if len(message) > maxErrorBytes {
		return message[:maxErrorBytes]
	}
	return message
}
