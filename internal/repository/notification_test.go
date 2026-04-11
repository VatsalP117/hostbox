package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vatsalpatel/hostbox/internal/models"
)

func createTestProjectForNotif(t *testing.T, db *sql.DB) *models.Project {
	t.Helper()
	ctx := context.Background()
	user := &models.User{Email: "notif@test.com", PasswordHash: "hash"}
	NewUserRepository(db).Create(ctx, user)
	project := &models.Project{
		OwnerID: user.ID, Name: "Notif Test", Slug: "notif-test",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	NewProjectRepository(db).Create(ctx, project)
	return project
}

func TestNotificationRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForNotif(t, db)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	config := &models.NotificationConfig{
		ProjectID:  &project.ID,
		Channel:    "discord",
		WebhookURL: "https://discord.com/api/webhooks/123",
		Events:     "all",
		Enabled:    true,
	}
	if err := repo.Create(ctx, config); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, config.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Channel != "discord" {
		t.Errorf("Channel = %q, want 'discord'", got.Channel)
	}
}

func TestNotificationRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForNotif(t, db)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	config := &models.NotificationConfig{
		ProjectID: &project.ID, Channel: "slack",
		WebhookURL: "https://hooks.slack.com/old", Events: "all", Enabled: true,
	}
	repo.Create(ctx, config)

	config.WebhookURL = "https://hooks.slack.com/new"
	config.Enabled = false
	if err := repo.Update(ctx, config); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, config.ID)
	if got.WebhookURL != "https://hooks.slack.com/new" {
		t.Errorf("WebhookURL = %q", got.WebhookURL)
	}
	if got.Enabled {
		t.Error("expected Enabled = false")
	}
}

func TestNotificationRepository_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForNotif(t, db)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.NotificationConfig{
		ProjectID: &project.ID, Channel: "discord",
		WebhookURL: "https://example.com/1", Events: "all", Enabled: true,
	})
	repo.Create(ctx, &models.NotificationConfig{
		ProjectID: &project.ID, Channel: "slack",
		WebhookURL: "https://example.com/2", Events: "all", Enabled: true,
	})

	configs, err := repo.ListByProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("len = %d, want 2", len(configs))
	}
}

func TestNotificationRepository_ListGlobal(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForNotif(t, db)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	// Global notification (no project_id)
	repo.Create(ctx, &models.NotificationConfig{
		Channel: "webhook", WebhookURL: "https://global.com", Events: "all", Enabled: true,
	})
	// Project notification
	repo.Create(ctx, &models.NotificationConfig{
		ProjectID: &project.ID, Channel: "discord",
		WebhookURL: "https://project.com", Events: "all", Enabled: true,
	})

	globals, err := repo.ListGlobal(ctx)
	if err != nil {
		t.Fatalf("ListGlobal: %v", err)
	}
	if len(globals) != 1 {
		t.Errorf("len = %d, want 1", len(globals))
	}
}

func TestNotificationRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForNotif(t, db)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	config := &models.NotificationConfig{
		ProjectID: &project.ID, Channel: "discord",
		WebhookURL: "https://del.com", Events: "all", Enabled: true,
	}
	repo.Create(ctx, config)
	repo.Delete(ctx, config.ID)

	_, err := repo.GetByID(ctx, config.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
