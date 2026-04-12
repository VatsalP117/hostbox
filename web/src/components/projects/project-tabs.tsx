import { useSearchParams } from "react-router-dom";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { DeploymentsTab } from "@/pages/deployments-tab";
import { ProjectSettingsTab } from "@/pages/project-settings-tab";
import { DomainsTab } from "@/pages/domains-tab";
import { EnvVarsTab } from "@/pages/env-vars-tab";
import { NotificationsTab } from "@/pages/notifications-tab";
import type { Project } from "@/types/models";

const tabs = [
  { value: "deployments", label: "Deployments" },
  { value: "settings", label: "Settings" },
  { value: "domains", label: "Domains" },
  { value: "environment", label: "Environment" },
  { value: "notifications", label: "Notifications" },
] as const;

interface ProjectTabsProps {
  project: Project;
}

export function ProjectTabs({ project }: ProjectTabsProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const currentTab = searchParams.get("tab") || "deployments";

  const handleTabChange = (value: string) => {
    setSearchParams({ tab: value }, { replace: true });
  };

  return (
    <Tabs value={currentTab} onValueChange={handleTabChange}>
      <TabsList className="w-full justify-start">
        {tabs.map((tab) => (
          <TabsTrigger key={tab.value} value={tab.value}>
            {tab.label}
          </TabsTrigger>
        ))}
      </TabsList>

      <div className="mt-6">
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
