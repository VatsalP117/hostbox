package worker

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	dockerpkg "github.com/VatsalP117/hostbox/internal/platform/docker"
	"github.com/VatsalP117/hostbox/internal/repository"
)

// Pool is a bounded goroutine worker pool that processes deployment build jobs.
type Pool struct {
	jobs       chan string
	maxWorkers int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	executor   *BuildExecutor
	deployRepo *repository.DeploymentRepository
	docker     dockerpkg.DockerClient
}

const queueReconcileInterval = 250 * time.Millisecond

// NewPool creates a worker pool. Call Start() to begin processing.
func NewPool(
	maxWorkers int,
	bufferSize int,
	executor *BuildExecutor,
	deployRepo *repository.DeploymentRepository,
	docker dockerpkg.DockerClient,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		jobs:       make(chan string, bufferSize),
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
		executor:   executor,
		deployRepo: deployRepo,
		docker:     docker,
	}
}

// Start launches worker goroutines and performs crash recovery.
func (p *Pool) Start() error {
	if err := p.recoverCrashedBuilds(); err != nil {
		slog.Error("worker pool: crash recovery failed", "err", err)
	}

	p.cleanOrphanedContainers()

	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.wg.Add(1)
	go p.dispatchQueued()
	slog.Info("worker pool started", "workers", p.maxWorkers)

	p.dispatchOnce()

	return nil
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	slog.Debug("worker started", "worker_id", id)

	for {
		select {
		case <-p.ctx.Done():
			slog.Debug("worker stopping", "worker_id", id)
			return
		case deploymentID, ok := <-p.jobs:
			if !ok {
				return
			}
			slog.Info("worker picked up job", "worker_id", id, "deployment_id", deploymentID)
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("worker panic recovered",
							"worker_id", id,
							"deployment_id", deploymentID,
							"panic", r,
							"stack", string(debug.Stack()),
						)
						p.failActiveDeployment(deploymentID, "Internal error: build worker panic")
					}
				}()
				p.executor.Execute(p.ctx, deploymentID)
			}()
		}
	}
}

// Offer wakes a worker for a durable queued deployment without blocking the
// caller. The dispatcher will offer it again while the database row is queued.
func (p *Pool) Offer(deploymentID string) bool {
	select {
	case p.jobs <- deploymentID:
		slog.Debug("job enqueued", "deployment_id", deploymentID)
		return true
	default:
		slog.Debug("job channel full; durable queue will retry", "deployment_id", deploymentID)
		return false
	case <-p.ctx.Done():
		slog.Warn("pool shutdown: dropping job", "deployment_id", deploymentID)
		return false
	}
}

// Shutdown gracefully stops the worker pool.
func (p *Pool) Shutdown(timeout time.Duration) {
	slog.Info("worker pool shutting down", "timeout", timeout)

	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("worker pool: all builds completed")
	case <-time.After(timeout):
		slog.Warn("worker pool: shutdown timeout, force-killing containers")
		p.forceKillContainers()
	}

}

func (p *Pool) recoverCrashedBuilds() error {
	ctx := context.Background()
	stuck, err := p.deployRepo.FindByStatus(ctx, "building")
	if err != nil {
		return err
	}

	for _, d := range stuck {
		slog.Warn("recovering stuck build", "deployment_id", d.ID)
		now := time.Now().UTC()
		d.Status = models.DeploymentStatusFailed
		errMsg := "Build interrupted by server restart"
		d.ErrorMessage = &errMsg
		d.CompletedAt = &now
		updated, err := p.deployRepo.UpdateIfStatus(ctx, &d, models.DeploymentStatusBuilding)
		if err != nil || !updated {
			slog.Error("failed to mark deployment as failed", "id", d.ID, "err", err)
		}
	}

	if len(stuck) > 0 {
		slog.Info("crash recovery complete", "recovered", len(stuck))
	}
	return nil
}

func (p *Pool) cleanOrphanedContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := p.docker.ListManagedContainers(ctx)
	if err != nil {
		slog.Warn("failed to list managed containers", "err", err)
		return
	}

	for _, c := range containers {
		slog.Info("removing orphaned container", "id", c.ID[:12], "name", c.Names)
		_ = p.docker.RemoveContainer(ctx, c.ID)
	}
}

func (p *Pool) dispatchQueued() {
	defer p.wg.Done()
	ticker := time.NewTicker(queueReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.dispatchOnce()
		}
	}
}

func (p *Pool) dispatchOnce() {
	queued, err := p.deployRepo.FindByStatus(p.ctx, string(models.DeploymentStatusQueued))
	if err != nil {
		if p.ctx.Err() != nil {
			return
		}
		slog.Error("failed to load queued deployments", "err", err)
		return
	}

	for _, d := range queued {
		if !p.Offer(d.ID) {
			return
		}
	}
}

func (p *Pool) failActiveDeployment(deploymentID, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	deployment, err := p.deployRepo.GetByID(ctx, deploymentID)
	if err != nil {
		return
	}
	expected := deployment.Status
	if expected != models.DeploymentStatusQueued && expected != models.DeploymentStatusBuilding {
		return
	}
	now := time.Now().UTC()
	deployment.Status = models.DeploymentStatusFailed
	deployment.ErrorMessage = &message
	deployment.CompletedAt = &now
	_, _ = p.deployRepo.UpdateIfStatus(ctx, deployment, expected)
}

func (p *Pool) forceKillContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containers, err := p.docker.ListManagedContainers(ctx)
	if err != nil {
		return
	}
	for _, c := range containers {
		_ = p.docker.StopContainer(ctx, c.ID, 2*time.Second)
		_ = p.docker.RemoveContainer(ctx, c.ID)
	}
}
