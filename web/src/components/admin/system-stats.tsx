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
} from "lucide-react";

import { MetricSparkline } from "@/components/admin/metric-sparkline";
import { SystemAlerts } from "@/components/admin/system-alerts";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
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
      <div className="grid gap-4 xl:grid-cols-2">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-48 rounded-xl" />
        ))}
      </div>
    );
  }

  if (!stats) return null;

  return (
    <div className="space-y-6">
      <SystemAlerts alerts={stats.alerts} />

      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="CPU Usage"
          value={formatPercent(stats.cpu.usage_percent)}
          description={`${stats.cpu.cores || "—"} cores · load ${stats.cpu.load1.toFixed(2)}`}
          icon={Gauge}
          progress={stats.cpu.usage_percent}
        />
        <MetricCard
          title="Memory Usage"
          value={`${formatBytes(stats.memory.used_bytes)} / ${formatBytes(stats.memory.total_bytes)}`}
          description={`${formatPercent(stats.memory.usage_percent)} used`}
          icon={MemoryStick}
          progress={stats.memory.usage_percent}
        />
        <MetricCard
          title="Disk Usage"
          value={`${formatBytes(stats.disk_usage.used_bytes)} / ${formatBytes(stats.disk_usage.total_bytes)}`}
          description={`${formatBytes(stats.disk_usage.platform_bytes)} Hostbox footprint`}
          icon={HardDrive}
          progress={stats.disk_usage.usage_percent}
        />
        <MetricCard
          title="Build Queue"
          value={`${stats.build_queue.active_builds} active / ${stats.build_queue.queued_builds} queued`}
          description={`${formatPercent(stats.build_queue.utilization_percent)} of ${stats.build_queue.max_concurrent_builds || 0} slots`}
          icon={Workflow}
          progress={stats.build_queue.utilization_percent}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Component Health</CardTitle>
            <CardDescription>Live readiness of the platform dependencies that matter most.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2">
            <HealthRow label="API" health={stats.components.api} />
            <HealthRow label="Database" health={stats.components.database} />
            <HealthRow label="Docker" health={stats.components.docker} />
            <HealthRow label="Caddy" health={stats.components.caddy} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Platform Activity</CardTitle>
            <CardDescription>Core capacity and deployment reliability over the last {stats.deployment_health.window_hours} hours.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <KeyValue label="Projects" value={stats.project_count} icon={Layers3} />
            <KeyValue label="Deployments" value={stats.deployment_count} icon={Activity} />
            <KeyValue
              label="Success Rate"
              value={formatPercent(stats.deployment_health.success_rate)}
              icon={ServerCog}
            />
            <KeyValue
              label="Uptime"
              value={formatUptime(stats.uptime_seconds)}
              icon={Clock}
            />
            <KeyValue
              label="Avg Successful Build"
              value={
                stats.deployment_health.average_build_duration_ms
                  ? formatDuration(stats.deployment_health.average_build_duration_ms)
                  : "—"
              }
              icon={Workflow}
            />
            <KeyValue
              label="Recent Failures"
              value={stats.deployment_health.failed}
              icon={Database}
            />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Resource Trends</CardTitle>
            <CardDescription>Last 24 hours of captured system samples.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
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
              title="Queued Builds"
              value={stats.build_queue.queued_builds}
              points={stats.trends.queued_builds}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Storage Breakdown</CardTitle>
            <CardDescription>Hostbox-owned data on the current filesystem.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <StorageRow
              label="Deployments"
              bytes={stats.disk_usage.deployments_bytes}
              total={stats.disk_usage.platform_bytes}
            />
            <StorageRow
              label="Logs"
              bytes={stats.disk_usage.logs_bytes}
              total={stats.disk_usage.platform_bytes}
            />
            <StorageRow
              label="Database"
              bytes={stats.disk_usage.database_bytes}
              total={stats.disk_usage.platform_bytes}
            />
            <StorageRow
              label="Backups"
              bytes={stats.disk_usage.backups_bytes}
              total={stats.disk_usage.platform_bytes}
            />
            <StorageRow
              label="Cache"
              bytes={stats.disk_usage.cache_bytes}
              total={stats.disk_usage.platform_bytes}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  description,
  icon: Icon,
  progress,
}: {
  title: string;
  value: string | number;
  description: string;
  icon: React.ElementType;
  progress: number;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
        <div className="space-y-1">
          <CardDescription>{title}</CardDescription>
          <CardTitle className="text-xl">{value}</CardTitle>
        </div>
        <div className="rounded-md bg-muted p-2">
          <Icon className="h-4 w-4 text-muted-foreground" />
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <Progress value={progress} className="h-2" />
        <p className="text-xs text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  );
}

function HealthRow({
  label,
  health,
}: {
  label: string;
  health: ServiceHealth;
}) {
  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-medium">{label}</p>
        <HealthBadge health={health} />
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        {health.message || "Healthy"}
      </p>
    </div>
  );
}

function HealthBadge({ health }: { health: ServiceHealth }) {
  const className =
    health.status === "healthy"
      ? "border-green-500/30 bg-green-500/10 text-green-700 dark:text-green-400"
      : health.status === "degraded"
        ? "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400"
        : "border-zinc-500/30 bg-zinc-500/10 text-zinc-700 dark:text-zinc-400";

  return (
    <Badge variant="outline" className={className}>
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
    <div className="rounded-lg border p-4">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <p className="text-sm font-medium">{title}</p>
          <p className="text-xs text-muted-foreground">Current: {value}</p>
        </div>
        <p className="text-xs text-muted-foreground">{points.length} samples</p>
      </div>
      <MetricSparkline points={points} />
    </div>
  );
}

function StorageRow({
  label,
  bytes,
  total,
}: {
  label: string;
  bytes: number;
  total: number;
}) {
  const ratio = total > 0 ? (bytes / total) * 100 : 0;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span>{label}</span>
        <span className="text-muted-foreground">{formatBytes(bytes)}</span>
      </div>
      <Progress value={ratio} className="h-2" />
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
    <div className="rounded-lg border p-4">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Icon className="h-4 w-4" />
        <span className="text-sm">{label}</span>
      </div>
      <p className="mt-3 text-xl font-semibold">{value}</p>
    </div>
  );
}
