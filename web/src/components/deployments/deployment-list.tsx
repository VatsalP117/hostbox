import { DeploymentRow } from "@/components/deployments/deployment-row";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import type { Deployment } from "@/types/models";
import { Rocket } from "lucide-react";

interface DeploymentListProps {
  deployments: Deployment[] | undefined;
  isLoading: boolean;
  projectId: string;
}

export function DeploymentList({
  deployments,
  isLoading,
  projectId,
}: DeploymentListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-16 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (!deployments?.length) {
    return (
      <EmptyState
        icon={Rocket}
        title="No deployments"
        description="Trigger a deployment to get started."
      />
    );
  }

  return (
    <div className="space-y-2">
      {deployments.map((deployment) => (
        <DeploymentRow
          key={deployment.id}
          deployment={deployment}
          projectId={projectId}
        />
      ))}
    </div>
  );
}
