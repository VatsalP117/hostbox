import { useNavigate } from "react-router-dom";
import { StatusBadge } from "@/components/shared/status-badge";
import { CommitInfo } from "@/components/shared/commit-info";
import { TimeAgo } from "@/components/shared/time-ago";
import { Badge } from "@/components/ui/badge";
import { routes } from "@/lib/constants";
import { formatDuration } from "@/lib/utils";
import type { Deployment } from "@/types/models";
import { GitBranch } from "lucide-react";

interface DeploymentRowProps {
  deployment: Deployment;
  projectId: string;
}

export function DeploymentRow({ deployment, projectId }: DeploymentRowProps) {
  const navigate = useNavigate();

  return (
    <div
      className="flex items-center justify-between rounded-lg border p-4 cursor-pointer hover:bg-accent/50 transition-colors"
      onClick={() =>
        navigate(routes.deployment(projectId, deployment.id))
      }
    >
      <div className="flex items-center gap-4 min-w-0">
        <StatusBadge status={deployment.status} />
        <div className="min-w-0 space-y-1">
          <CommitInfo
            sha={deployment.commit_sha}
            message={deployment.commit_message}
            author={deployment.commit_author}
          />
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <GitBranch className="h-3 w-3" />
            <span>{deployment.branch}</span>
            {deployment.is_production && (
              <Badge variant="outline" className="text-[10px] py-0">
                Production
              </Badge>
            )}
            {deployment.is_rollback && (
              <Badge variant="secondary" className="text-[10px] py-0">
                Rollback
              </Badge>
            )}
          </div>
        </div>
      </div>

      <div className="flex items-center gap-4 shrink-0">
        {deployment.build_duration_ms && (
          <span className="text-xs text-muted-foreground">
            {formatDuration(deployment.build_duration_ms)}
          </span>
        )}
        <TimeAgo date={deployment.created_at} />
      </div>
    </div>
  );
}
