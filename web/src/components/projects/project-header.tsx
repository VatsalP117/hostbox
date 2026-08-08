import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/status-badge";
import { TimeAgo } from "@/components/shared/time-ago";
import { routes } from "@/lib/constants";
import { getApiErrorMessage, cn } from "@/lib/utils";
import { useDeployLatest } from "@/hooks/use-deployments";
import { frameworkConfig as frameworkConfigMap } from "@/lib/constants";
import type { Project, Deployment, Domain, ProjectStats } from "@/types/models";
import {
  GitBranch,
  Rocket,
  ExternalLink,
  Github,
  Globe,
  Clock,
  CheckCircle2,
  Hexagon,
  Zap,
  Atom,
  Circle,
  Triangle,
  Flame,
  FileText,
  HelpCircle,
  Loader2,
  AlertTriangle,
} from "lucide-react";

const iconMap: Record<string, React.ElementType> = {
  Hexagon,
  Zap,
  Atom,
  Rocket,
  Circle,
  Triangle,
  Flame,
  FileText,
  Globe: Globe,
  HelpCircle,
};

interface ProjectHeaderProps {
  project: Project;
  latestDeployment: Deployment | null;
  domains: Domain[];
  stats: ProjectStats;
}

export function ProjectHeader({
  project,
  latestDeployment,
  domains,
  stats,
}: ProjectHeaderProps) {
  const navigate = useNavigate();
  const trigger = useDeployLatest();

  const productionDomain = domains.find((d) => d.verified);
  const productionUrl = productionDomain
    ? `https://${productionDomain.domain}`
    : latestDeployment?.deployment_url;

  const framework = project.framework;
  const fwConfig = framework
    ? frameworkConfigMap[framework]
    : { label: "Unknown", icon: "HelpCircle" };
  const FrameworkIcon = iconMap[fwConfig?.icon] ?? HelpCircle;
  const githubUnavailable =
    Boolean(project.github_installation_id) &&
    project.github_connection_status !== "active";

  const handleDeploy = () => {
    trigger.mutate(
      { projectId: project.id, branch: project.production_branch },
      {
        onSuccess: (data) => {
          toast.success("Latest branch deployment triggered");
          navigate(routes.deployment(project.id, data.deployment.id));
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const avgBuildTime = stats.average_build_time_ms
    ? Math.round(stats.average_build_time_ms / 1000)
    : null;
  const projectStateLabel =
    project.status === "healthy"
      ? "Production Ready"
      : project.status === "failed"
        ? "Deployment Failed"
        : project.status === "building"
          ? "Building"
          : project.status === "stopped"
            ? "No Deployments"
            : "Unknown";

  return (
    <div className="space-y-6">
      {/* Main Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-3">
          {/* Status Label */}
          <div className="flex items-center gap-2">
            <div
              className={cn(
                "w-2 h-2 rounded-full",
                project.status === "healthy"
                  ? "bg-success"
                  : project.status === "failed"
                    ? "bg-destructive"
                    : project.status === "building"
                      ? "bg-warning"
                      : "bg-outline",
              )}
            />
            <span className="font-sans text-xs text-muted-foreground">
              {projectStateLabel}
            </span>
          </div>

          {/* Title Row */}
          <div className="flex items-center gap-4">
            {/* Framework Icon */}
            <div className="flex h-10 w-10 items-center justify-center rounded-md border border-border bg-black">
              <FrameworkIcon className="h-5 w-5 text-on-surface-variant" />
            </div>

            <div>
              <h1 className="text-2xl font-semibold tracking-tight text-on-surface">
                {project.name}
              </h1>
              <p className="mt-1 text-sm text-on-surface-variant">
                {fwConfig?.label || "Unknown Framework"}
              </p>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-3">
          {productionUrl && (
            <Button
              variant="outline"
              size="default"
              className="border-border bg-black font-sans font-medium text-on-surface hover:bg-accent"
              onClick={() => window.open(productionUrl, "_blank")}
            >
              <Globe className="mr-2 h-4 w-4" />
              View Live Site
            </Button>
          )}

          <Button
            onClick={handleDeploy}
            disabled={trigger.isPending || githubUnavailable}
            className="font-sans font-medium"
          >
            {trigger.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Rocket className="mr-2 h-4 w-4" />
            )}
            Deploy
          </Button>
        </div>
      </div>

      {/* Metadata Row */}
      <div className="flex flex-wrap items-center gap-6 text-sm">
        {project.github_repo && (
          <a
            href={`https://github.com/${project.github_repo}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-on-surface-variant hover:text-on-surface transition-colors"
          >
            <Github className="h-4 w-4" />
            <span className="font-body">{project.github_repo}</span>
            <ExternalLink className="h-3 w-3" />
          </a>
        )}
        {githubUnavailable && (
          <span className="flex items-center gap-2 text-warning">
            <AlertTriangle className="h-4 w-4" />
            GitHub {project.github_connection_status.replace("_", " ")}
          </span>
        )}

        <div className="flex items-center gap-2 text-on-surface-variant">
          <GitBranch className="h-4 w-4" />
          <span className="font-body">{project.production_branch}</span>
        </div>

        <div className="flex items-center gap-2 text-on-surface-variant">
          <Clock className="h-4 w-4" />
          <span className="font-body">
            Created{" "}
            <TimeAgo date={project.created_at} />
          </span>
        </div>
      </div>

      {/* Stats Bar */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="mb-1 text-xs text-on-surface-variant">
            Total Deployments
          </p>
          <p className="text-xl font-semibold text-on-surface">
            {stats.total_deployments}
          </p>
        </div>

        <div className="rounded-lg border border-border bg-card p-4">
          <p className="mb-1 text-xs text-on-surface-variant">
            Last Deploy
          </p>
          <p className="text-xl font-semibold text-on-surface">
            {stats.last_deploy_at ? (
              <TimeAgo date={stats.last_deploy_at} />
            ) : (
              "—"
            )}
          </p>
        </div>

        <div className="rounded-lg border border-border bg-card p-4">
          <p className="mb-1 text-xs text-on-surface-variant">
            Avg Build Time
          </p>
          <p className="text-xl font-semibold text-on-surface">
            {avgBuildTime ? `${avgBuildTime}s` : "—"}
          </p>
        </div>

        <div className="rounded-lg border border-border bg-card p-4">
          <p className="mb-1 font-sans text-xs text-on-surface-variant">
            Current Status
          </p>
          <div className="flex items-center gap-2">
            {latestDeployment ? (
              <StatusBadge status={latestDeployment.status} />
            ) : project.status === "stopped" ? (
              <span className="font-body text-on-surface-variant">Not deployed</span>
            ) : (
              <span className="font-body text-on-surface-variant">—</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
