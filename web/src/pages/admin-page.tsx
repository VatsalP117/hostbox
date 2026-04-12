import { PageHeader } from "@/components/shared/page-header";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { SystemStatsGrid } from "@/components/admin/system-stats";
import { UserTable } from "@/components/admin/user-table";
import { ActivityLog } from "@/components/admin/activity-log";
import { AdminSettingsForm } from "@/components/admin/admin-settings-form";
import { useAdminStats, useAdminUsers } from "@/hooks/use-admin";

export function AdminPage() {
  const { data: stats, isLoading: statsLoading } = useAdminStats();
  const { data: users, isLoading: usersLoading } = useAdminUsers();

  return (
    <div className="space-y-6">
      <PageHeader
        title="Admin"
        description="Platform administration and monitoring."
      />

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6">
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
