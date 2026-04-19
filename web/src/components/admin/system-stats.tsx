import {
  Activity,
  Clock,
  Database,
  Gauge,
  HardDrive,
  Layers3,
  MemoryStick,
  ServerCog,
  Workflow,
  ArrowUpRight,
  ArrowDownRight,
} from "lucide-react";

import { MetricSparkline } from "@/components/admin/metric-sparkline";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { formatBytes, formatDuration, formatPercent, formatUptime } from "@/lib/utils";
import type { ServiceHealth, SystemStats } from "@/types/models";

interface SystemStatsProps {
  stats: SystemStats | undefined;
  isLoading: boolean;
}

export function SystemStatsGrid({ stats, isLoading }: SystemStatsProps) {
  if (isLoading) {
    return (
      <div className="space-y-6">
        {/* Bento Grid Skeleton */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-40 rounded-xl bg-surface-container-low" />
          ))}
        </div>
        {/* Main Content Skeleton */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <Skeleton className="h-80 rounded-xl bg-surface-container-low lg:col-span-2" />
          <Skeleton className="h-80 rounded-xl bg-surface-container-low" />
        </div>
      </div>
    );
  }

  if (!stats) return null;

  return (
    <div className="space-y-8">
      {/* Metrics Bento Grid */}
      <section className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* CPU Card */}
        <MetricCard
          title="CPU Load"
          value={Math.round(stats.cpu.usage_percent)}
          unit="%"
          description={`${stats.cpu.cores || "—"} cores · load ${stats.cpu.load1.toFixed(2)}`}
          icon={Gauge}
          progress={stats.cpu.usage_percent}
          color="bg-primary"
          trend={stats.trends.cpu_usage}
        />

        {/* Memory Card */}
        <MetricCard
          title="Memory"
          value={Math.round(stats.memory.usage_percent)}
          unit="%"
          description={`${formatBytes(stats.memory.used_bytes)} / ${formatBytes(stats.memory.total_bytes)}`}
          icon={MemoryStick}
          progress={stats.memory.usage_percent}
          color="bg-warning"
          trend={stats.trends.memory_usage}
        />

        {/* Disk Card */}
        <MetricCard
          title="Disk I/O"
          value={stats.disk_usage.usage_percent.toFixed(1)}
          unit="%"
          description={`${formatBytes(stats.disk_usage.platform_bytes)} Hostbox data`}
          icon={HardDrive}
          progress={stats.disk_usage.usage_percent}
          color="bg-primary"
          trend={stats.trends.disk_usage}
        />

        {/* Build Queue Card */}
        <MetricCard
          title="Queue Saturation"
          value={stats.build_queue.active_builds}
          unit="jobs"
          description={`${stats.build_queue.queued_builds} queued / ${stats.build_queue.max_concurrent_builds || 0} slots`}
          icon={Workflow}
          progress={stats.build_queue.utilization_percent}
          color={stats.build_queue.utilization_percent > 80 ? "bg-destructive" : "bg-primary"}
          segments
        />
      </section>

      {/* Complex Layout Area */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content (2/3) */}
        <section className="lg:col-span-2 space-y-6">
          {/* Component Health & Platform Activity */}
          <div className="bg-surface-container-low rounded-xl p-6 relative overflow-hidden">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h3 className="font-headline text-xl font-bold text-foreground">Platform Health</h3>
                <p className="font-label text-xs text-muted-foreground mt-1 uppercase tracking-wider">
                  Component Status
                </p>
              </div>
              <div className="font-headline text-2xl font-bold text-primary">
                {formatPercent(stats.deployment_health.success_rate)}
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <HealthRow label="API" health={stats.components.api} />
              <HealthRow label="Database" health={stats.components.database} />
              <HealthRow label="Docker" health={stats.components.docker} />
              <HealthRow label="Caddy" health={stats.components.caddy} />
            </div>

            <div className="mt-6 pt-6 border-t border-outline-variant/15">
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                <KeyValue label="Projects" value={stats.project_count} icon={Layers3} />
                <KeyValue label="Deployments" value={stats.deployment_count} icon={Activity} />
                <KeyValue label="Uptime" value={formatUptime(stats.uptime_seconds)} icon={Clock} />
                <KeyValue
                  label="Avg Build"
                  value={
                    stats.deployment_health.average_build_duration_ms
                      ? formatDuration(stats.deployment_health.average_build_duration_ms)
                      : "—"
                  }
                  icon={Workflow}
                />
              </div>
            </div>
          </div>

          {/* Resource Trends */}
          <div className="bg-surface-container-low rounded-xl p-6">
            <div className="flex items-center justify-between mb-6">
              <h3 className="font-headline text-lg font-bold text-foreground">Resource Trends</h3>
              <p className="font-label text-xs text-muted-foreground">Last 24 hours</p>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              <TrendCard
                title="CPU"
                value={formatPercent(stats.cpu.usage_percent)}
                points={stats.trends.cpu_usage}
              />
              <TrendCard
                title="Memory"
                value={formatPercent(stats.memory.usage_percent)}
                points={stats.trends.memory_usage}
              />
              <TrendCard
                title="Disk"
                value={formatPercent(stats.disk_usage.usage_percent)}
                points={stats.trends.disk_usage}
              />
              <TrendCard
                title="Build Queue"
                value={stats.build_queue.queued_builds}
                points={stats.trends.queued_builds}
              />
            </div>
          </div>
        </section>

        {/* Sidebar (1/3) */}
        <section className="space-y-6">
          {/* Storage Breakdown */}
          <div className="bg-surface-container-low rounded-xl p-6">
            <h3 className="font-headline text-lg font-bold text-foreground mb-6">Storage Breakdown</h3>
            <div className="space-y-4">
              <StorageRow
                label="Deployments"
                bytes={stats.disk_usage.deployments_bytes}
                total={stats.disk_usage.platform_bytes}
                color="bg-primary"
              />
              <StorageRow
                label="Logs"
                bytes={stats.disk_usage.logs_bytes}
                total={stats.disk_usage.platform_bytes}
                color="bg-warning"
              />
              <StorageRow
                label="Database"
                bytes={stats.disk_usage.database_bytes}
                total={stats.disk_usage.platform_bytes}
                color="bg-chart-4"
              />
              <StorageRow
                label="Backups"
                bytes={stats.disk_usage.backups_bytes}
                total={stats.disk_usage.platform_bytes}
                color="bg-chart-2"
              />
              <StorageRow
                label="Cache"
                bytes={stats.disk_usage.cache_bytes}
                total={stats.disk_usage.platform_bytes}
                color="bg-chart-5"
              />
            </div>
            <div className="mt-6 pt-4 border-t border-outline-variant/15">
              <div className="flex items-center justify-between">
                <span className="font-body text-sm text-muted-foreground">Total Platform Data</span>
                <span className="font-headline font-bold text-foreground">
                  {formatBytes(stats.disk_usage.platform_bytes)}
                </span>
              </div>
            </div>
          </div>

          {/* Deployment Stats */}
          <div className="bg-surface-container-low rounded-xl p-6">
            <h3 className="font-headline text-lg font-bold text-foreground mb-4">Deployment Stats</h3>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <ArrowUpRight className="h-4 w-4 text-chart-2" />
                  <span className="font-body text-sm text-muted-foreground">Successful</span>
                </div>
                <span className="font-label font-medium text-foreground">
                  {stats.deployment_health.successful}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <ArrowDownRight className="h-4 w-4 text-destructive" />
                  <span className="font-body text-sm text-muted-foreground">Failed</span>
                </div>
                <span className="font-label font-medium text-foreground">
                  {stats.deployment_health.failed}
                </span>
              </div>
              <div className="h-2 bg-surface-container-high rounded-full overflow-hidden flex">
                <div
                  className="h-full bg-chart-2"
                  style={{
                    width: `${(stats.deployment_health.successful / (stats.deployment_health.successful + stats.deployment_health.failed || 1)) * 100}%`,
                  }}
                />
                <div
                  className="h-full bg-destructive"
                  style={{
                    width: `${(stats.deployment_health.failed / (stats.deployment_health.successful + stats.deployment_health.failed || 1)) * 100}%`,
                  }}
                />
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  unit,
  description,
  icon: Icon,
  progress,
  color = "bg-primary",
  trend,
  segments = false,
}: {
  title: string;
  value: number | string;
  unit: string;
  description: string;
  icon: React.ElementType;
  progress: number;
  color?: string;
  trend?: { timestamp: string; value: number }[];
  segments?: boolean;
}) {
  return (
    <div className="bg-surface-container-low rounded-xl p-5 flex flex-col justify-between relative overflow-hidden group hover:bg-surface-container transition-colors duration-300">
      <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 rounded-full blur-2xl -mr-10 -mt-10 group-hover:bg-primary/10 transition-colors" />
      <div className="flex justify-between items-start mb-4 relative z-10">
        <span className="font-label text-xs uppercase tracking-widest text-muted-foreground">{title}</span>
        <Icon className="h-5 w-5 text-muted-foreground/50" />
      </div>
      <div className="relative z-10">
        <div className="flex items-baseline space-x-1">
          <span className="font-headline text-3xl font-bold tracking-tight text-foreground">{value}</span>
          <span className="font-label text-sm text-muted-foreground">{unit}</span>
        </div>
        <p className="font-body text-xs text-muted-foreground mt-1">{description}</p>
        {segments ? (
          <div className="mt-4 h-1 bg-surface-container-high rounded-full overflow-hidden flex space-x-0.5">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className={`h-full rounded-full flex-1 ${
                  i < Math.ceil((progress / 100) * 3) ? color : "bg-surface-container-high"
                }`}
              />
            ))}
          </div>
        ) : (
          <div className="mt-4 w-full h-1 bg-surface-container-high rounded-full overflow-hidden">
            <div
              className={`h-full ${color} rounded-full transition-all duration-500`}
              style={{ width: `${Math.min(progress, 100)}%` }}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function HealthRow({
  label,
  health,
}: {
  label: string;
  health: ServiceHealth;
}) {
  const statusColors = {
    healthy: "bg-chart-2 shadow-[0_0_12px_rgba(34,197,94,0.3)]",
    degraded: "bg-warning shadow-[0_0_12px_rgba(245,158,11,0.3)]",
    unhealthy: "bg-destructive shadow-[0_0_12px_rgba(239,68,68,0.3)]",
  };

  return (
    <div className="flex items-center justify-between p-3 bg-surface-container rounded-lg">
      <div className="flex items-center space-x-3">
        <div className={`w-2 h-2 rounded-full ${statusColors[health.status as keyof typeof statusColors] || statusColors.healthy}`} />
        <span className="font-body text-sm font-medium text-foreground">{label}</span>
      </div>
      <div className="flex items-center space-x-2">
        <HealthBadge health={health} />
      </div>
    </div>
  );
}

function HealthBadge({ health }: { health: ServiceHealth }) {
  const className =
    health.status === "healthy"
      ? "border-chart-2/30 bg-chart-2/10 text-chart-2"
      : health.status === "degraded"
        ? "bg-warning/10 text-warning border-warning/30"
        : "bg-destructive/10 text-destructive border-destructive/30";

  return (
    <Badge variant="outline" className={`${className} font-label text-xs uppercase tracking-wider`}>
      {health.status}
    </Badge>
  );
}

function TrendCard({
  title,
  value,
  points,
}: {
  title: string;
  value: string | number;
  points: { timestamp: string; value: number }[];
}) {
  return (
    <div className="rounded-lg bg-surface-container p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="font-label text-xs text-muted-foreground uppercase tracking-wider">{title}</span>
        <span className="font-headline text-sm font-bold text-foreground">{value}</span>
      </div>
      <MetricSparkline points={points} />
      <p className="font-label text-[0.65rem] text-muted-foreground mt-2">{points.length} samples</p>
    </div>
  );
}

function StorageRow({
  label,
  bytes,
  total,
  color = "bg-primary",
}: {
  label: string;
  bytes: number;
  total: number;
  color?: string;
}) {
  const ratio = total > 0 ? (bytes / total) * 100 : 0;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <div className={`w-3 h-3 rounded-sm ${color}`} />
          <span className="font-body text-sm text-foreground">{label}</span>
        </div>
        <span className="font-label text-sm text-muted-foreground">{formatBytes(bytes)}</span>
      </div>
      <div className="h-1 bg-surface-container-high rounded-full overflow-hidden">
        <div className={`h-full ${color} rounded-full`} style={{ width: `${ratio}%` }} />
      </div>
    </div>
  );
}

function KeyValue({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: string | number;
  icon: React.ElementType;
}) {
  return (
    <div className="rounded-lg bg-surface-container p-3">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Icon className="h-4 w-4" />
        <span className="font-label text-xs uppercase tracking-wider">{label}</span>
      </div>
      <p className="mt-2 font-headline text-lg font-bold text-foreground">{value}</p>
    </div>
  );
}
