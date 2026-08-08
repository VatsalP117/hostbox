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
  const statusColor = hasErrors ? "bg-destructive" : hasAlerts ? "bg-warning" : "bg-success";

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <header className="flex flex-col justify-between gap-4 md:flex-row md:items-center">
        <div>
          <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
            <div className={`h-1.5 w-1.5 rounded-full ${statusColor}`} />
            System {systemStatus.toLowerCase()}
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">Administration</h1>
          <p className="mt-1 max-w-xl text-sm text-muted-foreground">
            VM health, users, activity, and platform configuration.
            {stats?.uptime_seconds && (
              <span className="ml-1">Uptime {formatUptime(stats.uptime_seconds)}.</span>
            )}
          </p>
        </div>
        <div className="flex space-x-3">
          <Button
            variant="outline"
            onClick={handleRefresh}
            className="h-9 bg-black"
          >
            <RefreshCw className="h-4 w-4" />
            <span>Refresh</span>
          </Button>
        </div>
      </header>

      {/* System Status Bar - Only show if there are alerts */}
      {hasAlerts && stats?.alerts && (
        <SystemAlerts alerts={stats.alerts} />
      )}

      {/* Tabs Navigation */}
      <Tabs value={currentTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList className="h-auto w-full justify-start gap-6 overflow-x-auto rounded-none border-b border-border bg-transparent p-0">
          <TabsTrigger
            value="overview"
            className="rounded-none border-b-2 border-transparent px-0 py-3 text-sm text-muted-foreground data-[state=active]:border-white data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
          >
            <LayoutDashboard className="h-4 w-4 mr-2" />
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="users"
            className="rounded-none border-b-2 border-transparent px-0 py-3 text-sm text-muted-foreground data-[state=active]:border-white data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
          >
            <Users className="h-4 w-4 mr-2" />
            Users
          </TabsTrigger>
          <TabsTrigger
            value="activity"
            className="rounded-none border-b-2 border-transparent px-0 py-3 text-sm text-muted-foreground data-[state=active]:border-white data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
          >
            <Activity className="h-4 w-4 mr-2" />
            Activity
          </TabsTrigger>
          <TabsTrigger
            value="settings"
            className="rounded-none border-b-2 border-transparent px-0 py-3 text-sm text-muted-foreground data-[state=active]:border-white data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none"
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
