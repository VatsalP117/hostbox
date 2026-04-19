import { useState, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { routes } from "@/lib/constants";
import { getApiErrorMessage, truncateSha, formatDuration } from "@/lib/utils";
import {
  useCancelDeployment,
  useRollbackDeployment,
  useRedeployment,
} from "@/hooks/use-deployments";
import type { Deployment } from "@/types/models";
import { statusConfig } from "@/lib/constants";
import { GitBranch, Clock, User, ExternalLink, MoreHorizontal, Loader2, XCircle, RotateCcw, RefreshCw } from "lucide-react";

interface DeploymentHeaderProps {
  deployment: Deployment;
}

export function DeploymentHeader({ deployment }: DeploymentHeaderProps) {
  const cancel = useCancelDeployment();
  const rollback = useRollbackDeployment();
  const redeploy = useRedeployment();
  const [showMoreActions, setShowMoreActions] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowMoreActions(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const isActive =
    deployment.status === "queued" || deployment.status === "building";

  const status = statusConfig[deployment.status];

  const handleCancel = () => {
    cancel.mutate(deployment.id, {
      onSuccess: () => toast.success("Deployment cancelled"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  const handleRollback = () => {
    rollback.mutate(
      {
        projectId: deployment.project_id,
        deploymentId: deployment.id,
      },
      {
        onSuccess: () => toast.success("Rollback triggered"),
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const handleRedeploy = () => {
    redeploy.mutate(deployment.project_id, {
      onSuccess: () => toast.success("Redeployment triggered"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[hsl(30,4%,70%)] font-label text-sm">
        <Link
          to={routes.project(deployment.project_id)}
          className="hover:text-[hsl(220,100%,84%)] transition-colors"
        >
          Project
        </Link>
        <span className="text-[hsl(30,4%,50%)]">/</span>
        <span className="text-[hsl(30,4%,90%)]">dep_{truncateSha(deployment.id)}</span>
      </div>

      {/* Main Header Content */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        {/* Left: Title and Info */}
        <div className="space-y-3">
          {/* Title with Status */}
          <div className="flex items-center gap-3">
            <h1 className="font-headline text-4xl md:text-5xl font-extrabold tracking-tight text-[hsl(30,4%,90%)]">
              Deployment {deployment.status === "failed" ? "Failed" : deployment.status === "ready" ? "Complete" : "In Progress"}
            </h1>
          </div>

          {/* Metadata Row */}
          <div className="flex flex-wrap items-center gap-4 font-label text-sm text-[hsl(30,4%,70%)]">
            {/* Status Badge with Glowing Orb */}
            <span className="flex items-center gap-2">
              <span 
                className={`w-2 h-2 rounded-full ${status.dotClassName}`}
                style={
                  deployment.status === "failed" 
                    ? { boxShadow: "0 0 12px 2px rgba(255, 180, 171, 0.3)" }
                    : deployment.status === "ready"
                    ? { boxShadow: "0 0 12px 2px rgba(74, 222, 128, 0.3)" }
                    : deployment.status === "building"
                    ? { boxShadow: "0 0 12px 2px rgba(59, 130, 246, 0.3)" }
                    : undefined
                }
              />
              <span className="text-[hsl(30,4%,90%)] capitalize">{deployment.status}</span>
            </span>

            {/* Commit Info */}
            <span className="flex items-center gap-1">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z"/>
              </svg>
              {deployment.branch} @ {truncateSha(deployment.commit_sha)}
            </span>

            {/* Duration */}
            {deployment.build_duration_ms && (
              <span className="flex items-center gap-1">
                <Clock className="h-4 w-4" />
                {formatDuration(deployment.build_duration_ms)} duration
              </span>
            )}
          </div>

          {/* Commit Message */}
          {deployment.commit_message && (
            <p className="text-[hsl(30,4%,70%)] text-sm max-w-2xl">
              {deployment.commit_message}
            </p>
          )}
        </div>

        {/* Right: Action Buttons */}
        <div className="flex items-center gap-3">
          {/* Retry/Redeploy Button */}
          {(deployment.status === "ready" || deployment.status === "failed") && (
            <Button
              onClick={handleRedeploy}
              disabled={redeploy.isPending}
              className="gradient-btn px-5 py-2 rounded-xl text-sm font-bold shadow-[0_0_24px_rgba(173,198,255,0.15)] hover:shadow-[0_0_32px_rgba(173,198,255,0.25)] transition-all"
            >
              {redeploy.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              Redeploy
            </Button>
          )}

          {/* Cancel Button (if active) */}
          {isActive && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleCancel}
              disabled={cancel.isPending}
              className="px-4 py-2 border border-[hsl(220,10%,28%)]/15 rounded-xl text-[hsl(30,4%,90%)] font-label text-sm hover:bg-[hsl(0,0%,16%)] transition-colors bg-transparent"
            >
              {cancel.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <XCircle className="mr-2 h-4 w-4" />
              )}
              Cancel
            </Button>
          )}

          {/* View Live Button */}
          {deployment.deployment_url && (
            <a
              href={deployment.deployment_url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 border border-[hsl(220,10%,28%)]/15 rounded-xl text-[hsl(30,4%,90%)] font-label text-sm hover:bg-[hsl(0,0%,16%)] transition-colors"
            >
              <ExternalLink className="h-4 w-4" />
              View Live
            </a>
          )}

          {/* Rollback Button (for production ready deployments) */}
          {deployment.status === "ready" && deployment.is_production && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleRollback}
              disabled={rollback.isPending}
              className="px-4 py-2 border border-[hsl(220,10%,28%)]/15 rounded-xl text-[hsl(30,4%,90%)] font-label text-sm hover:bg-[hsl(0,0%,16%)] transition-colors bg-transparent"
            >
              {rollback.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RotateCcw className="mr-2 h-4 w-4" />
              )}
              Rollback
            </Button>
          )}

          {/* More Actions Dropdown */}
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setShowMoreActions(!showMoreActions)}
              className="p-2 rounded-xl border border-[hsl(220,10%,28%)]/15 text-[hsl(30,4%,70%)] hover:text-[hsl(30,4%,90%)] hover:bg-[hsl(0,0%,16%)] transition-colors"
            >
              <MoreHorizontal className="h-4 w-4" />
            </button>
            
            {showMoreActions && (
              <div className="absolute right-0 mt-2 w-48 bg-[hsl(0,0%,11%)] border border-[hsl(220,10%,28%)]/15 rounded-xl shadow-lg z-50 py-1">
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(deployment.id);
                    toast.success("Deployment ID copied");
                    setShowMoreActions(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-[hsl(30,4%,90%)] hover:bg-[hsl(0,0%,16%)] transition-colors"
                >
                  Copy Deployment ID
                </button>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(deployment.commit_sha);
                    toast.success("Commit SHA copied");
                    setShowMoreActions(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-[hsl(30,4%,90%)] hover:bg-[hsl(0,0%,16%)] transition-colors"
                >
                  Copy Commit SHA
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Additional Metadata */}
      <div className="flex flex-wrap items-center gap-6 text-sm text-[hsl(30,4%,70%)] font-label border-t border-[hsl(220,10%,28%)]/10 pt-4">
        {deployment.commit_author && (
          <div className="flex items-center gap-2">
            <User className="h-4 w-4" />
            <span>{deployment.commit_author}</span>
          </div>
        )}
        <div className="flex items-center gap-2">
          <GitBranch className="h-4 w-4" />
          <span>{deployment.branch}</span>
        </div>
        <div className="flex items-center gap-2 font-mono">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z"/>
          </svg>
          <span>{truncateSha(deployment.commit_sha)}</span>
        </div>
        {deployment.started_at && (
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4" />
            <span>Started {new Date(deployment.started_at).toLocaleString()}</span>
          </div>
        )}
      </div>
    </div>
  );
}
