import { useSearchParams } from "react-router-dom";
import { RefreshCw, Users, Activity, Settings, LayoutDashboard } from "lucide-react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { SystemStatsGrid } from "@/components/admin/system-stats";
import { SystemAlerts } from "@/components/admin/system-alerts";
import { UserTable } from "@/components/admin/user-table";
import { ActivityLog } from "@/components/admin/activity-log";
import { AdminSettingsForm } from "@/components/admin/admin-settings-form";
import { useAdminStats, useAdminUsers } from "@/hooks/use-admin";
import { formatUptime } from "@/lib/utils";

export function AdminPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: stats, isLoading: statsLoading, refetch } = useAdminStats();
  const { data: users, isLoading: usersLoading } = useAdminUsers();
  const currentTab = searchParams.get("tab") || "overview";

  const handleRefresh = () => {
    refetch();
  };

  const handleTabChange = (value: string) => {
    setSearchParams({ tab: value }, { replace: true });
  };

  // Determine overall system status
  const hasAlerts = stats?.alerts && stats.alerts.length > 0;
  const hasErrors = stats?.alerts?.some((a) => a.severity === "error");
  const systemStatus = hasErrors ? "Error" : hasAlerts ? "Warning" : "Healthy";
  const statusColor = hasErrors ? "bg-destructive" : hasAlerts ? "bg-warning" : "bg-primary";
  const statusGlow = hasErrors ? "shadow-[0_0_12px_rgba(255,180,171,0.3)]" : "shadow-[0_0_12px_rgba(173,198,255,0.3)]";

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <header className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-2">
          <div className="flex items-center space-x-3 mb-3">
            <div className={`w-2.5 h-2.5 rounded-full ${statusColor} ${statusGlow} animate-pulse`} />
            <span className="font-label text-sm text-primary tracking-wider uppercase">
              Platform Operations
            </span>
          </div>
          <h1 className="font-headline text-4xl md:text-5xl font-extrabold tracking-tighter text-foreground leading-none">
            System {systemStatus}
          </h1>
          <p className="font-body text-muted-foreground max-w-xl mt-4">
            Core infrastructure is operating optimally across all geographic regions.
            {stats?.uptime_seconds && (
              <span className="block mt-1">Uptime: {formatUptime(stats.uptime_seconds)}</span>
            )}
          </p>
        </div>
        <div className="flex space-x-3">
          <Button
            variant="outline"
            onClick={handleRefresh}
            className="rounded-lg border-outline-variant/30 hover:bg-surface-container-high transition-colors flex items-center space-x-2"
          >
            <RefreshCw className="h-4 w-4" />
            <span className="font-label text-sm">Refresh Data</span>
          </Button>
        </div>
      </header>

      {/* System Status Bar - Only show if there are alerts */}
      {hasAlerts && stats?.alerts && (
        <SystemAlerts alerts={stats.alerts} />
      )}

      {/* Tabs Navigation */}
      <Tabs value={currentTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList className="bg-surface-container-low p-1 rounded-xl h-auto">
          <TabsTrigger
            value="overview"
            className="rounded-lg px-4 py-2.5 data-[state=active]:bg-surface-container data-[state=active]:text-foreground font-label text-xs uppercase tracking-widest text-muted-foreground transition-all"
          >
            <LayoutDashboard className="h-4 w-4 mr-2" />
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="users"
            className="rounded-lg px-4 py-2.5 data-[state=active]:bg-surface-container data-[state=active]:text-foreground font-label text-xs uppercase tracking-widest text-muted-foreground transition-all"
          >
            <Users className="h-4 w-4 mr-2" />
            Users
          </TabsTrigger>
          <TabsTrigger
            value="activity"
            className="rounded-lg px-4 py-2.5 data-[state=active]:bg-surface-container data-[state=active]:text-foreground font-label text-xs uppercase tracking-widest text-muted-foreground transition-all"
          >
            <Activity className="h-4 w-4 mr-2" />
            Activity
          </TabsTrigger>
          <TabsTrigger
            value="settings"
            className="rounded-lg px-4 py-2.5 data-[state=active]:bg-surface-container data-[state=active]:text-foreground font-label text-xs uppercase tracking-widest text-muted-foreground transition-all"
          >
            <Settings className="h-4 w-4 mr-2" />
            Settings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6 space-y-6">
          <SystemStatsGrid stats={stats} isLoading={statsLoading} />
        </TabsContent>

        <TabsContent value="users" className="mt-6">
          <UserTable users={users?.users} isLoading={usersLoading} />
        </TabsContent>

        <TabsContent value="activity" className="mt-6">
          <ActivityLog />
        </TabsContent>

        <TabsContent value="settings" className="mt-6">
          <AdminSettingsForm />
        </TabsContent>
      </Tabs>
    </div>
  );
}
