package worker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/VatsalP117/hostbox/internal/config"
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

func TestPoolRecoverCrashedBuildRemovesIncompleteWorkspace(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "recovery.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	userRepo := repository.NewUserRepository(db)
	user := &models.User{Email: "recovery@hostbox.local", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	project := &models.Project{
		OwnerID: user.ID, Name: "Recovery", Slug: "recovery",
		ProductionBranch: "main", RootDirectory: "/", NodeVersion: "20",
	}
	if err := projectRepo.Create(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	deploymentRepo := repository.NewDeploymentRepository(db)
	deployment := &models.Deployment{
		ProjectID: project.ID, CommitSHA: "abc", Branch: "main",
		Status: models.DeploymentStatusBuilding,
	}
	if err := deploymentRepo.Create(context.Background(), deployment); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	deploymentsDir := filepath.Join(base, "deployments")
	clonesDir := filepath.Join(base, "clones")
	artifactDir := filepath.Join(deploymentsDir, project.ID, deployment.ID)
	cloneDir := filepath.Join(clonesDir, "clone-"+deployment.ID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatal(err)
	}
	executor := &BuildExecutor{cfg: &config.BuildConfig{
		DeploymentBaseDir: deploymentsDir,
		CloneBaseDir:      clonesDir,
	}}
	pool := &Pool{deployRepo: deploymentRepo, executor: executor}
	if err := pool.recoverCrashedBuilds(); err != nil {
		t.Fatal(err)
	}
	stored, err := deploymentRepo.GetByID(context.Background(), deployment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.DeploymentStatusFailed {
		t.Fatalf("status = %q, want failed", stored.Status)
	}
	for _, path := range []string{artifactDir, cloneDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("workspace %q still exists: %v", path, err)
		}
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
