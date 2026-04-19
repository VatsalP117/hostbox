package admin

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/dto"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
)

const (
	trendWindow        = 24 * time.Hour
	snapshotRetention  = 7 * 24 * time.Hour
	deployHealthWindow = 24 * time.Hour
)

type projectCounter interface {
	Count(ctx context.Context) (int, error)
}

type userCounter interface {
	Count(ctx context.Context) (int, error)
}

type deploymentMetricsRepo interface {
	Count(ctx context.Context) (int, error)
	CountByStatuses(ctx context.Context, statuses ...string) (int, error)
	SummarizeSince(ctx context.Context, since time.Time) (repository.DeploymentHealthSummary, error)
}

type systemMetricStore interface {
	CreateSnapshot(ctx context.Context, snapshot *models.SystemMetricSnapshot) error
	ListSince(ctx context.Context, since time.Time, limit int) ([]models.SystemMetricSnapshot, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type dockerHealthChecker interface {
	Ping(ctx context.Context) error
}

type caddyHealthChecker interface {
	Healthy(ctx context.Context) bool
}

type MetricsService struct {
	db          *sql.DB
	cfg         *config.Config
	projectRepo projectCounter
	userRepo    userCounter
	deployRepo  deploymentMetricsRepo
	store       systemMetricStore
	docker      dockerHealthChecker
	caddy       caddyHealthChecker
	logger      *slog.Logger
	now         func() time.Time
}

func NewMetricsService(
	db *sql.DB,
	cfg *config.Config,
	projectRepo projectCounter,
	userRepo userCounter,
	deployRepo deploymentMetricsRepo,
	store systemMetricStore,
	docker dockerHealthChecker,
	caddy caddyHealthChecker,
	logger *slog.Logger,
) *MetricsService {
	return &MetricsService{
		db:          db,
		cfg:         cfg,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		deployRepo:  deployRepo,
		store:       store,
		docker:      docker,
		caddy:       caddy,
		logger:      logger,
		now:         time.Now,
	}
}

func (s *MetricsService) GetStats(ctx context.Context, uptimeSeconds int64) (*dto.AdminStatsResponse, error) {
	now := s.now().UTC()

	projectCount, err := s.projectRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	deploymentCount, err := s.deployRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	activeBuilds, err := s.deployRepo.CountByStatuses(ctx, string(models.DeploymentStatusBuilding))
	if err != nil {
		return nil, err
	}
	queuedBuilds, err := s.deployRepo.CountByStatuses(ctx, string(models.DeploymentStatusQueued))
	if err != nil {
		return nil, err
	}
	userCount, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	componentHealth := s.componentHealth(ctx)
	snapshot := s.collectCurrentSnapshot(ctx, int64(activeBuilds), int64(queuedBuilds), now)
	diskUsage := s.getDiskUsage(snapshot)
	deploySummary, err := s.deployRepo.SummarizeSince(ctx, now.Add(-deployHealthWindow))
	if err != nil {
		return nil, err
	}
	trends, err := s.buildTrends(ctx, now, snapshot)
	if err != nil {
		return nil, err
	}

	buildQueue := dto.BuildQueueResponse{
		ActiveBuilds:        int64(activeBuilds),
		QueuedBuilds:        int64(queuedBuilds),
		MaxConcurrentBuilds: int64(s.cfg.MaxConcurrentBuilds),
		Saturated:           s.cfg.MaxConcurrentBuilds > 0 && activeBuilds >= s.cfg.MaxConcurrentBuilds,
	}
	if s.cfg.MaxConcurrentBuilds > 0 {
		buildQueue.UtilizationPercent = round1((float64(activeBuilds) / float64(s.cfg.MaxConcurrentBuilds)) * 100)
	}

	deploymentHealth := dto.DeploymentHealthResponse{
		WindowHours:            int(deployHealthWindow.Hours()),
		Total:                  deploySummary.Total,
		Successful:             deploySummary.Successful,
		Failed:                 deploySummary.Failed,
		Cancelled:              deploySummary.Cancelled,
		AverageBuildDurationMs: deploySummary.AverageBuildDurationMs,
	}
	if deploySummary.Total > 0 {
		deploymentHealth.SuccessRate = round1((float64(deploySummary.Successful) / float64(deploySummary.Total)) * 100)
	}
	if deploySummary.LastSuccessAt != nil {
		ts := deploySummary.LastSuccessAt.UTC().Format(time.RFC3339)
		deploymentHealth.LastSuccessAt = &ts
	}
	if deploySummary.LastFailureAt != nil {
		ts := deploySummary.LastFailureAt.UTC().Format(time.RFC3339)
		deploymentHealth.LastFailureAt = &ts
	}

	stats := &dto.AdminStatsResponse{
		ProjectCount:        int64(projectCount),
		DeploymentCount:     int64(deploymentCount),
		ActiveBuilds:        int64(activeBuilds),
		QueuedBuilds:        int64(queuedBuilds),
		MaxConcurrentBuilds: int64(s.cfg.MaxConcurrentBuilds),
		UserCount:           int64(userCount),
		DiskUsage:           diskUsage,
		Components:          componentHealth,
		CPU: dto.CPUStatsResponse{
			UsagePercent: round1(snapshot.CPUUsagePercent),
			Cores:        cpuCount(),
			Load1:        round2(snapshot.Load1),
			Load5:        round2(snapshot.Load5),
			Load15:       round2(snapshot.Load15),
		},
		Memory: dto.MemoryStatsResponse{
			UsedBytes:      snapshot.MemoryUsedBytes,
			TotalBytes:     snapshot.MemoryTotalBytes,
			AvailableBytes: snapshot.MemoryAvailableBytes,
			UsagePercent:   round1(snapshot.MemoryUsagePercent),
		},
		BuildQueue:       buildQueue,
		DeploymentHealth: deploymentHealth,
		Trends:           trends,
		UptimeSeconds:    uptimeSeconds,
	}
	stats.Alerts = deriveAlerts(stats)

	return stats, nil
}

func (s *MetricsService) RecordSnapshot(ctx context.Context) error {
	now := s.now().UTC()
	activeBuilds, err := s.deployRepo.CountByStatuses(ctx, string(models.DeploymentStatusBuilding))
	if err != nil {
		return err
	}
	queuedBuilds, err := s.deployRepo.CountByStatuses(ctx, string(models.DeploymentStatusQueued))
	if err != nil {
		return err
	}

	snapshot := s.collectCurrentSnapshot(ctx, int64(activeBuilds), int64(queuedBuilds), now)
	if err := s.store.CreateSnapshot(ctx, &snapshot); err != nil {
		return err
	}
	if _, err := s.store.DeleteOlderThan(ctx, now.Add(-snapshotRetention)); err != nil {
		return err
	}

	return nil
}

func (s *MetricsService) componentHealth(ctx context.Context) dto.ComponentHealthResponse {
	components := dto.ComponentHealthResponse{
		API:      dto.ServiceHealthResponse{Status: "healthy"},
		Database: dto.ServiceHealthResponse{Status: "healthy"},
		Docker:   dto.ServiceHealthResponse{Status: "unavailable", Message: "docker client not configured"},
		Caddy:    dto.ServiceHealthResponse{Status: "degraded"},
	}

	if err := s.db.PingContext(ctx); err != nil {
		components.Database = dto.ServiceHealthResponse{Status: "degraded", Message: err.Error()}
	}
	if s.docker != nil {
		if err := s.docker.Ping(ctx); err != nil {
			components.Docker = dto.ServiceHealthResponse{Status: "degraded", Message: err.Error()}
		} else {
			components.Docker = dto.ServiceHealthResponse{Status: "healthy"}
		}
	}
	if s.caddy != nil {
		if s.caddy.Healthy(ctx) {
			components.Caddy = dto.ServiceHealthResponse{Status: "healthy"}
		} else {
			components.Caddy = dto.ServiceHealthResponse{Status: "degraded", Message: "caddy admin API not reachable"}
		}
	} else {
		components.Caddy = dto.ServiceHealthResponse{Status: "unavailable", Message: "caddy client not configured"}
	}

	return components
}

func (s *MetricsService) collectCurrentSnapshot(ctx context.Context, activeBuilds, queuedBuilds int64, now time.Time) models.SystemMetricSnapshot {
	snapshot := models.SystemMetricSnapshot{
		ActiveBuilds: activeBuilds,
		QueuedBuilds: queuedBuilds,
		CreatedAt:    now,
	}

	if usage, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false); err == nil && len(usage) > 0 {
		snapshot.CPUUsagePercent = usage[0]
	}
	if avg, err := load.AvgWithContext(ctx); err == nil {
		snapshot.Load1 = avg.Load1
		snapshot.Load5 = avg.Load5
		snapshot.Load15 = avg.Load15
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		snapshot.MemoryUsedBytes = int64(vm.Used)
		snapshot.MemoryTotalBytes = int64(vm.Total)
		snapshot.MemoryAvailableBytes = int64(vm.Available)
		snapshot.MemoryUsagePercent = vm.UsedPercent
	}
	if du, err := disk.UsageWithContext(ctx, s.cfg.DeploymentsDir); err == nil {
		snapshot.DiskUsedBytes = int64(du.Used)
		snapshot.DiskTotalBytes = int64(du.Total)
		snapshot.DiskAvailableBytes = int64(du.Free)
		snapshot.DiskUsagePercent = du.UsedPercent
	}

	return snapshot
}

func (s *MetricsService) getDiskUsage(snapshot models.SystemMetricSnapshot) dto.DiskUsageResponse {
	deploymentsSize := dirSize(s.cfg.DeploymentsDir)
	logsSize := dirSize(s.cfg.LogsDir)
	dbSize := fileGroupSize(s.cfg.DatabasePath)
	backupsSize := dirSize(s.cfg.BackupDir)
	cacheSize := dirSize(s.cfg.CacheDir)
	platformBytes := deploymentsSize + logsSize + dbSize + backupsSize + cacheSize

	return dto.DiskUsageResponse{
		DeploymentsBytes: deploymentsSize,
		DeploymentBytes:  deploymentsSize,
		LogsBytes:        logsSize,
		DatabaseBytes:    dbSize,
		BackupsBytes:     backupsSize,
		CacheBytes:       cacheSize,
		PlatformBytes:    platformBytes,
		TotalBytes:       snapshot.DiskTotalBytes,
		UsedBytes:        snapshot.DiskUsedBytes,
		AvailableBytes:   snapshot.DiskAvailableBytes,
		UsagePercent:     round1(snapshot.DiskUsagePercent),
	}
}

func (s *MetricsService) buildTrends(ctx context.Context, now time.Time, current models.SystemMetricSnapshot) (dto.MetricTrendsResponse, error) {
	snapshots, err := s.store.ListSince(ctx, now.Add(-trendWindow), 0)
	if err != nil {
		return dto.MetricTrendsResponse{}, err
	}
	if len(snapshots) == 0 || snapshots[len(snapshots)-1].CreatedAt.Before(current.CreatedAt.Add(-time.Minute)) {
		snapshots = append(snapshots, current)
	}

	trends := dto.MetricTrendsResponse{
		CPUUsage:     make([]dto.MetricPointResponse, 0, len(snapshots)),
		MemoryUsage:  make([]dto.MetricPointResponse, 0, len(snapshots)),
		DiskUsage:    make([]dto.MetricPointResponse, 0, len(snapshots)),
		QueuedBuilds: make([]dto.MetricPointResponse, 0, len(snapshots)),
	}

	for _, snapshot := range snapshots {
		ts := snapshot.CreatedAt.UTC().Format(time.RFC3339)
		trends.CPUUsage = append(trends.CPUUsage, dto.MetricPointResponse{Timestamp: ts, Value: round1(snapshot.CPUUsagePercent)})
		trends.MemoryUsage = append(trends.MemoryUsage, dto.MetricPointResponse{Timestamp: ts, Value: round1(snapshot.MemoryUsagePercent)})
		trends.DiskUsage = append(trends.DiskUsage, dto.MetricPointResponse{Timestamp: ts, Value: round1(snapshot.DiskUsagePercent)})
		trends.QueuedBuilds = append(trends.QueuedBuilds, dto.MetricPointResponse{Timestamp: ts, Value: float64(snapshot.QueuedBuilds)})
	}

	return trends, nil
}

func deriveAlerts(stats *dto.AdminStatsResponse) []dto.SystemAlertResponse {
	var alerts []dto.SystemAlertResponse

	if stats.Components.Database.Status != "healthy" {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "error",
			Title:    "Database degraded",
			Message:  defaultMessage(stats.Components.Database.Message, "Database connectivity is degraded."),
		})
	}
	if stats.Components.Docker.Status == "degraded" {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "error",
			Title:    "Docker unavailable",
			Message:  defaultMessage(stats.Components.Docker.Message, "Builds cannot run until Docker is reachable."),
		})
	}
	if stats.Components.Caddy.Status == "degraded" {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "error",
			Title:    "Caddy degraded",
			Message:  defaultMessage(stats.Components.Caddy.Message, "Deployment routing may be out of sync."),
		})
	}
	if stats.DiskUsage.UsagePercent >= 95 {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "error",
			Title:    "Disk critically full",
			Message:  "Filesystem usage is above 95%. Deployments and builds may fail soon.",
		})
	} else if stats.DiskUsage.UsagePercent >= 85 {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "warning",
			Title:    "Disk usage is high",
			Message:  "Filesystem usage is above 85%. Consider cleaning old artifacts or logs.",
		})
	}
	if stats.Memory.UsagePercent >= 90 {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "warning",
			Title:    "Memory pressure is high",
			Message:  "Memory usage is above 90%. Builds may become unstable under load.",
		})
	}
	if stats.CPU.UsagePercent >= 90 {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "warning",
			Title:    "CPU usage is high",
			Message:  "CPU utilization is above 90%. Expect slower build and API response times.",
		})
	}
	if stats.BuildQueue.QueuedBuilds > 0 && stats.BuildQueue.Saturated {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "warning",
			Title:    "Build queue is backlogged",
			Message:  "Queued builds are waiting for capacity. Increase concurrency or reduce load.",
		})
	}
	if stats.DeploymentHealth.Total >= 5 && stats.DeploymentHealth.SuccessRate < 80 {
		alerts = append(alerts, dto.SystemAlertResponse{
			Severity: "warning",
			Title:    "Recent deployment success rate is low",
			Message:  "More than 20% of deployments failed or were cancelled in the last 24 hours.",
		})
	}

	return alerts
}

func defaultMessage(message, fallback string) string {
	if message == "" {
		return fallback
	}
	return message
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func cpuCount() int {
	count, err := cpu.Counts(false)
	if err != nil || count <= 0 {
		return 0
	}
	return count
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func fileGroupSize(path string) int64 {
	return fileSize(path) + fileSize(path+"-wal") + fileSize(path+"-shm")
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
