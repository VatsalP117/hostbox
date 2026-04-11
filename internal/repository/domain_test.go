package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
)

func createTestProjectForDomain(t *testing.T, db *sql.DB) *models.Project {
	t.Helper()
	ctx := context.Background()
	user := &models.User{Email: "domain@test.com", PasswordHash: "hash"}
	NewUserRepository(db).Create(ctx, user)
	project := &models.Project{
		OwnerID: user.ID, Name: "Domain Test", Slug: "domain-test",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	NewProjectRepository(db).Create(ctx, project)
	return project
}

func TestDomainRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	domain := &models.Domain{
		ProjectID: project.ID,
		Domain:    "example.com",
	}
	if err := repo.Create(ctx, domain); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, domain.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Domain != "example.com" {
		t.Errorf("Domain = %q, want 'example.com'", got.Domain)
	}
	if got.Verified {
		t.Error("expected Verified = false")
	}
}

func TestDomainRepository_GetByDomain(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Domain{ProjectID: project.ID, Domain: "find.com"})

	got, err := repo.GetByDomain(ctx, "find.com")
	if err != nil {
		t.Fatalf("GetByDomain: %v", err)
	}
	if got.Domain != "find.com" {
		t.Errorf("Domain = %q", got.Domain)
	}
}

func TestDomainRepository_UniqueDomain(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Domain{ProjectID: project.ID, Domain: "dup.com"})
	err := repo.Create(ctx, &models.Domain{ProjectID: project.ID, Domain: "dup.com"})
	if err == nil {
		t.Fatal("expected error for duplicate domain")
	}
}

func TestDomainRepository_UpdateVerification(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	domain := &models.Domain{ProjectID: project.ID, Domain: "verify.com"}
	repo.Create(ctx, domain)

	now := time.Now().UTC()
	if err := repo.UpdateVerification(ctx, domain.ID, true, &now); err != nil {
		t.Fatalf("UpdateVerification: %v", err)
	}

	got, _ := repo.GetByID(ctx, domain.ID)
	if !got.Verified {
		t.Error("expected Verified = true")
	}
	if got.VerifiedAt == nil {
		t.Error("expected VerifiedAt to be set")
	}
}

func TestDomainRepository_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Domain{ProjectID: project.ID, Domain: "a.com"})
	repo.Create(ctx, &models.Domain{ProjectID: project.ID, Domain: "b.com"})

	domains, err := repo.ListByProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("len = %d, want 2", len(domains))
	}
}

func TestDomainRepository_ListUnverified(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &models.Domain{ProjectID: project.ID, Domain: "unverified.com"})
	d2 := &models.Domain{ProjectID: project.ID, Domain: "verified.com"}
	repo.Create(ctx, d2)
	now := time.Now().UTC()
	repo.UpdateVerification(ctx, d2.ID, true, &now)

	unverified, err := repo.ListUnverified(ctx)
	if err != nil {
		t.Fatalf("ListUnverified: %v", err)
	}
	if len(unverified) != 1 {
		t.Errorf("len = %d, want 1", len(unverified))
	}
}

func TestDomainRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	project := createTestProjectForDomain(t, db)
	repo := NewDomainRepository(db)
	ctx := context.Background()

	domain := &models.Domain{ProjectID: project.ID, Domain: "del.com"}
	repo.Create(ctx, domain)
	repo.Delete(ctx, domain.ID)

	_, err := repo.GetByID(ctx, domain.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
