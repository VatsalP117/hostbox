import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { routes, queryKeys } from "@/lib/constants";
import { formatBytes, formatPercent } from "@/lib/utils";
import { timeAgo } from "@/lib/date";
import { StatusBadge } from "@/components/shared/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import type { AdminStatsResponse, DeploymentListResponse } from "@/types/api";
import {
  AlertCircle,
  CheckCircle2,
  Cpu,
  HardDrive,
  MemoryStick,
  XCircle,
  Activity,
  Clock,
  ArrowRight,
  Globe,
  Loader2,
  LayoutGrid,
  Layers,
} from "lucide-react";

export function DashboardPage() {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const isAdmin = !!user?.is_admin;

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: queryKeys.adminStats,
    queryFn: () => api.get<AdminStatsResponse>("/admin/stats"),
    enabled: isAdmin,
  });

  const { data: recentDeploys, isLoading: deploysLoading } = useQuery({
    queryKey: ["recent-deployments"],
    queryFn: () =>
      api.get<DeploymentListResponse>("/admin/deployments", {
        per_page: 5,
      }),
    enabled: isAdmin,
  });

  // Check if there are any error alerts
  const hasAlerts = stats?.alerts && stats.alerts.length > 0;
  const errorAlerts = stats?.alerts?.filter((a) => a.severity === "error") || [];
  const systemState = errorAlerts.length > 0 ? "Degraded" : hasAlerts ? "Warning" : "Healthy";

  return (
    <div className="space-y-10">
      {/* Alert Strip - Conditional */}
      {isAdmin && hasAlerts && (
        <div className="bg-destructive/10 border-b border-destructive/20 px-6 py-3 flex items-center justify-between -mx-6 -mt-6">
          <div className="flex items-center gap-3">
            <div className="w-2 h-2 rounded-full bg-destructive shadow-[0_0_12px_0_rgba(239,68,68,0.4)]" />
            <span className="font-sans text-sm text-destructive font-medium">
              {errorAlerts.length > 0
                ? errorAlerts[0].message
                : stats?.alerts?.[0]?.message}
            </span>
          </div>
          <button
            onClick={() => navigate(routes.adminTab("overview"))}
            className="text-destructive/70 hover:text-destructive text-sm font-sans transition-colors"
          >
            View Details
          </button>
        </div>
      )}

      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <h1 className="font-sans font-extrabold text-4xl tracking-tight text-foreground mb-2">
            Command Center
          </h1>
          <p className="font-sans text-muted-foreground text-sm">
            {isAdmin ? "System Admin View" : "Deployment Overview"} • Last updated: Just now
          </p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => navigate(routes.adminTab("activity"))}
            className="px-4 py-2 rounded-lg border border-border/50 text-foreground font-sans text-sm hover:bg-accent transition-colors"
          >
            Review Activity
          </button>
        </div>
      </div>

      {/* Attention & Metrics Grid (Asymmetric Bento) */}
      {isAdmin && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Attention Cards - Left Column (spans 4) */}
          <div className="lg:col-span-4 flex flex-col gap-4">
            {/* Failed Deployments Card */}
            <div className="bg-card rounded-xl p-5 border border-destructive/20 flex flex-col justify-between h-full relative overflow-hidden group">
              <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                <AlertCircle className="w-16 h-16 text-destructive" />
              </div>
              <div>
                <div className="flex items-center gap-2 mb-4">
                  <AlertCircle className="w-5 h-5 text-destructive" />
                  <span className="font-sans text-xs uppercase tracking-widest text-destructive">
                    Attention Required
                  </span>
                </div>
                {statsLoading ? (
                  <Skeleton className="h-8 w-20 mb-1" />
                ) : (
                  <div className="font-sans font-bold text-3xl text-foreground mb-1">
                    {stats?.deployment_health?.failed || 0} Failed
                  </div>
                )}
                <div className="font-sans text-muted-foreground text-sm">
                  Deployments in the last 24h
                </div>
              </div>
              <div className="mt-6 pt-4 border-t border-border/30">
                <button
                  onClick={() => navigate(routes.admin)}
                  className="font-sans text-sm text-primary hover:text-primary/80 transition-colors flex items-center gap-1"
                >
                  Investigate{" "}
                  <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Alerts Card */}
            <div className="bg-card rounded-xl p-5 border border-warning/20 flex flex-col justify-between h-full relative overflow-hidden group">
              <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                <Globe className="w-16 h-16 text-warning" />
              </div>
              <div>
                <div className="flex items-center gap-2 mb-4">
                  <Clock className="w-5 h-5 text-warning" />
                  <span className="font-sans text-xs uppercase tracking-widest text-warning">
                    Action Needed
                  </span>
                </div>
                {statsLoading ? (
                  <Skeleton className="h-8 w-24 mb-1" />
                ) : (
                  <div className="font-sans font-bold text-3xl text-foreground mb-1">
                    {stats?.alerts?.length || 0} Open
                  </div>
                )}
                <div className="font-sans text-muted-foreground text-sm">
                  {hasAlerts
                    ? stats?.alerts?.[0]?.title || "System attention needed"
                    : "No active system alerts"}
                </div>
              </div>
              <div className="mt-6 pt-4 border-t border-border/30">
                <button
                  onClick={() => navigate(routes.adminTab("overview"))}
                  className="font-sans text-sm text-warning hover:text-warning/80 transition-colors flex items-center gap-1"
                >
                  Open Monitoring{" "}
                  <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          {/* System Health Summary - Right Column (spans 8) */}
          <div className="lg:col-span-8">
            <div className="bg-card/50 border border-border/30 rounded-xl p-6 h-full">
              <div className="flex items-center justify-between mb-6">
                <span className="font-sans text-xs uppercase tracking-widest text-muted-foreground">
                  Global System Health
                </span>
                <div className="flex items-center gap-2">
                  <div
                    className={`w-2 h-2 rounded-full ${
                      systemState === "Healthy"
                        ? "bg-primary shadow-[0_0_12px_0_rgba(255,255,255,0.3)]"
                        : systemState === "Warning"
                          ? "bg-warning"
                          : "bg-destructive"
                    }`}
                  />
                  <span
                    className={`font-sans text-xs ${
                      systemState === "Healthy"
                        ? "text-primary"
                        : systemState === "Warning"
                          ? "text-warning"
                          : "text-destructive"
                    }`}
                  >
                    {systemState}
                  </span>
                </div>
              </div>

              {/* System Stats Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
                {/* CPU Usage */}
                <div className="bg-card rounded-xl p-5 relative overflow-hidden group">
                  <div className="absolute top-3 right-3 opacity-20 group-hover:opacity-30 transition-opacity">
                    <Cpu className="w-8 h-8 text-primary" />
                  </div>
                  <div className="flex justify-between items-start mb-4">
                    <span className="font-mono text-sm text-muted-foreground">
                      CPU Usage
                    </span>
                    <span className="font-mono text-sm text-foreground">
                      {statsLoading
                        ? "—"
                        : formatPercent(stats?.cpu?.usage_percent || 0)}
                    </span>
                  </div>
                  <div className="w-full bg-muted h-1.5 rounded-full overflow-hidden">
                    <div
                      className="bg-primary h-full rounded-full transition-all"
                      style={{
                        width: `${stats?.cpu?.usage_percent || 0}%`,
                      }}
                    />
                  </div>
                  <div className="mt-3 font-sans text-xs text-muted-foreground">
                    {stats ? `${stats.cpu.cores} cores` : "—"}
                  </div>
                </div>

                {/* Memory Usage */}
                <div className="bg-card rounded-xl p-5 relative overflow-hidden group">
                  <div className="absolute top-3 right-3 opacity-20 group-hover:opacity-30 transition-opacity">
                    <MemoryStick className="w-8 h-8 text-warning" />
                  </div>
                  <div className="flex justify-between items-start mb-4">
                    <span className="font-mono text-sm text-muted-foreground">
                      Memory (RAM)
                    </span>
                    <span className="font-mono text-sm text-foreground">
                      {statsLoading
                        ? "—"
                        : formatPercent(stats?.memory?.usage_percent || 0)}
                    </span>
                  </div>
                  <div className="w-full bg-muted h-1.5 rounded-full overflow-hidden">
                    <div
                      className="bg-warning h-full rounded-full transition-all"
                      style={{
                        width: `${stats?.memory?.usage_percent || 0}%`,
                      }}
                    />
                  </div>
                  <div className="mt-3 font-sans text-xs text-muted-foreground">
                    {stats
                      ? `${formatBytes(stats.memory.used_bytes)} / ${formatBytes(stats.memory.total_bytes)}`
                      : "—"}
                  </div>
                </div>

                {/* Disk Usage */}
                <div className="bg-card rounded-xl p-5 relative overflow-hidden group">
                  <div className="absolute top-3 right-3 opacity-20 group-hover:opacity-30 transition-opacity">
                    <HardDrive className="w-8 h-8 text-primary" />
                  </div>
                  <div className="flex justify-between items-start mb-4">
                    <span className="font-mono text-sm text-muted-foreground">
                      Disk Usage
                    </span>
                    <span className="font-mono text-sm text-foreground">
                      {statsLoading
                        ? "—"
                        : formatPercent(stats?.disk_usage?.usage_percent || 0)}
                    </span>
                  </div>
                  <div className="w-full bg-muted h-1.5 rounded-full overflow-hidden">
                    <div
                      className="bg-primary h-full rounded-full transition-all"
                      style={{
                        width: `${stats?.disk_usage?.usage_percent || 0}%`,
                      }}
                    />
                  </div>
                  <div className="mt-3 font-sans text-xs text-muted-foreground">
                    {stats
                      ? `${formatBytes(stats.disk_usage.used_bytes)} / ${formatBytes(stats.disk_usage.total_bytes)}`
                      : "—"}
                  </div>
                </div>

                {/* Build Queue */}
                <div className="bg-card rounded-xl p-5 relative overflow-hidden group">
                  <div className="absolute top-3 right-3 opacity-20 group-hover:opacity-30 transition-opacity">
                    <Layers className="w-8 h-8 text-secondary" />
                  </div>
                  <div className="flex justify-between items-start mb-4">
                    <span className="font-mono text-sm text-muted-foreground">
                      Build Queue
                    </span>
                    <span className="font-mono text-sm text-foreground">
                      {statsLoading
                        ? "—"
                        : `${stats?.build_queue?.active_builds || 0} active`}
                    </span>
                  </div>
                  <div className="w-full bg-muted h-1.5 rounded-full overflow-hidden">
                    <div
                      className="bg-secondary h-full rounded-full transition-all"
                      style={{
                        width: `${stats?.build_queue?.utilization_percent || 0}%`,
                      }}
                    />
                  </div>
                  <div className="mt-3 font-sans text-xs text-muted-foreground">
                    {stats
                      ? `${stats.build_queue.queued_builds} queued • ${formatPercent(stats.build_queue.utilization_percent)} utilized`
                      : "—"}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Lower Section: Activity & Queue */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Recent Deployments - Spans 2 */}
        <div className="lg:col-span-2">
          <h2 className="font-sans font-bold text-xl text-foreground mb-6 flex items-center gap-2">
            <LayoutGrid className="w-5 h-5 text-muted-foreground" />
            Recent Deployments
          </h2>
          <div className="bg-card rounded-xl border border-border/30 overflow-hidden">
            {deploysLoading ? (
              <div className="p-4 space-y-3">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : !recentDeploys?.deployments?.length ? (
              <div className="p-8 text-center">
                <p className="text-sm text-muted-foreground">
                  No deployments yet. Create a project to get started.
                </p>
              </div>
            ) : (
              <>
                {recentDeploys.deployments.map((d, index) => (
                  <div
                    key={d.id}
                    onClick={() =>
                      navigate(routes.deployment(d.project_id, d.id))
                    }
                    className={`p-4 flex items-center justify-between group cursor-pointer hover:bg-accent/50 transition-colors ${
                      index !== recentDeploys.deployments.length - 1
                        ? "border-b border-border/30"
                        : ""
                    } ${d.status === "failed" ? "bg-destructive/5" : ""}`}
                  >
                    <div className="flex items-center gap-4">
                      <div
                        className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                          d.status === "failed"
                            ? "bg-destructive/10 text-destructive"
                            : d.status === "ready"
                            ? "bg-primary/10 text-primary"
                            : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {d.status === "failed" ? (
                          <XCircle className="w-5 h-5" />
                        ) : d.status === "ready" ? (
                          <CheckCircle2 className="w-5 h-5" />
                        ) : (
                          <Loader2 className="w-5 h-5 animate-spin" />
                        )}
                      </div>
                      <div>
                        <div className="font-sans font-medium text-foreground text-sm">
                          {d.commit_message || d.branch}
                        </div>
                        <div className="font-mono text-xs text-muted-foreground mt-1">
                          commit{" "}
                          <span className="text-foreground">
                            {d.commit_sha.slice(0, 7)}
                          </span>{" "}
                          • {timeAgo(d.created_at)}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <StatusBadge status={d.status} />
                      <span className="font-sans text-xs text-muted-foreground group-hover:text-primary transition-colors">
                        {d.branch}
                      </span>
                    </div>
                  </div>
                ))}
              </>
            )}
          </div>
        </div>

        {/* Build Queue - Spans 1 */}
        <div className="lg:col-span-1">
          <h2 className="font-sans font-bold text-xl text-foreground mb-6 flex items-center gap-2">
            <Activity className="w-5 h-5 text-muted-foreground" />
            Build Queue
          </h2>
          <div className="bg-card/50 border border-border/30 rounded-xl p-5 relative overflow-hidden">
            {/* Scanning line effect */}
            <div className="absolute top-0 left-0 w-full h-1 bg-primary/20 animate-pulse" />

            {statsLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-16 w-full" />
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between mb-4">
                  <span className="font-sans text-xs uppercase tracking-widest text-muted-foreground">
                    Build Capacity
                  </span>
                  <span className="font-mono text-xs text-primary">
                    {stats?.build_queue?.active_builds
                      ? `${stats.build_queue.active_builds}/${stats.build_queue.max_concurrent_builds} active`
                      : "Idle"}
                  </span>
                </div>

                <div className="space-y-4">
                  {/* Active Build */}
                  {stats && stats.build_queue && stats.build_queue.active_builds > 0 && (
                    <div className="bg-card p-4 rounded-lg border border-primary/30 relative">
                      <div className="absolute -left-1 top-1/2 -translate-y-1/2 w-2 h-8 bg-primary rounded-r-md" />
                      <div className="font-sans text-sm text-foreground font-medium mb-1">
                        {stats.build_queue.active_builds} build{stats.build_queue.active_builds === 1 ? "" : "s"} in progress
                      </div>
                      <div className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
                        <Loader2 className="w-4 h-4 text-primary animate-spin" />
                        Queue utilization {formatPercent(stats.build_queue.utilization_percent)}
                      </div>
                    </div>
                  )}

                  {/* Queue Status */}
                  <div
                    className={`p-3 border border-border/30 rounded-lg ${
                      stats && stats.build_queue && stats.build_queue.queued_builds > 0
                        ? "border-dashed"
                        : ""
                    }`}
                  >
                    <div className="font-sans text-sm text-foreground font-medium mb-1">
                      Queue Status
                    </div>
                    <div className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
                      {stats && stats.build_queue && stats.build_queue.queued_builds > 0 ? (
                        <>
                          <Clock className="w-4 h-4" />
                          {stats.build_queue.queued_builds} builds queued
                        </>
                      ) : (
                        <>
                          <CheckCircle2 className="w-4 h-4 text-success" />
                          No builds queued
                        </>
                      )}
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
