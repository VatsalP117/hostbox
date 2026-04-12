import { StatusBadge } from "@/components/shared/status-badge";
import { CommitInfo } from "@/components/shared/commit-info";
import { TimeAgo } from "@/components/shared/time-ago";
import { ExternalLink } from "@/components/shared/external-link";
import { DeploymentActions } from "@/components/deployments/deployment-actions";
import { Badge } from "@/components/ui/badge";
import { formatDuration, formatBytes } from "@/lib/utils";
import type { Deployment } from "@/types/models";
import { GitBranch, Clock, HardDrive } from "lucide-react";

interface DeploymentHeaderProps {
  deployment: Deployment;
}

export function DeploymentHeader({ deployment }: DeploymentHeaderProps) {
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-2">
          <div className="flex items-center gap-3">
            <StatusBadge status={deployment.status} />
            {deployment.is_production && (
              <Badge variant="outline">Production</Badge>
            )}
            {deployment.is_rollback && (
              <Badge variant="secondary">Rollback</Badge>
            )}
          </div>

          <CommitInfo
            sha={deployment.commit_sha}
            message={deployment.commit_message}
            author={deployment.commit_author}
          />

          <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <GitBranch className="h-3.5 w-3.5" />
              {deployment.branch}
            </div>
            {deployment.build_duration_ms && (
              <div className="flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5" />
                {formatDuration(deployment.build_duration_ms)}
              </div>
            )}
            {deployment.artifact_size_bytes && (
              <div className="flex items-center gap-1.5">
                <HardDrive className="h-3.5 w-3.5" />
                {formatBytes(deployment.artifact_size_bytes)}
              </div>
            )}
            <TimeAgo date={deployment.created_at} />
          </div>

          {deployment.deployment_url && (
            <ExternalLink href={deployment.deployment_url}>
              {deployment.deployment_url.replace(/^https?:\/\//, "")}
            </ExternalLink>
          )}
        </div>

        <DeploymentActions deployment={deployment} />
      </div>

      {deployment.error_message && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {deployment.error_message}
        </div>
      )}
    </div>
  );
}
