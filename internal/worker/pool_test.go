package worker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/VatsalP117/hostbox/internal/database"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/migrations"
)

func TestPoolOfferDoesNotBlockWhenChannelIsFull(t *testing.T) {
	pool := &Pool{
		jobs: make(chan string, 1),
		ctx:  context.Background(),
	}
	if !pool.Offer("first") {
		t.Fatal("first offer should fit")
	}
	if pool.Offer("second") {
		t.Fatal("second offer should report saturation")
	}
}

func TestPoolDispatchOnceOffersDurableQueuedRows(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	userRepo := repository.NewUserRepository(db)
	user := &models.User{Email: "pool@hostbox.local", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	project := &models.Project{
		OwnerID: user.ID, Name: "Pool", Slug: "pool", ProductionBranch: "main",
		RootDirectory: "/", NodeVersion: "20",
	}
	if err := projectRepo.Create(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc", Branch: "main",
		Status: models.DeploymentStatusQueued,
	}
	if err := deploymentRepo.Create(context.Background(), deployment); err != nil {
		t.Fatal(err)
	}
	pool := &Pool{
		jobs:       make(chan string, 1),
		ctx:        context.Background(),
		deployRepo: deploymentRepo,
	}

	pool.dispatchOnce()
	select {
	case got := <-pool.jobs:
		if got != deployment.ID {
			t.Fatalf("offered ID = %q, want %q", got, deployment.ID)
		}
	default:
		t.Fatal("durable queued row was not offered")
	}
}
