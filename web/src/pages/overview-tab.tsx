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
    ready: <Rocket className="h-4 w-4 text-success" />,
    failed: <AlertCircle className="h-5 w-5 text-error" />,
    building: <Clock className="h-5 w-5 text-outline" />,
    queued: <Clock className="h-5 w-5 text-outline" />,
    cancelled: <AlertCircle className="h-5 w-5 text-outline" />,
  };

  return (
    <div
      onClick={() => navigate(routes.deployment(projectId, deployment.id))}
      className={cn(
        "group flex cursor-pointer items-start gap-3 border-b border-border px-1 py-4 last:border-0",
        "transition-colors hover:bg-white/[0.02]"
      )}
    >
      <div className="mt-0.5">{statusIcons[deployment.status]}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <p className="font-body text-sm font-medium text-on-surface truncate">
            {deployment.commit_message || "No commit message"}
          </p>
          <span className="ml-2 shrink-0 text-xs text-muted-foreground transition-colors group-hover:text-foreground">
            <TimeAgo date={deployment.created_at} />
          </span>
        </div>
        <p className="mb-2 text-xs text-on-surface-variant">
          Commit{" "}
          <span className="font-mono text-foreground">
            {deployment.commit_sha.slice(0, 7)}
          </span>{" "}
          - {deployment.branch}
        </p>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="rounded border border-border bg-black px-2 py-0.5 text-[10px] text-muted-foreground">
            {deployment.branch}
          </span>
          {deployment.is_production && (
            <span className="rounded border border-border bg-black px-2 py-0.5 text-[10px] text-foreground">
              Production
            </span>
          )}
          <span
            className={cn(
              "rounded border border-border bg-black px-2 py-0.5 text-[10px]",
              deployment.status === "failed"
                ? "text-error"
                : deployment.status === "ready"
                  ? "text-success"
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
    <div className="rounded-lg border border-border bg-card p-5">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-black">
          <Globe className="h-4 w-4 text-[#52a8ff]" />
        </div>
        <div>
          <h4 className="text-sm font-medium text-on-surface">
            Production domain
          </h4>
        </div>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between rounded-md border border-border bg-black p-3">
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
              <div className="h-1.5 w-1.5 rounded-full bg-success" />
              <span className="text-xs text-success">
                Verified and active
              </span>
            </>
          ) : (
            <>
              <div className="w-1.5 h-1.5 rounded-full bg-outline" />
              <span className="text-xs text-muted-foreground">
                Pending verification
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
    <div className="rounded-lg border border-dashed border-border bg-card/50 p-5">
      <div className="flex items-center gap-3 mb-4">
        <div className="w-10 h-10 rounded-lg bg-surface-container-high flex items-center justify-center">
          <Globe className="h-5 w-5 text-outline" />
        </div>
        <div>
          <h4 className="text-sm font-medium text-on-surface">
            Production domain
          </h4>
        </div>
      </div>
      <p className="font-body text-sm text-on-surface-variant">
        No verified domains yet.
      </p>
      <Button
        variant="outline"
        size="sm"
        className="mt-3 font-sans text-xs"
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
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
      {/* Left Column: Recent Deployments */}
      <div className="lg:col-span-2">
        <div className="rounded-lg border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-6">
            <div><h3 className="text-sm font-medium text-on-surface">Recent deployments</h3><p className="mt-1 text-xs text-muted-foreground">Latest builds across this project.</p></div>
            <Link
              to={`?tab=deployments`}
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            >
              View all
              <ArrowRight className="h-3 w-3" />
            </Link>
          </div>

          {isLoadingDeployments ? (
            <div>
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
            <div className="rounded-md border border-dashed border-border bg-black/30 py-12 text-center">
              <Rocket className="h-8 w-8 text-outline mx-auto mb-3" />
              <p className="font-body text-sm text-on-surface-variant">
                No deployments yet
              </p>
              <p className="font-sans text-xs text-outline mt-1">
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
          <Skeleton className="h-40 w-full rounded-lg" />
        ) : productionDomain ? (
          <DomainCard domain={productionDomain} />
        ) : (
          <EmptyDomainCard />
        )}

        {/* Configuration Summary */}
        <div className="rounded-lg border border-border bg-card p-5">
          <h3 className="mb-5 text-sm font-medium text-on-surface">
            Configuration
          </h3>

          <ul className="space-y-4">
            <li className="flex items-center gap-4">
              <div className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-black text-muted-foreground">
                <Globe className="h-5 w-5" />
              </div>
              <div>
                <p className="font-body text-sm font-medium text-on-surface">
                  {domains.length} Domain{domains.length !== 1 ? "s" : ""}
                </p>
                <p className="font-sans text-xs text-outline">
                  {verifiedDomains.length} Verified
                  {domains.length - verifiedDomains.length > 0
                    ? `, ${domains.length - verifiedDomains.length} Pending`
                    : ""}
                </p>
              </div>
            </li>

            <li className="flex items-center gap-4">
              <div className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-black text-muted-foreground">
                <Database className="h-5 w-5" />
              </div>
              <div>
                <p className="font-body text-sm font-medium text-on-surface">
                  {envVars.length} Env Var{envVars.length !== 1 ? "s" : ""}
                </p>
                <p className="font-sans text-xs text-outline">
                  Production Environment
                </p>
              </div>
            </li>

            <li className="flex items-center gap-4">
              <div className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-black text-muted-foreground">
                <MessageSquare className="h-5 w-5" />
              </div>
              <div>
                <p className="font-body text-sm font-medium text-on-surface">
                  {hasNotifications ? "Notifications Active" : "No Notifications"}
                </p>
                <p className="font-sans text-xs text-outline">
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
          <div className="rounded-lg border border-border bg-card p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-medium text-on-surface">
                Environment variables
              </h3>
              <Link
                to={`?tab=environment`}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Manage
              </Link>
            </div>
            <div className="space-y-2">
              {envVars.slice(0, 3).map((envVar) => (
                <div
                  key={envVar.id}
                  className="flex items-center justify-between rounded-md border border-border bg-black p-3"
                >
                  <code className="font-mono text-xs text-on-surface">
                    {envVar.key}
                  </code>
                  <span className="font-sans text-[10px] text-outline uppercase px-2 py-0.5 rounded bg-surface-container-lowest ghost-border">
                    {envVar.scope}
                  </span>
                </div>
              ))}
              {envVars.length > 3 && (
                <p className="font-sans text-xs text-center text-outline pt-2">
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
