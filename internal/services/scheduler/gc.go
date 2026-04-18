package scheduler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
)

// GarbageCollector cleans up old deployment artifacts, orphaned logs, and Docker resources.
type GarbageCollector struct {
	deployRepo  *repository.DeploymentRepository
	projectRepo *repository.ProjectRepository
	settingsRepo *repository.SettingsRepository
	logDir      string
	logger      *slog.Logger
}

func NewGarbageCollector(
	deployRepo *repository.DeploymentRepository,
	projectRepo *repository.ProjectRepository,
	settingsRepo *repository.SettingsRepository,
	logDir string,
	logger *slog.Logger,
) *GarbageCollector {
	return &GarbageCollector{
		deployRepo:  deployRepo,
		projectRepo: projectRepo,
		settingsRepo: settingsRepo,
		logDir:      logDir,
		logger:      logger,
	}
}

func (gc *GarbageCollector) Run(ctx context.Context) {
	gc.Collect(ctx)

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			gc.logger.Info("garbage collector stopped")
			return
		case <-ticker.C:
			gc.Collect(ctx)
		}
	}
}

func (gc *GarbageCollector) Collect(ctx context.Context) {
	gc.logger.Info("garbage collection started")
	start := time.Now()

	artifactsRemoved := gc.collectArtifacts(ctx)
	logsRemoved := gc.collectOrphanedLogs(ctx)

	gc.logger.Info("garbage collection completed",
		"duration", time.Since(start).Round(time.Millisecond),
		"artifacts_removed", artifactsRemoved,
		"orphaned_logs_removed", logsRemoved,
	)
}

func (gc *GarbageCollector) collectArtifacts(ctx context.Context) int {
	maxDeployments := 10
	retentionDays := 30

	if v, err := gc.settingsRepo.Get(ctx, "max_deployments_per_project"); err == nil && v != "" {
		if n := parseInt(v); n > 0 {
			maxDeployments = n
		}
	}
	if v, err := gc.settingsRepo.Get(ctx, "artifact_retention_days"); err == nil && v != "" {
		if n := parseInt(v); n > 0 {
			retentionDays = n
		}
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	removed := 0

	projects, err := gc.projectRepo.ListAll(ctx)
	if err != nil {
		gc.logger.Error("gc: failed to list projects", "error", err)
		return 0
	}

	for _, project := range projects {
		removed += gc.collectProjectArtifacts(ctx, project, maxDeployments, cutoff)
	}
	return removed
}

func (gc *GarbageCollector) collectProjectArtifacts(ctx context.Context, project models.Project, maxDeployments int, cutoff time.Time) int {
	deployments, err := gc.deployRepo.ListAllByProject(ctx, project.ID)
	if err != nil {
		gc.logger.Error("gc: failed to list deployments", "project_id", project.ID, "error", err)
		return 0
	}

	protected := map[string]bool{}

	// Protect current production deployment
	for _, d := range deployments {
		if d.Status == models.DeploymentStatusReady && d.IsProduction {
			protected[d.ID] = true
			break
		}
	}

	// Protect latest ready deployment per branch
	seenBranches := map[string]bool{}
	for _, d := range deployments {
		if d.Status == models.DeploymentStatusReady && !seenBranches[d.Branch] {
			if d.ArtifactPath != nil && *d.ArtifactPath != "" {
				protected[d.ID] = true
				seenBranches[d.Branch] = true
			}
		}
	}

	removed := 0
	kept := 0
	for _, d := range deployments {
		if protected[d.ID] {
			continue
		}
		if d.ArtifactPath == nil || *d.ArtifactPath == "" {
			continue
		}

		kept++
		if kept <= maxDeployments {
			continue
		}

		if d.CreatedAt.After(cutoff) && kept <= maxDeployments*2 {
			continue
		}

		if d.ArtifactPath != nil {
			os.RemoveAll(*d.ArtifactPath)
		}
		if d.LogPath != nil {
			os.Remove(*d.LogPath)
		}

		if err := gc.deployRepo.ClearArtifact(ctx, d.ID); err != nil {
			gc.logger.Error("gc: failed to clear artifact", "deployment_id", d.ID, "error", err)
		}
		removed++
	}
	return removed
}

func (gc *GarbageCollector) collectOrphanedLogs(ctx context.Context) int {
	if gc.logDir == "" {
		return 0
	}
	entries, err := os.ReadDir(gc.logDir)
	if err != nil {
		return 0
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		deploymentID := strings.TrimSuffix(entry.Name(), ".log")
		exists, err := gc.deployRepo.HasLogPath(ctx, deploymentID)
		if err != nil || exists {
			continue
		}
		os.Remove(filepath.Join(gc.logDir, entry.Name()))
		removed++
	}
	return removed
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
