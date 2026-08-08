import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { routes, statusConfig } from "@/lib/constants";
import { formatDuration, getApiErrorMessage, truncateSha } from "@/lib/utils";
import {
  useCancelDeployment,
  useRedeployment,
  useRollbackDeployment,
} from "@/hooks/use-deployments";
import type { Deployment } from "@/types/models";
import {
  Clock,
  ExternalLink,
  GitBranch,
  Loader2,
  MoreHorizontal,
  RotateCcw,
  User,
  XCircle,
} from "lucide-react";

export function DeploymentHeader({ deployment }: { deployment: Deployment }) {
  const cancel = useCancelDeployment();
  const rollback = useRollbackDeployment();
  const redeploy = useRedeployment();
  const [showMoreActions, setShowMoreActions] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const status = statusConfig[deployment.status];
  const isActive = deployment.status === "queued" || deployment.status === "building";

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) setShowMoreActions(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  const handleCancel = () => cancel.mutate(deployment.id, {
    onSuccess: () => toast.success("Deployment cancelled"),
    onError: (err) => toast.error(getApiErrorMessage(err)),
  });
  const handleRollback = () => rollback.mutate({ projectId: deployment.project_id, deploymentId: deployment.id }, {
    onSuccess: () => toast.success("Rollback triggered"),
    onError: (err) => toast.error(getApiErrorMessage(err)),
  });
  const handleRedeploy = () => redeploy.mutate({ projectId: deployment.project_id, deploymentId: deployment.id }, {
    onSuccess: () => toast.success("Redeployment triggered"),
    onError: (err) => toast.error(getApiErrorMessage(err)),
  });

  return (
    <section className="space-y-5 border-b border-border pb-6">
      <nav className="flex items-center gap-2 text-xs text-muted-foreground" aria-label="Breadcrumb">
        <Link to={routes.project(deployment.project_id)} className="hover:text-foreground">Project</Link>
        <span>/</span>
        <span className="font-mono text-foreground">dep_{truncateSha(deployment.id)}</span>
      </nav>

      <div className="flex flex-col justify-between gap-5 lg:flex-row lg:items-end">
        <div className="min-w-0">
          <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
            <span className={`h-1.5 w-1.5 rounded-full ${status.dotClassName}`} />
            <span className="capitalize">{deployment.status}</span>
            <span>·</span>
            <span className="font-mono">{deployment.branch}@{truncateSha(deployment.commit_sha)}</span>
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            {deployment.commit_message || "Deployment details"}
          </h1>
          <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
            {deployment.commit_author && <span className="flex items-center gap-1.5"><User className="h-3.5 w-3.5" />{deployment.commit_author}</span>}
            <span className="flex items-center gap-1.5"><GitBranch className="h-3.5 w-3.5" />{deployment.branch}</span>
            {deployment.build_duration_ms && <span className="flex items-center gap-1.5"><Clock className="h-3.5 w-3.5" />{formatDuration(deployment.build_duration_ms)}</span>}
            {deployment.started_at && <span>Started {new Date(deployment.started_at).toLocaleString()}</span>}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {(deployment.status === "ready" || deployment.status === "failed") && (
            <Button onClick={handleRedeploy} disabled={redeploy.isPending}>
              {redeploy.isPending && <Loader2 className="animate-spin" />} Redeploy
            </Button>
          )}
          {isActive && <Button variant="outline" onClick={handleCancel} disabled={cancel.isPending}>{cancel.isPending ? <Loader2 className="animate-spin" /> : <XCircle />}Cancel</Button>}
          {deployment.deployment_url && <Button variant="outline" asChild><a href={deployment.deployment_url} target="_blank" rel="noreferrer"><ExternalLink />Visit</a></Button>}
          {deployment.status === "ready" && deployment.is_production && <Button variant="outline" onClick={handleRollback} disabled={rollback.isPending}>{rollback.isPending ? <Loader2 className="animate-spin" /> : <RotateCcw />}Rollback</Button>}
          <div className="relative" ref={dropdownRef}>
            <Button variant="outline" size="icon" onClick={() => setShowMoreActions((value) => !value)} aria-label="More deployment actions"><MoreHorizontal /></Button>
            {showMoreActions && (
              <div className="absolute right-0 z-50 mt-2 w-48 overflow-hidden rounded-md border border-border bg-[#111] p-1 shadow-xl">
                {[["Copy deployment ID", deployment.id], ["Copy commit SHA", deployment.commit_sha]].map(([label, value]) => (
                  <button key={label} className="w-full rounded px-3 py-2 text-left text-xs text-foreground hover:bg-accent" onClick={() => { navigator.clipboard.writeText(value); toast.success(`${label} copied`); setShowMoreActions(false); }}>{label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
