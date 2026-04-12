import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/utils";
import {
  useCancelDeployment,
  useRollbackDeployment,
  useRedeployment,
} from "@/hooks/use-deployments";
import type { Deployment } from "@/types/models";
import { Loader2, XCircle, RotateCcw, RefreshCw } from "lucide-react";

interface DeploymentActionsProps {
  deployment: Deployment;
}

export function DeploymentActions({ deployment }: DeploymentActionsProps) {
  const cancel = useCancelDeployment();
  const rollback = useRollbackDeployment();
  const redeploy = useRedeployment();

  const isActive =
    deployment.status === "queued" || deployment.status === "building";

  const handleCancel = () => {
    cancel.mutate(deployment.id, {
      onSuccess: () => toast.success("Deployment cancelled"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  const handleRollback = () => {
    rollback.mutate(deployment.id, {
      onSuccess: () => toast.success("Rollback triggered"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  const handleRedeploy = () => {
    redeploy.mutate(deployment.id, {
      onSuccess: () => toast.success("Redeployment triggered"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  return (
    <div className="flex items-center gap-2">
      {isActive && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleCancel}
          disabled={cancel.isPending}
        >
          {cancel.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <XCircle className="mr-2 h-4 w-4" />
          )}
          Cancel
        </Button>
      )}

      {deployment.status === "ready" && deployment.is_production && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleRollback}
          disabled={rollback.isPending}
        >
          {rollback.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RotateCcw className="mr-2 h-4 w-4" />
          )}
          Rollback
        </Button>
      )}

      {(deployment.status === "ready" || deployment.status === "failed") && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleRedeploy}
          disabled={redeploy.isPending}
        >
          {redeploy.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Redeploy
        </Button>
      )}
    </div>
  );
}
