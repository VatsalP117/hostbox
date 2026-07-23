package github

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/migrations"
)

type deliveryRouterFunc func(context.Context, string, []byte, string) error

func (f deliveryRouterFunc) Route(ctx context.Context, event string, payload []byte, deliveryID string) error {
	return f(ctx, event, payload, deliveryID)
}

func TestDeliveryProcessorAcceptDuplicateExecutesOnce(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	var calls atomic.Int32
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(context.Context, string, []byte, string) error {
		calls.Add(1)
		return nil
	}))

	created, err := processor.Accept(context.Background(), "delivery-1", "ping", []byte(`{}`))
	if err != nil || !created {
		t.Fatalf("first Accept() = (%v, %v)", created, err)
	}
	created, err = processor.Accept(context.Background(), "delivery-1", "ping", []byte(`{"duplicate":true}`))
	if err != nil || created {
		t.Fatalf("duplicate Accept() = (%v, %v)", created, err)
	}
	startTestDeliveryProcessor(t, processor)

	waitForWebhookStatus(t, repo, "delivery-1", models.GitHubWebhookDeliveryCompleted)
	if got := calls.Load(); got != 1 {
		t.Fatalf("router calls = %d, want 1", got)
	}
}

func TestDeliveryProcessorRetriesThenCompletes(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	var calls atomic.Int32
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(context.Context, string, []byte, string) error {
		if calls.Add(1) == 1 {
			return errors.New("temporary failure")
		}
		return nil
	}))
	startTestDeliveryProcessor(t, processor)
	if _, err := processor.Accept(context.Background(), "delivery-retry", "push", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}

	delivery := waitForWebhookStatus(t, repo, "delivery-retry", models.GitHubWebhookDeliveryCompleted)
	if delivery.Attempts != 2 || calls.Load() != 2 {
		t.Fatalf("completed attempts=%d calls=%d, want 2,2", delivery.Attempts, calls.Load())
	}
}

func TestDeliveryProcessorFailsAfterBoundedAttempts(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	var calls atomic.Int32
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(context.Context, string, []byte, string) error {
		calls.Add(1)
		return errors.New("permanent failure")
	}))
	startTestDeliveryProcessor(t, processor)
	if _, err := processor.Accept(context.Background(), "delivery-fail", "push", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}

	delivery := waitForWebhookStatus(t, repo, "delivery-fail", models.GitHubWebhookDeliveryFailed)
	if delivery.Attempts != 2 || calls.Load() != 2 {
		t.Fatalf("failed attempts=%d calls=%d, want 2,2", delivery.Attempts, calls.Load())
	}
	if delivery.LastError == nil || *delivery.LastError != "permanent failure" {
		t.Fatalf("last error = %v", delivery.LastError)
	}
}

func TestDeliveryProcessorDoesNotRetryPermanentWebhookError(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	var calls atomic.Int32
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(context.Context, string, []byte, string) error {
		calls.Add(1)
		return NewPermanentWebhookError("unsupported fork pull request")
	}))
	startTestDeliveryProcessor(t, processor)
	if _, err := processor.Accept(context.Background(), "delivery-permanent", "pull_request", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}

	delivery := waitForWebhookStatus(t, repo, "delivery-permanent", models.GitHubWebhookDeliveryFailed)
	if delivery.Attempts != 1 || calls.Load() != 1 {
		t.Fatalf("failed attempts=%d calls=%d, want 1,1", delivery.Attempts, calls.Load())
	}
	if delivery.LastError == nil || *delivery.LastError != "unsupported fork pull request" {
		t.Fatalf("last error = %v", delivery.LastError)
	}
}

func TestDeliveryProcessorRecoversFromRouterPanicAndKeepsWorkerAlive(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	var calls atomic.Int32
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(_ context.Context, _ string, _ []byte, deliveryID string) error {
		calls.Add(1)
		if deliveryID == "delivery-panic" {
			panic("boom")
		}
		return nil
	}))
	startTestDeliveryProcessor(t, processor)
	if _, err := processor.Accept(context.Background(), "delivery-panic", "push", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Accept(context.Background(), "delivery-after-panic", "ping", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}

	failed := waitForWebhookStatus(t, repo, "delivery-panic", models.GitHubWebhookDeliveryFailed)
	waitForWebhookStatus(t, repo, "delivery-after-panic", models.GitHubWebhookDeliveryCompleted)
	if failed.LastError == nil || !strings.Contains(*failed.LastError, "handler panic: boom") {
		t.Fatalf("panic error = %v", failed.LastError)
	}
	if calls.Load() < 3 {
		t.Fatalf("router calls = %d, want panic retries plus subsequent delivery", calls.Load())
	}
}

func TestDeliveryProcessorRecoversInterruptedDelivery(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	ctx := context.Background()
	if _, err := repo.Create(ctx, &models.GitHubWebhookDelivery{
		DeliveryID: "delivery-recovered",
		EventType:  "ping",
		Payload:    []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, time.Now().UTC().Add(time.Second), 2); err != nil {
		t.Fatal(err)
	}

	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(context.Context, string, []byte, string) error { return nil }))
	startTestDeliveryProcessor(t, processor)
	delivery := waitForWebhookStatus(t, repo, "delivery-recovered", models.GitHubWebhookDeliveryCompleted)
	if delivery.Attempts != 2 {
		t.Fatalf("recovered attempts = %d, want 2", delivery.Attempts)
	}
}

func TestDeliveryProcessorUnavailableRouterIsRetried(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	cfg := testDeliveryProcessorConfig()
	processor := NewDeliveryProcessor(repo, RouterProviderFunc(func() (DeliveryRouter, bool) {
		return nil, false
	}), slog.Default(), cfg)
	startTestDeliveryProcessor(t, processor)
	if _, err := processor.Accept(context.Background(), "delivery-no-router", "ping", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	delivery := waitForWebhookStatus(t, repo, "delivery-no-router", models.GitHubWebhookDeliveryFailed)
	if delivery.Attempts != cfg.MaxAttempts {
		t.Fatalf("attempts = %d, want %d", delivery.Attempts, cfg.MaxAttempts)
	}
}

func TestDeliveryProcessorShutdownHonorsDeadline(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	blocked := make(chan struct{})
	started := make(chan struct{})
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(context.Context, string, []byte, string) error {
		close(started)
		<-blocked
		return nil
	}))
	if err := processor.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Accept(context.Background(), "delivery-blocked", "ping", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("router did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := processor.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want deadline exceeded", err)
	}
	close(blocked)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := processor.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
}

func TestDeliveryProcessorShutdownCancelsRouterContext(t *testing.T) {
	repo := deliveryProcessorTestRepository(t)
	started := make(chan struct{})
	processor := newTestDeliveryProcessor(repo, deliveryRouterFunc(func(ctx context.Context, _ string, _ []byte, _ string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}))
	if err := processor.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Accept(context.Background(), "delivery-cancel", "ping", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("router did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := processor.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	delivery, err := repo.GetByDeliveryID(context.Background(), "delivery-cancel")
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != models.GitHubWebhookDeliveryQueued {
		t.Fatalf("cancelled delivery status = %q, want queued", delivery.Status)
	}
}

func deliveryProcessorTestRepository(t *testing.T) *repository.GitHubWebhookDeliveryRepository {
	t.Helper()
	db, err := database.Open(t.TempDir() + "/processor.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return repository.NewGitHubWebhookDeliveryRepository(db)
}

func newTestDeliveryProcessor(repo *repository.GitHubWebhookDeliveryRepository, router DeliveryRouter) *DeliveryProcessor {
	return NewDeliveryProcessor(repo, RouterProviderFunc(func() (DeliveryRouter, bool) {
		return router, true
	}), slog.Default(), testDeliveryProcessorConfig())
}

func testDeliveryProcessorConfig() DeliveryProcessorConfig {
	return DeliveryProcessorConfig{
		Workers:      1,
		ScanBatch:    2,
		ScanInterval: time.Millisecond,
		MaxAttempts:  2,
		BaseBackoff:  time.Millisecond,
		MaxBackoff:   2 * time.Millisecond,
	}
}

func startTestDeliveryProcessor(t *testing.T, processor *DeliveryProcessor) {
	t.Helper()
	if err := processor.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := processor.Shutdown(ctx); err != nil {
			t.Errorf("processor shutdown: %v", err)
		}
	})
}

func waitForWebhookStatus(
	t *testing.T,
	repo *repository.GitHubWebhookDeliveryRepository,
	deliveryID string,
	want models.GitHubWebhookDeliveryStatus,
) *models.GitHubWebhookDelivery {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		delivery, err := repo.GetByDeliveryID(context.Background(), deliveryID)
		if err == nil && delivery.Status == want {
			return delivery
		}
		time.Sleep(time.Millisecond)
	}
	delivery, err := repo.GetByDeliveryID(context.Background(), deliveryID)
	t.Fatalf("delivery %q did not reach %q: delivery=%#v error=%v", deliveryID, want, delivery, err)
	return nil
}
