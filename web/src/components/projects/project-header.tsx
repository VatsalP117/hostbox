import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { FrameworkBadge } from "@/components/shared/framework-badge";
import { ExternalLink } from "@/components/shared/external-link";
import { StatusBadge } from "@/components/shared/status-badge";
import { TimeAgo } from "@/components/shared/time-ago";
import { routes } from "@/lib/constants";
import { getApiErrorMessage } from "@/lib/utils";
import { useTriggerDeployment } from "@/hooks/use-deployments";
import type { Project, Deployment, Domain } from "@/types/models";
import { GitBranch, Rocket, ExternalLink as ExternalLinkIcon } from "lucide-react";
import { Loader2 } from "lucide-react";

interface ProjectHeaderProps {
  project: Project;
  latestDeployment: Deployment | null;
  domains: Domain[];
}

export function ProjectHeader({
  project,
  latestDeployment,
  domains,
}: ProjectHeaderProps) {
  const navigate = useNavigate();
  const trigger = useTriggerDeployment(project.id);

  const productionDomain = domains.find((d) => d.verified);
  const productionUrl = productionDomain
    ? `https://${productionDomain.domain}`
    : latestDeployment?.deployment_url;

  const handleDeploy = () => {
    trigger.mutate(
      { branch: project.production_branch },
      {
        onSuccess: (data) => {
          toast.success("Deployment triggered");
          navigate(
            routes.deployment(project.id, data.deployment.id),
          );
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight">
              {project.name}
            </h1>
            <FrameworkBadge framework={project.framework} />
          </div>
          <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            {project.github_repo && (
              <ExternalLink
                href={`https://github.com/${project.github_repo}`}
              >
                <GitBranch className="h-3.5 w-3.5" />
                {project.github_repo}
              </ExternalLink>
            )}
            {productionUrl && (
              <ExternalLink href={productionUrl}>
                <ExternalLinkIcon className="h-3.5 w-3.5" />
                {productionUrl.replace(/^https?:\/\//, "")}
              </ExternalLink>
            )}
          </div>
        </div>
        <Button onClick={handleDeploy} disabled={trigger.isPending}>
          {trigger.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Rocket className="mr-2 h-4 w-4" />
          )}
          Deploy
        </Button>
      </div>

      {latestDeployment && (
        <div className="flex items-center gap-3 text-sm">
          <StatusBadge status={latestDeployment.status} />
          <span className="text-muted-foreground">
            {latestDeployment.branch}
          </span>
          <span className="font-mono text-xs text-muted-foreground">
            {latestDeployment.commit_sha.slice(0, 7)}
          </span>
          <TimeAgo date={latestDeployment.created_at} />
        </div>
      )}
    </div>
  );
}
