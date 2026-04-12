import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { DiskUsageBar } from "@/components/admin/disk-usage-bar";
import { formatBytes, formatUptime } from "@/lib/utils";
import type { SystemStats } from "@/types/models";
import { FolderKanban, Rocket, Hammer, HardDrive, Clock } from "lucide-react";

interface SystemStatsProps {
  stats: SystemStats | undefined;
  isLoading: boolean;
}

export function SystemStatsGrid({ stats, isLoading }: SystemStatsProps) {
  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Projects"
          value={stats?.project_count}
          icon={FolderKanban}
          loading={isLoading}
        />
        <StatCard
          title="Total Deployments"
          value={stats?.deployment_count}
          icon={Rocket}
          loading={isLoading}
        />
        <StatCard
          title="Active Builds"
          value={stats?.active_builds}
          icon={Hammer}
          loading={isLoading}
        />
        <StatCard
          title="Uptime"
          value={stats ? formatUptime(stats.uptime_seconds) : undefined}
          icon={Clock}
          loading={isLoading}
        />
      </div>

      {stats && (
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <HardDrive className="h-4 w-4 text-muted-foreground" />
              Disk Usage
            </div>
            <DiskUsageBar
              usedBytes={stats.disk_usage.used_bytes}
              totalBytes={stats.disk_usage.total_bytes}
              deploymentBytes={stats.disk_usage.deployment_bytes}
            />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>
                {formatBytes(stats.disk_usage.used_bytes)} used of{" "}
                {formatBytes(stats.disk_usage.total_bytes)}
              </span>
              <span>
                {formatBytes(stats.disk_usage.deployment_bytes)} deployments
              </span>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function StatCard({
  title,
  value,
  icon: Icon,
  loading,
}: {
  title: string;
  value?: string | number;
  icon: React.ElementType;
  loading: boolean;
}) {
  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <div className="rounded-md bg-muted p-2">
          <Icon className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <p className="text-sm text-muted-foreground">{title}</p>
          {loading ? (
            <Skeleton className="mt-1 h-6 w-16" />
          ) : (
            <p className="text-2xl font-bold">{value ?? "—"}</p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
