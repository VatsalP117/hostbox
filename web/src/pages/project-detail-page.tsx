import { useParams } from "react-router-dom";
import { useProject } from "@/hooks/use-projects";
import { ProjectHeader } from "@/components/projects/project-header";
import { ProjectTabs } from "@/components/projects/project-tabs";
import { Skeleton } from "@/components/ui/skeleton";

export function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading } = useProject(id!);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-10 w-96" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    );
  }

  if (!data) return null;

  return (
    <div className="space-y-6">
      <ProjectHeader
        project={data.project}
        latestDeployment={data.latest_deployment}
        domains={data.domains}
      />
      <ProjectTabs project={data.project} />
    </div>
  );
}
