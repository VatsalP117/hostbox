package repository

import (
	"context"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
)

func TestActivityRepository_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewActivityRepository(db)
	ctx := context.Background()

	entry := &models.ActivityLog{
		Action:       "deployment.created",
		ResourceType: "deployment",
		ResourceID:   strPtr("dep-123"),
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected ID to be set (autoincrement)")
	}

	entries, total, err := repo.List(ctx, 1, 10, nil, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}
}

func TestActivityRepository_ListWithFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := NewActivityRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.ActivityLog{Action: "deployment.created", ResourceType: "deployment"})
	repo.Create(ctx, &models.ActivityLog{Action: "project.created", ResourceType: "project"})
	repo.Create(ctx, &models.ActivityLog{Action: "deployment.failed", ResourceType: "deployment"})

	action := "deployment.created"
	entries, total, _ := repo.List(ctx, 1, 10, &action, nil)
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}

	resType := "deployment"
	entries, total, _ = repo.List(ctx, 1, 10, nil, &resType)
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestActivityRepository_ListByResource(t *testing.T) {
	db := setupTestDB(t)
	repo := NewActivityRepository(db)
	ctx := context.Background()

	id := "proj-abc"
	repo.Create(ctx, &models.ActivityLog{Action: "project.created", ResourceType: "project", ResourceID: &id})
	repo.Create(ctx, &models.ActivityLog{Action: "project.updated", ResourceType: "project", ResourceID: &id})
	repo.Create(ctx, &models.ActivityLog{Action: "deployment.created", ResourceType: "deployment"})

	entries, total, err := repo.ListByResource(ctx, "project", id, 1, 10)
	if err != nil {
		t.Fatalf("ListByResource: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(entries) != 2 {
		t.Errorf("len = %d, want 2", len(entries))
	}
}

func strPtr(s string) *string {
	return &s
}
