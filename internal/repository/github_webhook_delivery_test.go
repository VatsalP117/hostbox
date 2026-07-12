package repository

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

func TestGitHubWebhookDeliveryCreateIsIdempotent(t *testing.T) {
	repo := NewGitHubWebhookDeliveryRepository(setupTestDB(t))
	ctx := context.Background()
	delivery := testWebhookDelivery("delivery-1")

	created, err := repo.Create(ctx, delivery)
	if err != nil || !created {
		t.Fatalf("first Create() = (%v, %v), want (true, nil)", created, err)
	}
	duplicate := testWebhookDelivery("delivery-1")
	duplicate.Payload = []byte(`{"different":true}`)
	created, err = repo.Create(ctx, duplicate)
	if err != nil || created {
		t.Fatalf("duplicate Create() = (%v, %v), want (false, nil)", created, err)
	}

	stored, err := repo.GetByDeliveryID(ctx, "delivery-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.Payload) != `{}` {
		t.Fatalf("duplicate replaced payload: %s", stored.Payload)
	}
}

func TestGitHubWebhookDeliveryClaimIsAtomic(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGitHubWebhookDeliveryRepository(db)
	ctx := context.Background()
	if _, err := repo.Create(ctx, testWebhookDelivery("delivery-claim")); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.ClaimNext(ctx, time.Now().UTC().Add(time.Second), 3)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	claimed, empty := 0, 0
	for err := range results {
		switch err {
		case nil:
			claimed++
		case sql.ErrNoRows:
			empty++
		default:
			t.Fatalf("ClaimNext() unexpected error: %v", err)
		}
	}
	if claimed != 1 || empty != 1 {
		t.Fatalf("claim outcomes = claimed %d, empty %d; want 1,1", claimed, empty)
	}

	stored, err := repo.GetByDeliveryID(ctx, "delivery-claim")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.GitHubWebhookDeliveryProcessing || stored.Attempts != 1 {
		t.Fatalf("claimed delivery = status %q attempts %d", stored.Status, stored.Attempts)
	}
}

func TestGitHubWebhookDeliveryTransitionsRequireProcessing(t *testing.T) {
	repo := NewGitHubWebhookDeliveryRepository(setupTestDB(t))
	ctx := context.Background()
	if _, err := repo.Create(ctx, testWebhookDelivery("delivery-transition")); err != nil {
		t.Fatal(err)
	}

	changed, err := repo.MarkCompleted(ctx, "delivery-transition", time.Now().UTC())
	if err != nil || changed {
		t.Fatalf("complete queued delivery = (%v, %v), want (false, nil)", changed, err)
	}
	claimed, err := repo.ClaimNext(ctx, time.Now().UTC().Add(time.Second), 3)
	if err != nil {
		t.Fatal(err)
	}
	next := time.Now().UTC().Add(time.Minute)
	changed, err = repo.MarkForRetry(ctx, claimed.DeliveryID, "temporary", next, time.Now().UTC())
	if err != nil || !changed {
		t.Fatalf("MarkForRetry() = (%v, %v)", changed, err)
	}
	stored, _ := repo.GetByDeliveryID(ctx, claimed.DeliveryID)
	if stored.Status != models.GitHubWebhookDeliveryQueued || stored.LastError == nil || *stored.LastError != "temporary" {
		t.Fatalf("retry state = %#v", stored)
	}
	if _, err := repo.ClaimNext(ctx, time.Now().UTC(), 3); err != sql.ErrNoRows {
		t.Fatalf("future retry ClaimNext() error = %v, want sql.ErrNoRows", err)
	}
}

func TestGitHubWebhookDeliveryRecoveryQueuesOrFailsInterruptedRows(t *testing.T) {
	repo := NewGitHubWebhookDeliveryRepository(setupTestDB(t))
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := repo.Create(ctx, testWebhookDelivery("retryable")); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, now.Add(time.Second), 2); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, testWebhookDelivery("final")); err != nil {
		t.Fatal(err)
	}

	// Claim the second row once, retry it immediately, then claim its final attempt.
	second, err := repo.ClaimNext(ctx, now.Add(2*time.Second), 2)
	if err != nil {
		t.Fatal(err)
	}
	if second.DeliveryID != "final" {
		t.Fatalf("claimed %q, want final", second.DeliveryID)
	}
	if _, err := repo.MarkForRetry(ctx, second.DeliveryID, "retry", now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, now.Add(3*time.Second), 2); err != nil {
		t.Fatal(err)
	}

	recovered, err := repo.RecoverProcessing(ctx, now.Add(4*time.Second), 2)
	if err != nil || recovered != 2 {
		t.Fatalf("RecoverProcessing() = (%d, %v), want (2, nil)", recovered, err)
	}
	retryable, _ := repo.GetByDeliveryID(ctx, "retryable")
	final, _ := repo.GetByDeliveryID(ctx, "final")
	if retryable.Status != models.GitHubWebhookDeliveryQueued {
		t.Fatalf("retryable status = %q", retryable.Status)
	}
	if final.Status != models.GitHubWebhookDeliveryFailed {
		t.Fatalf("final status = %q", final.Status)
	}
}

func TestGitHubWebhookDeliveryPrunesOnlyOldCompletedRowsInBoundedBatches(t *testing.T) {
	repo := NewGitHubWebhookDeliveryRepository(setupTestDB(t))
	ctx := context.Background()
	now := time.Now().UTC()
	for _, id := range []string{"old-1", "old-2", "recent"} {
		if _, err := repo.Create(ctx, testWebhookDelivery(id)); err != nil {
			t.Fatal(err)
		}
		claimed, err := repo.ClaimNext(ctx, now.Add(time.Second), 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.MarkCompleted(ctx, claimed.DeliveryID, now); err != nil {
			t.Fatal(err)
		}
	}
	old := formatWebhookTime(now.Add(-31 * 24 * time.Hour))
	if _, err := repo.db.ExecContext(ctx, `UPDATE github_webhook_deliveries SET completed_at = ? WHERE delivery_id IN ('old-1', 'old-2')`, old); err != nil {
		t.Fatal(err)
	}

	deleted, err := repo.DeleteCompletedBefore(ctx, now.Add(-30*24*time.Hour), 1)
	if err != nil || deleted != 1 {
		t.Fatalf("first prune = (%d, %v), want (1, nil)", deleted, err)
	}
	deleted, err = repo.DeleteCompletedBefore(ctx, now.Add(-30*24*time.Hour), 10)
	if err != nil || deleted != 1 {
		t.Fatalf("second prune = (%d, %v), want (1, nil)", deleted, err)
	}
	if _, err := repo.GetByDeliveryID(ctx, "recent"); err != nil {
		t.Fatalf("recent completed delivery was pruned: %v", err)
	}
}

func testWebhookDelivery(id string) *models.GitHubWebhookDelivery {
	return &models.GitHubWebhookDelivery{
		DeliveryID: id,
		EventType:  "ping",
		Payload:    []byte(`{}`),
	}
}
