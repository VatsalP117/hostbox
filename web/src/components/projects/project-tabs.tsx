import { useSearchParams } from "react-router-dom";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { DeploymentsTab } from "@/pages/deployments-tab";
import { ProjectSettingsTab } from "@/pages/project-settings-tab";
import { DomainsTab } from "@/pages/domains-tab";
import { EnvVarsTab } from "@/pages/env-vars-tab";
import { NotificationsTab } from "@/pages/notifications-tab";
import { OverviewTab } from "@/pages/overview-tab";
import type { Project } from "@/types/models";
import { cn } from "@/lib/utils";

const tabs = [
  { value: "overview", label: "Overview" },
  { value: "deployments", label: "Deployments" },
  { value: "domains", label: "Domains" },
  { value: "environment", label: "Environment" },
  { value: "notifications", label: "Notifications" },
  { value: "settings", label: "Settings" },
] as const;

interface ProjectTabsProps {
  project: Project;
}

export function ProjectTabs({ project }: ProjectTabsProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const currentTab = searchParams.get("tab") || "overview";

  const handleTabChange = (value: string) => {
    setSearchParams({ tab: value }, { replace: true });
  };

  return (
    <Tabs value={currentTab} onValueChange={handleTabChange} className="w-full">
      <TabsList className="w-full justify-start bg-surface-container-low rounded-xl p-1 h-auto gap-1">
        {tabs.map((tab) => (
          <TabsTrigger
            key={tab.value}
            value={tab.value}
            className={cn(
              "rounded-lg px-4 py-2 font-label text-sm font-medium transition-all",
              "data-[state=inactive]:text-on-surface-variant data-[state=inactive]:hover:text-on-surface",
              "data-[state=active]:bg-surface-container data-[state=active]:text-on-surface data-[state=active]:shadow-sm"
            )}
          >
            {tab.label}
          </TabsTrigger>
        ))}
      </TabsList>

      <div className="mt-8">
        <TabsContent value="overview" className="mt-0">
          <OverviewTab project={project} />
        </TabsContent>
        <TabsContent value="deployments" className="mt-0">
          <DeploymentsTab projectId={project.id} productionBranch={project.production_branch} />
        </TabsContent>
        <TabsContent value="settings" className="mt-0">
          <ProjectSettingsTab project={project} />
        </TabsContent>
        <TabsContent value="domains" className="mt-0">
          <DomainsTab projectId={project.id} />
        </TabsContent>
        <TabsContent value="environment" className="mt-0">
          <EnvVarsTab projectId={project.id} />
        </TabsContent>
        <TabsContent value="notifications" className="mt-0">
          <NotificationsTab projectId={project.id} />
        </TabsContent>
      </div>
    </Tabs>
  );
}
