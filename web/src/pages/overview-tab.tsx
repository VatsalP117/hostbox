import { useNavigate, Link } from "react-router-dom";
import { useDeployments } from "@/hooks/use-deployments";
import { useDomains } from "@/hooks/use-domains";
import { useEnvVars } from "@/hooks/use-env-vars";
import { useNotifications } from "@/hooks/use-notifications";
import { StatusBadge } from "@/components/shared/status-badge";
import { TimeAgo } from "@/components/shared/time-ago";
import { CopyButton } from "@/components/shared/copy-button";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { routes } from "@/lib/constants";
import { cn, formatDuration } from "@/lib/utils";
import type { Project, Deployment, Domain } from "@/types/models";
import {
  Rocket,
  Globe,
  GitBranch,
  CheckCircle2,
  AlertCircle,
  Clock,
  ExternalLink,
  Database,
  MessageSquare,
  ArrowRight,
  Copy,
  Check,
} from "lucide-react";
import { useState } from "react";

interface OverviewTabProps {
  project: Project;
}

interface DeploymentItemProps {
  deployment: Deployment;
  projectId: string;
}

function DeploymentItem({ deployment, projectId }: DeploymentItemProps) {
  const navigate = useNavigate();

  const statusIcons: Record<string, React.ReactNode> = {
    ready: <Rocket className="h-5 w-5 text-primary-container" />,
    failed: <AlertCircle className="h-5 w-5 text-error" />,
    building: <Clock className="h-5 w-5 text-outline" />,
    queued: <Clock className="h-5 w-5 text-outline" />,
    cancelled: <AlertCircle className="h-5 w-5 text-outline" />,
  };

  return (
    <div
      onClick={() => navigate(routes.deployment(projectId, deployment.id))}
      className={cn(
        "bg-surface-container rounded-lg p-4 flex items-start gap-4",
        "hover:bg-surface-container-high transition-colors cursor-pointer group"
      )}
    >
      <div className="mt-0.5">{statusIcons[deployment.status]}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <p className="font-body text-sm font-medium text-on-surface truncate">
            {deployment.commit_message || "No commit message"}
          </p>
          <span className="font-label text-xs text-outline group-hover:text-primary-container transition-colors shrink-0 ml-2">
            <TimeAgo date={deployment.created_at} />
          </span>
        </div>
        <p className="font-label text-xs text-on-surface-variant mb-2">
          Commit{" "}
          <span className="text-primary-container">
            {deployment.commit_sha.slice(0, 7)}
          </span>{" "}
          - {deployment.branch}
        </p>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="px-2 py-0.5 rounded bg-surface-container-lowest font-label text-[10px] text-outline ghost-border">
            {deployment.branch}
          </span>
          {deployment.is_production && (
            <span className="px-2 py-0.5 rounded bg-surface-container-lowest font-label text-[10px] text-primary-container ghost-border">
              Production
            </span>
          )}
          <span
            className={cn(
              "px-2 py-0.5 rounded bg-surface-container-lowest font-label text-[10px] ghost-border",
              deployment.status === "failed"
                ? "text-error"
                : deployment.status === "ready"
                  ? "text-primary-container"
                  : "text-outline"
            )}
          >
            {deployment.status.charAt(0).toUpperCase() +
              deployment.status.slice(1)}
          </span>
        </div>
      </div>
    </div>
  );
}

interface DomainCardProps {
  domain: Domain;
}

function DomainCard({ domain }: DomainCardProps) {
  const [copied, setCopied] = useState(false);
  const url = `https://${domain.domain}`;

  const handleCopy = async () => {
    await navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-surface-container rounded-xl p-5">
      <div className="flex items-center gap-3 mb-4">
        <div className="w-10 h-10 rounded-lg bg-surface-container-high flex items-center justify-center">
          <Globe className="h-5 w-5 text-primary-container" />
        </div>
        <div>
          <h4 className="font-label text-sm text-on-surface-variant uppercase tracking-wider">
            Production Domain
          </h4>
        </div>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between bg-surface-container-low rounded-lg p-3">
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="font-body text-sm text-on-surface hover:text-primary-container transition-colors flex items-center gap-2"
          >
            {domain.domain}
            <ExternalLink className="h-3.5 w-3.5" />
          </a>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={handleCopy}
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-green-500" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
          </Button>
        </div>

        <div className="flex items-center gap-2">
          {domain.verified ? (
            <>
              <div className="w-1.5 h-1.5 rounded-full bg-primary-container glow-active" />
              <span className="font-label text-xs text-primary-container">
                Verified & Propagated
              </span>
            </>
          ) : (
            <>
              <div className="w-1.5 h-1.5 rounded-full bg-outline" />
              <span className="font-label text-xs text-outline">
                Pending Verification
              </span>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function EmptyDomainCard() {
  return (
    <div className="bg-surface-container rounded-xl p-5">
      <div className="flex items-center gap-3 mb-4">
        <div className="w-10 h-10 rounded-lg bg-surface-container-high flex items-center justify-center">
          <Globe className="h-5 w-5 text-outline" />
        </div>
        <div>
          <h4 className="font-label text-sm text-on-surface-variant uppercase tracking-wider">
            Production Domain
          </h4>
        </div>
      </div>
      <p className="font-body text-sm text-on-surface-variant">
        No verified domains yet.
      </p>
      <Button
        variant="outline"
        size="sm"
        className="mt-3 font-label text-xs"
        asChild
      >
        <Link to={`?tab=domains`}>Add Domain</Link>
      </Button>
    </div>
  );
}

export function OverviewTab({ project }: OverviewTabProps) {
  const { data: deploymentsData, isLoading: isLoadingDeployments } =
    useDeployments(project.id, { page: 1, per_page: 5 });
  const { data: domainsData, isLoading: isLoadingDomains } = useDomains(
    project.id
  );
  const { data: envVarsData, isLoading: isLoadingEnvVars } = useEnvVars(
    project.id
  );
  const { data: notificationsData, isLoading: isLoadingNotifications } =
    useNotifications(project.id);

  const deployments = deploymentsData?.deployments || [];
  const domains = domainsData?.domains || [];
  const envVars = envVarsData?.env_vars || [];
  const notifications = notificationsData?.notifications || [];

  const verifiedDomains = domains.filter((d) => d.verified);
  const productionDomain = verifiedDomains[0] || domains[0];

  const hasNotifications = notifications.length > 0;
  const discordNotification = notifications.find(
    (n) => n.channel === "discord" && n.enabled
  );

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
      {/* Left Column: Recent Deployments */}
      <div className="lg:col-span-2">
        <div className="bg-surface-container-low rounded-xl p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="font-label text-sm text-on-surface-variant uppercase tracking-widest">
              Recent Deployments
            </h3>
            <Link
              to={`?tab=deployments`}
              className="font-label text-xs text-primary-container hover:underline decoration-primary-container/30 underline-offset-4 flex items-center gap-1"
            >
              View All
              <ArrowRight className="h-3 w-3" />
            </Link>
          </div>

          {isLoadingDeployments ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-20 w-full rounded-lg" />
              ))}
            </div>
          ) : deployments.length > 0 ? (
            <div className="space-y-3">
              {deployments.slice(0, 5).map((deployment) => (
                <DeploymentItem
                  key={deployment.id}
                  deployment={deployment}
                  projectId={project.id}
                />
              ))}
            </div>
          ) : (
            <div className="text-center py-12 bg-surface-container rounded-lg">
              <Rocket className="h-8 w-8 text-outline mx-auto mb-3" />
              <p className="font-body text-sm text-on-surface-variant">
                No deployments yet
              </p>
              <p className="font-label text-xs text-outline mt-1">
                Trigger your first deployment to get started
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Right Column: Domain + Config */}
      <div className="lg:col-span-1 space-y-6">
        {/* Production Domain */}
        {isLoadingDomains ? (
          <Skeleton className="h-40 w-full rounded-xl" />
        ) : productionDomain ? (
          <DomainCard domain={productionDomain} />
        ) : (
          <EmptyDomainCard />
        )}

        {/* Configuration Summary */}
        <div className="bg-surface-container-low rounded-xl p-6">
          <h3 className="font-label text-sm text-on-surface-variant uppercase tracking-widest mb-6">
            Configuration
          </h3>

          <ul className="space-y-4">
            <li className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg bg-surface-container flex items-center justify-center text-primary-container">
                <Globe className="h-5 w-5" />
              </div>
              <div>
                <p className="font-body text-sm font-medium text-on-surface">
                  {domains.length} Domain{domains.length !== 1 ? "s" : ""}
                </p>
                <p className="font-label text-xs text-outline">
                  {verifiedDomains.length} Verified
                  {domains.length - verifiedDomains.length > 0
                    ? `, ${domains.length - verifiedDomains.length} Pending`
                    : ""}
                </p>
              </div>
            </li>

            <li className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg bg-surface-container flex items-center justify-center text-primary-container">
                <Database className="h-5 w-5" />
              </div>
              <div>
                <p className="font-body text-sm font-medium text-on-surface">
                  {envVars.length} Env Var{envVars.length !== 1 ? "s" : ""}
                </p>
                <p className="font-label text-xs text-outline">
                  Production Environment
                </p>
              </div>
            </li>

            <li className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg bg-surface-container flex items-center justify-center text-primary-container">
                <MessageSquare className="h-5 w-5" />
              </div>
              <div>
                <p className="font-body text-sm font-medium text-on-surface">
                  {hasNotifications ? "Notifications Active" : "No Notifications"}
                </p>
                <p className="font-label text-xs text-outline">
                  {discordNotification
                    ? `Discord on #deployments`
                    : "Configure alerts"}
                </p>
              </div>
            </li>
          </ul>
        </div>

        {/* Environment Variables Preview */}
        {envVars.length > 0 && (
          <div className="bg-surface-container-low rounded-xl p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-label text-sm text-on-surface-variant uppercase tracking-widest">
                Environment Variables
              </h3>
              <Link
                to={`?tab=environment`}
                className="font-label text-xs text-primary-container hover:underline decoration-primary-container/30 underline-offset-4"
              >
                Manage
              </Link>
            </div>
            <div className="space-y-2">
              {envVars.slice(0, 3).map((envVar) => (
                <div
                  key={envVar.id}
                  className="flex items-center justify-between bg-surface-container rounded-lg p-3"
                >
                  <code className="font-mono text-xs text-on-surface">
                    {envVar.key}
                  </code>
                  <span className="font-label text-[10px] text-outline uppercase px-2 py-0.5 rounded bg-surface-container-lowest ghost-border">
                    {envVar.scope}
                  </span>
                </div>
              ))}
              {envVars.length > 3 && (
                <p className="font-label text-xs text-center text-outline pt-2">
                  +{envVars.length - 3} more
                </p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
