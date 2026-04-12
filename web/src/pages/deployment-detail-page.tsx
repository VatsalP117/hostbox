import { useParams, useNavigate } from "react-router-dom";
import { useDeployment } from "@/hooks/use-deployments";
import { useDeploymentLogs } from "@/hooks/use-deployment-logs";
import { DeploymentHeader } from "@/components/deployments/deployment-header";
import { BuildProgress } from "@/components/deployments/build-progress";
import { LogViewer } from "@/components/deployments/log-viewer";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { routes } from "@/lib/constants";
import { ArrowLeft } from "lucide-react";

export function DeploymentDetailPage() {
  const { id, deploymentId } = useParams<{
    id: string;
    deploymentId: string;
  }>();
  const navigate = useNavigate();

  const { data, isLoading } = useDeployment(deploymentId!);
  const deployment = data?.deployment;

  const isActive =
    deployment?.status === "queued" || deployment?.status === "building";

  const { lines, status: phase, isConnected } = useDeploymentLogs(
    deploymentId!,
    { enabled: !!deployment && isActive },
  );

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-lg" />
        <Skeleton className="h-[500px] w-full rounded-lg" />
      </div>
    );
  }

  if (!deployment) return null;

  return (
    <div className="space-y-6">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate(routes.project(id!))}
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to project
      </Button>

      <DeploymentHeader deployment={deployment} />

      {isActive && (
        <BuildProgress phase={phase} status={deployment.status} />
      )}

      <LogViewer
        lines={lines}
        isStreaming={isConnected}
      />
    </div>
  );
}
