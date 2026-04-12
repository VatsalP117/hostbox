import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { routes, queryKeys } from "@/lib/constants";
import { formatBytes, formatUptime } from "@/lib/utils";
import { timeAgo } from "@/lib/date";
import { PageHeader } from "@/components/shared/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import type { AdminStatsResponse, DeploymentListResponse } from "@/types/api";
import {
  FolderKanban,
  Rocket,
  HardDrive,
  Hammer,
  Plus,
} from "lucide-react";

export function DashboardPage() {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: queryKeys.adminStats,
    queryFn: () => api.get<AdminStatsResponse>("/admin/stats"),
  });

  const { data: recentDeploys, isLoading: deploysLoading } = useQuery({
    queryKey: ["recent-deployments"],
    queryFn: () =>
      api.get<DeploymentListResponse>("/admin/deployments", {
        per_page: 5,
      }),
  });

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Welcome back, ${user?.display_name || "User"}`}
        description="Here's an overview of your platform."
      >
        <Button onClick={() => navigate(routes.newProject)}>
          <Plus className="mr-2 h-4 w-4" />
          New Project
        </Button>
      </PageHeader>

      {/* Stats row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatsCard
          title="Total Projects"
          value={stats?.project_count}
          icon={FolderKanban}
          loading={statsLoading}
        />
        <StatsCard
          title="Total Deployments"
          value={stats?.deployment_count}
          icon={Rocket}
          loading={statsLoading}
        />
        <StatsCard
          title="Active Builds"
          value={stats?.active_builds}
          icon={Hammer}
          loading={statsLoading}
        />
        <StatsCard
          title="Disk Usage"
          value={
            stats
              ? `${formatBytes(stats.disk_usage.used_bytes)} / ${formatBytes(stats.disk_usage.total_bytes)}`
              : undefined
          }
          icon={HardDrive}
          loading={statsLoading}
        />
      </div>

      {/* Recent Deployments */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Recent Deployments</CardTitle>
        </CardHeader>
        <CardContent>
          {deploysLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : !recentDeploys?.deployments?.length ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No deployments yet. Create a project to get started.
            </p>
          ) : (
            <div className="space-y-2">
              {recentDeploys.deployments.map((d) => (
                <div
                  key={d.id}
                  className="flex items-center justify-between rounded-md border p-3 cursor-pointer hover:bg-accent transition-colors"
                  onClick={() =>
                    navigate(routes.deployment(d.project_id, d.id))
                  }
                >
                  <div className="flex items-center gap-3">
                    <StatusBadge status={d.status} />
                    <div>
                      <p className="text-sm font-medium">
                        {d.commit_message || d.branch}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {d.branch} · {d.commit_sha.slice(0, 7)}
                      </p>
                    </div>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {timeAgo(d.created_at)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function StatsCard({
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
