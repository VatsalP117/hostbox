package repository

import (
	"context"
	"testing"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
)

func TestSystemMetricRepository_CreateListAndDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemMetricRepository(db)
	ctx := context.Background()

	oldSnapshot := &models.SystemMetricSnapshot{
		CPUUsagePercent:      21.5,
		MemoryUsedBytes:      100,
		MemoryTotalBytes:     200,
		MemoryAvailableBytes: 100,
		MemoryUsagePercent:   50,
		DiskUsedBytes:        300,
		DiskTotalBytes:       1000,
		DiskAvailableBytes:   700,
		DiskUsagePercent:     30,
		ActiveBuilds:         1,
		QueuedBuilds:         0,
		CreatedAt:            time.Now().UTC().Add(-2 * time.Hour),
	}
	newSnapshot := &models.SystemMetricSnapshot{
		CPUUsagePercent:      42.5,
		MemoryUsedBytes:      200,
		MemoryTotalBytes:     400,
		MemoryAvailableBytes: 200,
		MemoryUsagePercent:   50,
		DiskUsedBytes:        500,
		DiskTotalBytes:       1000,
		DiskAvailableBytes:   500,
		DiskUsagePercent:     50,
		ActiveBuilds:         2,
		QueuedBuilds:         3,
		CreatedAt:            time.Now().UTC(),
	}

	if err := repo.CreateSnapshot(ctx, oldSnapshot); err != nil {
		t.Fatalf("CreateSnapshot old: %v", err)
	}
	if err := repo.CreateSnapshot(ctx, newSnapshot); err != nil {
		t.Fatalf("CreateSnapshot new: %v", err)
	}

	snapshots, err := repo.ListSince(ctx, time.Now().UTC().Add(-90*time.Minute), 0)
	if err != nil {
		t.Fatalf("ListSince: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
	if snapshots[0].QueuedBuilds != 3 {
		t.Fatalf("queued builds = %d, want 3", snapshots[0].QueuedBuilds)
	}

	removed, err := repo.DeleteOlderThan(ctx, time.Now().UTC().Add(-90*time.Minute))
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
}

func TestDeploymentRepository_SummarizeSince(t *testing.T) {
	db := setupTestDB(t)
	_, project := createTestProject(t, db)
	repo := NewDeploymentRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	successDuration := int64(120000)
	failedMsg := "failed"
	successCompleted := now.Add(-10 * time.Minute)
	failedCompleted := now.Add(-5 * time.Minute)

	if err := repo.Create(ctx, &models.Deployment{
		ProjectID:       project.ID,
		CommitSHA:       "abc123def456abc123def456abc123def456abc1",
		Branch:          "main",
		Status:          models.DeploymentStatusReady,
		BuildDurationMs: &successDuration,
		CompletedAt:     &successCompleted,
	}); err != nil {
		t.Fatalf("create success deployment: %v", err)
	}

	if err := repo.Create(ctx, &models.Deployment{
		ProjectID:    project.ID,
		CommitSHA:    "def456abc123def456abc123def456abc123def4",
		Branch:       "main",
		Status:       models.DeploymentStatusFailed,
		ErrorMessage: &failedMsg,
		CompletedAt:  &failedCompleted,
	}); err != nil {
		t.Fatalf("create failed deployment: %v", err)
	}

	summary, err := repo.SummarizeSince(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("SummarizeSince: %v", err)
	}

	if summary.Total != 2 || summary.Successful != 1 || summary.Failed != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.AverageBuildDurationMs == nil || *summary.AverageBuildDurationMs != successDuration {
		t.Fatalf("avg build duration = %v, want %d", summary.AverageBuildDurationMs, successDuration)
	}
	if summary.LastSuccessAt == nil || !summary.LastSuccessAt.Equal(successCompleted.Truncate(time.Second)) {
		t.Fatalf("last success = %v, want %v", summary.LastSuccessAt, successCompleted)
	}
	if summary.LastFailureAt == nil || !summary.LastFailureAt.Equal(failedCompleted.Truncate(time.Second)) {
		t.Fatalf("last failure = %v, want %v", summary.LastFailureAt, failedCompleted)
	}
}
