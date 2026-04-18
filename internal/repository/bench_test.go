package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/migrations"
)

func setupBenchDB(b *testing.B) (*Repositories, func()) {
	b.Helper()
	dir := b.TempDir()
	db, err := database.Open(dir + "/bench.db")
	if err != nil {
		b.Fatalf("failed to open bench db: %v", err)
	}
	if err := database.Migrate(db, migrations.FS); err != nil {
		b.Fatalf("failed to run migrations: %v", err)
	}
	repos := New(db)
	return repos, func() { db.Close() }
}

func seedUser(b *testing.B, repos *Repositories) string {
	b.Helper()
	ctx := context.Background()
	name := "Bench User"
	user := &models.User{
		Email:        "bench@example.com",
		PasswordHash: "hash",
		DisplayName:  &name,
		IsAdmin:      false,
	}
	if err := repos.User.Create(ctx, user); err != nil {
		b.Fatalf("create user: %v", err)
	}
	return user.ID
}

func seedProject(b *testing.B, repos *Repositories, ownerID, suffix string) string {
	b.Helper()
	ctx := context.Background()
	project := &models.Project{
		Name:    "project-" + suffix,
		Slug:    "project-" + suffix,
		OwnerID: ownerID,
	}
	if err := repos.Project.Create(ctx, project); err != nil {
		b.Fatalf("create project: %v", err)
	}
	return project.ID
}

func BenchmarkDeploymentList_100(b *testing.B) {
	repos, cleanup := setupBenchDB(b)
	defer cleanup()
	ctx := context.Background()

	userID := seedUser(b, repos)
	projectID := seedProject(b, repos, userID, "bench")

	for i := 0; i < 100; i++ {
		d := &models.Deployment{
			ProjectID: projectID,
			CommitSHA: fmt.Sprintf("abc%04d", i),
			Branch:    "main",
			Status:    models.DeploymentStatusReady,
		}
		if err := repos.Deployment.Create(ctx, d); err != nil {
			b.Fatalf("create deployment: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := repos.Deployment.ListByProject(ctx, projectID, 1, 20, nil, nil)
		if err != nil {
			b.Fatalf("list: %v", err)
		}
	}
}

func BenchmarkProjectList_100(b *testing.B) {
	repos, cleanup := setupBenchDB(b)
	defer cleanup()
	ctx := context.Background()

	userID := seedUser(b, repos)
	for i := 0; i < 100; i++ {
		seedProject(b, repos, userID, fmt.Sprintf("bench-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := repos.Project.ListByOwner(ctx, userID, 1, 20, "")
		if err != nil {
			b.Fatalf("list: %v", err)
		}
	}
}

func BenchmarkDeploymentCreate(b *testing.B) {
	repos, cleanup := setupBenchDB(b)
	defer cleanup()
	ctx := context.Background()

	userID := seedUser(b, repos)
	projectID := seedProject(b, repos, userID, "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := &models.Deployment{
			ProjectID: projectID,
			CommitSHA: fmt.Sprintf("sha%06d", i),
			Branch:    "main",
			Status:    models.DeploymentStatusQueued,
		}
		if err := repos.Deployment.Create(ctx, d); err != nil {
			b.Fatalf("create: %v", err)
		}
	}
}
