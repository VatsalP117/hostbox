package repository

import (
	"context"
	"testing"
)

func TestSettingsRepository_GetDefaults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	val, err := repo.Get(ctx, "setup_complete")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "false" {
		t.Errorf("setup_complete = %q, want 'false'", val)
	}
}

func TestSettingsRepository_Set(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "setup_complete", "true"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, _ := repo.Get(ctx, "setup_complete")
	if val != "true" {
		t.Errorf("setup_complete = %q, want 'true'", val)
	}
}

func TestSettingsRepository_GetAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 6 {
		t.Errorf("len = %d, want 6 default settings", len(all))
	}
	if all["max_concurrent_builds"] != "1" {
		t.Errorf("max_concurrent_builds = %q, want '1'", all["max_concurrent_builds"])
	}
}

func TestSettingsRepository_GetBool(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	b, err := repo.GetBool(ctx, "registration_enabled")
	if err != nil {
		t.Fatalf("GetBool: %v", err)
	}
	if b {
		t.Error("expected false")
	}

	repo.Set(ctx, "registration_enabled", "true")
	b, _ = repo.GetBool(ctx, "registration_enabled")
	if !b {
		t.Error("expected true after set")
	}
}

func TestSettingsRepository_GetInt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	n, err := repo.GetInt(ctx, "max_projects")
	if err != nil {
		t.Fatalf("GetInt: %v", err)
	}
	if n != 50 {
		t.Errorf("max_projects = %d, want 50", n)
	}
}

func TestSettingsRepository_SetNewKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	repo.Set(ctx, "custom_key", "custom_value")
	val, _ := repo.Get(ctx, "custom_key")
	if val != "custom_value" {
		t.Errorf("custom_key = %q, want 'custom_value'", val)
	}
}
