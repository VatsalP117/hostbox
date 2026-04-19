import { useParams, useNavigate } from "react-router-dom";
import { useDeployment } from "@/hooks/use-deployments";
import { useDeploymentLogs } from "@/hooks/use-deployment-logs";
import { DeploymentHeader } from "@/components/deployments/deployment-header";
import { BuildProgress } from "@/components/deployments/build-progress";
import { LogViewer } from "@/components/deployments/log-viewer";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/lib/constants";
import { ChevronLeft } from "lucide-react";

export function DeploymentDetailPage() {
  const { id, deploymentId } = useParams<{
    id: string;
    deploymentId: string;
  }>();
  const navigate = useNavigate();

  const { data, isLoading } = useDeployment(deploymentId!);
  const deployment = data?.deployment;

  const isActive =
    deployment?.status === "queued" || deployment?.status === "building";

  const { lines, status: phase, isConnected } = useDeploymentLogs(
    deploymentId!,
    { enabled: !!deployment && isActive },
  );

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-48 w-full rounded-xl" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <Skeleton className="h-[500px] w-full rounded-xl lg:col-span-2" />
          <Skeleton className="h-[300px] w-full rounded-xl" />
        </div>
      </div>
    );
  }

  if (!deployment) return null;

  return (
    <div className="space-y-8">
      {/* Back Navigation */}
      <button
        onClick={() => navigate(routes.project(id!))}
        className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-[hsl(0,0%,11%)] hover:bg-[hsl(0,0%,16%)] transition-colors text-[hsl(30,4%,70%)] hover:text-[hsl(30,4%,90%)] text-sm font-label"
      >
        <ChevronLeft className="h-4 w-4" />
        Back to project
      </button>

      {/* Deployment Header */}
      <DeploymentHeader deployment={deployment} />

      {/* Build Progress (only for active deployments) */}
      {isActive && (
        <BuildProgress phase={phase} status={deployment.status} />
      )}

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column: Error Panel & Logs */}
        <div className="lg:col-span-2 space-y-6">
          {/* Error Panel (only for failed deployments) */}
          {deployment.status === "failed" && deployment.error_message && (
            <div className="relative bg-[hsl(0,0%,11%)] rounded-xl border border-[hsl(0,84%,30%)]/30 overflow-hidden">
              <div className="absolute top-0 left-0 w-1 h-full bg-[hsl(0,84%,60%)]" />
              <div className="p-6">
                <div className="flex items-center gap-3 mb-4 text-[hsl(0,84%,60%)]">
                  <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
                  </svg>
                  <h2 className="font-headline font-bold text-xl tracking-tight">Build Failed</h2>
                </div>
                <p className="font-body text-[hsl(30,4%,70%)] mb-6">
                  The build process terminated with an error. Below is the error message captured from the build process.
                </p>
                <div className="bg-[hsl(0,0%,5.5%)] border border-[hsl(220,10%,28%)]/15 rounded-lg p-4 font-mono text-sm text-[hsl(30,4%,90%)] overflow-x-auto">
                  <div className="text-[hsl(0,84%,60%)]">{deployment.error_message}</div>
                </div>
              </div>
            </div>
          )}

          {/* Log Viewer */}
          <LogViewer
            lines={lines}
            isStreaming={isConnected}
          />
        </div>

        {/* Right Column: Timeline & Metadata */}
        <div className="space-y-6">
          {/* Build Timeline */}
          <div className="bg-[hsl(0,0%,11%)] rounded-xl p-6 border border-[hsl(220,10%,28%)]/10">
            <h3 className="font-headline font-bold text-lg mb-6">Build Timeline</h3>
            <div className="relative">
              <div className="absolute left-3 top-2 bottom-2 w-px bg-[hsl(220,10%,28%)]/20" />
              
              {/* Queued */}
              <div className="flex gap-4 mb-6 relative">
                <div className={`w-6 h-6 rounded-full bg-[hsl(0,0%,16%)] border flex items-center justify-center shrink-0 z-10 mt-0.5 ${
                  deployment.status !== "queued" ? "border-[hsl(220,100%,84%)]" : "border-[hsl(220,100%,84%)]"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    deployment.status !== "queued" ? "bg-[hsl(220,100%,84%)]" : "bg-[hsl(220,100%,84%)] animate-pulse"
                  }`} />
                </div>
                <div>
                  <div className="font-label font-bold text-[hsl(30,4%,90%)]">Queued</div>
                  <div className="text-xs text-[hsl(30,4%,70%)] font-mono mt-1">
                    {deployment.created_at ? new Date(deployment.created_at).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }) + " UTC" : "-"}
                  </div>
                </div>
              </div>

              {/* Started */}
              <div className="flex gap-4 mb-6 relative">
                <div className={`w-6 h-6 rounded-full bg-[hsl(0,0%,16%)] border flex items-center justify-center shrink-0 z-10 mt-0.5 ${
                  deployment.started_at ? (deployment.status === "failed" || deployment.status === "cancelled" ? "border-[hsl(220,10%,28%)]" : "border-[hsl(220,100%,84%)]") : "border-[hsl(220,10%,28%)]"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    deployment.started_at ? (deployment.status === "failed" || deployment.status === "cancelled" ? "bg-[hsl(220,10%,28%)]" : "bg-[hsl(220,100%,84%)]") : ""
                  }`} />
                </div>
                <div>
                  <div className={`font-label font-bold ${
                    deployment.started_at ? "text-[hsl(30,4%,90%)]" : "text-[hsl(30,4%,50%)]"
                  }`}>Started</div>
                  <div className="text-xs text-[hsl(30,4%,70%)] font-mono mt-1">
                    {deployment.started_at ? new Date(deployment.started_at).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }) + " UTC" : "-"}
                  </div>
                </div>
              </div>

              {/* Status (Complete/Failed/Cancelled) */}
              <div className="flex gap-4 relative">
                <div className={`w-6 h-6 rounded-full border flex items-center justify-center shrink-0 z-10 mt-0.5 ${
                  deployment.status === "ready" ? "bg-[hsl(142,76%,36%)]/20 border-[hsl(142,76%,56%)]" :
                  deployment.status === "failed" ? "bg-[hsl(0,84%,60%)]/20 border-[hsl(0,84%,60%)]" :
                  deployment.status === "cancelled" ? "bg-[hsl(0,0%,30%)] border-[hsl(0,0%,50%)]" :
                  "bg-[hsl(0,0%,16%)] border-[hsl(220,10%,28%)]"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    deployment.status === "ready" ? "bg-[hsl(142,76%,56%)]" :
                    deployment.status === "failed" ? "bg-[hsl(0,84%,60%)]" :
                    deployment.status === "cancelled" ? "bg-[hsl(0,0%,50%)]" :
                    ""
                  }`} style={
                    deployment.status === "failed" ? { boxShadow: "0 0 8px 1px rgba(255, 180, 171, 0.4)" } :
                    deployment.status === "ready" ? { boxShadow: "0 0 8px 1px rgba(74, 222, 128, 0.4)" } : undefined
                  } />
                </div>
                <div>
                  <div className={`font-label font-bold ${
                    deployment.status === "ready" ? "text-[hsl(142,76%,56%)]" :
                    deployment.status === "failed" ? "text-[hsl(0,84%,60%)]" :
                    deployment.status === "cancelled" ? "text-[hsl(0,0%,50%)]" :
                    "text-[hsl(30,4%,50%)]"
                  }`}>
                    {deployment.status === "ready" ? "Complete" :
                     deployment.status === "failed" ? "Failed" :
                     deployment.status === "cancelled" ? "Cancelled" :
                     deployment.status === "building" ? "Building..." :
                     deployment.status === "queued" ? "Waiting..." : deployment.status}
                  </div>
                  <div className="text-xs text-[hsl(30,4%,70%)] font-mono mt-1">
                    {deployment.completed_at ? new Date(deployment.completed_at).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }) + " UTC" : "-"}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Configuration Metadata */}
          <div className="bg-[hsl(0,0%,11%)] rounded-xl p-6 border border-[hsl(220,10%,28%)]/10">
            <h3 className="font-headline font-bold text-lg mb-4">Configuration</h3>
            <div className="space-y-4 font-label text-sm">
              <div className="flex justify-between items-center border-b border-[hsl(220,10%,28%)]/10 pb-2">
                <span className="text-[hsl(30,4%,70%)]">Status</span>
                <span className="text-[hsl(30,4%,90%)] capitalize">{deployment.status}</span>
              </div>
              {deployment.build_duration_ms && (
                <div className="flex justify-between items-center border-b border-[hsl(220,10%,28%)]/10 pb-2">
                  <span className="text-[hsl(30,4%,70%)]">Duration</span>
                  <span className="text-[hsl(30,4%,90%)] font-mono">
                    {Math.round(deployment.build_duration_ms / 1000)}s
                  </span>
                </div>
              )}
              {deployment.artifact_size_bytes && (
                <div className="flex justify-between items-center border-b border-[hsl(220,10%,28%)]/10 pb-2">
                  <span className="text-[hsl(30,4%,70%)]">Artifact Size</span>
                  <span className="text-[hsl(30,4%,90%)] font-mono">
                    {(deployment.artifact_size_bytes / 1024 / 1024).toFixed(1)} MB
                  </span>
                </div>
              )}
              <div className="flex justify-between items-center border-b border-[hsl(220,10%,28%)]/10 pb-2">
                <span className="text-[hsl(30,4%,70%)]">Branch</span>
                <span className="text-[hsl(30,4%,90%)] font-mono">{deployment.branch}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-[hsl(30,4%,70%)]">Production</span>
                <span className="text-[hsl(30,4%,90%)]">{deployment.is_production ? "Yes" : "No"}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
