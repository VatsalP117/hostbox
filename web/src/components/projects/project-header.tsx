import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/status-badge";
import { TimeAgo } from "@/components/shared/time-ago";
import { routes } from "@/lib/constants";
import { getApiErrorMessage, cn } from "@/lib/utils";
import { useTriggerDeployment } from "@/hooks/use-deployments";
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
  const trigger = useTriggerDeployment(project.id);

  const productionDomain = domains.find((d) => d.verified);
  const productionUrl = productionDomain
    ? `https://${productionDomain.domain}`
    : latestDeployment?.deployment_url;

  const framework = project.framework;
  const fwConfig = framework
    ? frameworkConfigMap[framework]
    : { label: "Unknown", icon: "HelpCircle" };
  const FrameworkIcon = iconMap[fwConfig?.icon] ?? HelpCircle;

  const handleDeploy = () => {
    trigger.mutate(
      { branch: project.production_branch },
      {
        onSuccess: (data) => {
          toast.success("Deployment triggered");
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
    <div className="space-y-8">
      {/* Main Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-3">
          {/* Status Label */}
          <div className="flex items-center gap-2">
            <div
              className={cn(
                "w-2 h-2 rounded-full",
                project.status === "healthy"
                  ? "bg-primary-container glow-active"
                  : project.status === "failed"
                    ? "bg-error glow-error"
                    : project.status === "building"
                      ? "bg-warning"
                      : "bg-outline",
              )}
            />
            <span className="font-label text-xs text-primary-container tracking-widest uppercase">
              {projectStateLabel}
            </span>
          </div>

          {/* Title Row */}
          <div className="flex items-center gap-4">
            {/* Framework Icon */}
            <div className="w-12 h-12 rounded-xl bg-surface-container border border-outline-variant/20 flex items-center justify-center">
              <FrameworkIcon className="h-6 w-6 text-on-surface-variant" />
            </div>

            <div>
              <h1 className="font-headline text-3xl font-bold text-on-surface">
                {project.name}
              </h1>
              <p className="font-body text-on-surface-variant mt-1">
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
              className="font-label font-medium bg-surface-container-low border-outline-variant/20 hover:bg-surface-container text-on-surface"
              onClick={() => window.open(productionUrl, "_blank")}
            >
              <Globe className="mr-2 h-4 w-4" />
              View Live Site
            </Button>
          )}

          <Button
            onClick={handleDeploy}
            disabled={trigger.isPending}
            className="font-headline font-bold bg-primary text-on-primary hover:bg-primary/90"
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
        <div className="bg-surface-container-low rounded-xl p-4">
          <p className="font-label text-xs text-on-surface-variant uppercase tracking-wider mb-1">
            Total Deployments
          </p>
          <p className="font-headline text-2xl font-bold text-on-surface">
            {stats.total_deployments}
          </p>
        </div>

        <div className="bg-surface-container-low rounded-xl p-4">
          <p className="font-label text-xs text-on-surface-variant uppercase tracking-wider mb-1">
            Last Deploy
          </p>
          <p className="font-headline text-2xl font-bold text-on-surface">
            {stats.last_deploy_at ? (
              <TimeAgo date={stats.last_deploy_at} />
            ) : (
              "—"
            )}
          </p>
        </div>

        <div className="bg-surface-container-low rounded-xl p-4">
          <p className="font-label text-xs text-on-surface-variant uppercase tracking-wider mb-1">
            Avg Build Time
          </p>
          <p className="font-headline text-2xl font-bold text-on-surface">
            {avgBuildTime ? `${avgBuildTime}s` : "—"}
          </p>
        </div>

        <div className="bg-surface-container-low rounded-xl p-4">
          <p className="font-label text-xs text-on-surface-variant uppercase tracking-wider mb-1">
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
