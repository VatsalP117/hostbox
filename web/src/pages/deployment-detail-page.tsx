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

  const deploymentStatus = deployment?.status ?? "queued";

  const {
    lines,
    status: phase,
    isConnected,
    complete,
    isLoadingHistory,
  } = useDeploymentLogs(
    deploymentId!,
    {
      enabled: !!deployment,
      liveEnabled:
        deploymentStatus === "queued" || deploymentStatus === "building",
    },
  );

  const effectiveStatus = complete?.status ?? deploymentStatus;
  const isActive =
    effectiveStatus === "queued" || effectiveStatus === "building";

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-48 w-full rounded-lg" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <Skeleton className="h-[500px] w-full rounded-lg lg:col-span-2" />
          <Skeleton className="h-[300px] w-full rounded-lg" />
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
        className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-card hover:bg-muted transition-colors text-muted-foreground hover:text-foreground text-sm font-sans"
      >
        <ChevronLeft className="h-4 w-4" />
        Back to project
      </button>

      {/* Deployment Header */}
      <DeploymentHeader deployment={deployment} />

      {/* Build Progress (only for active deployments) */}
      {isActive && (
        <BuildProgress phase={phase} status={effectiveStatus} />
      )}

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column: Error Panel & Logs */}
        <div className="lg:col-span-2 space-y-6">
          {/* Error Panel (only for failed deployments) */}
          {effectiveStatus === "failed" && deployment.error_message && (
            <div className="relative overflow-hidden rounded-lg border border-destructive/30 bg-card">
              <div className="p-6">
                <div className="flex items-center gap-3 mb-4 text-destructive">
                  <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
                  </svg>
                  <h2 className="text-lg font-medium tracking-tight">Build failed</h2>
                </div>
                <p className="font-body text-muted-foreground mb-6">
                  The build process terminated with an error. Below is the error message captured from the build process.
                </p>
                <div className="bg-black border border-border rounded-lg p-4 font-mono text-sm text-foreground overflow-x-auto">
                  <div className="text-destructive">{deployment.error_message}</div>
                </div>
              </div>
            </div>
          )}

          {/* Log Viewer */}
          <LogViewer
            lines={lines}
            isStreaming={isConnected}
            isLoading={isLoadingHistory && lines.length === 0}
            emptyMessage={
              isActive
                ? isConnected
                  ? "Waiting for logs..."
                  : "Connecting to live logs..."
                : "No logs available for this deployment."
            }
          />
        </div>

        {/* Right Column: Timeline & Metadata */}
        <div className="space-y-6">
          {/* Build Timeline */}
          <div className="bg-card rounded-lg p-6 border border-border">
            <h3 className="mb-6 text-base font-medium">Build timeline</h3>
            <div className="relative">
              <div className="absolute bottom-2 left-3 top-2 w-px bg-border" />
              
              {/* Queued */}
              <div className="flex gap-4 mb-6 relative">
                <div className={`w-6 h-6 rounded-full bg-muted border flex items-center justify-center shrink-0 z-10 mt-0.5 ${
                  "border-[#52a8ff]"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    deployment.status !== "queued" ? "bg-[#52a8ff]" : "bg-[#52a8ff] animate-pulse"
                  }`} />
                </div>
                <div>
                  <div className="font-sans font-bold text-foreground">Queued</div>
                  <div className="text-xs text-muted-foreground font-mono mt-1">
                    {deployment.created_at ? new Date(deployment.created_at).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }) + " UTC" : "-"}
                  </div>
                </div>
              </div>

              {/* Started */}
              <div className="flex gap-4 mb-6 relative">
                <div className={`w-6 h-6 rounded-full bg-muted border flex items-center justify-center shrink-0 z-10 mt-0.5 ${
                  deployment.started_at ? (deployment.status === "failed" || deployment.status === "cancelled" ? "border-border" : "border-[#52a8ff]") : "border-border"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    deployment.started_at ? (deployment.status === "failed" || deployment.status === "cancelled" ? "bg-border" : "bg-[#52a8ff]") : ""
                  }`} />
                </div>
                <div>
                  <div className={`font-sans font-bold ${
                    deployment.started_at ? "text-foreground" : "text-muted-foreground"
                  }`}>Started</div>
                  <div className="text-xs text-muted-foreground font-mono mt-1">
                    {deployment.started_at ? new Date(deployment.started_at).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }) + " UTC" : "-"}
                  </div>
                </div>
              </div>

              {/* Status (Complete/Failed/Cancelled) */}
              <div className="flex gap-4 relative">
                <div className={`w-6 h-6 rounded-full border flex items-center justify-center shrink-0 z-10 mt-0.5 ${
                  deployment.status === "ready" ? "bg-success/10 border-success" :
                  deployment.status === "failed" ? "bg-destructive/10 border-destructive" :
                  deployment.status === "cancelled" ? "bg-muted border-muted-foreground" :
                  "bg-muted border-border"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    deployment.status === "ready" ? "bg-success" :
                    deployment.status === "failed" ? "bg-destructive" :
                    deployment.status === "cancelled" ? "bg-muted-foreground" :
                    ""
                  }`} />
                </div>
                <div>
                  <div className={`font-sans font-bold ${
                    effectiveStatus === "ready" ? "text-success" :
                    effectiveStatus === "failed" ? "text-destructive" :
                    effectiveStatus === "cancelled" ? "text-muted-foreground" :
                    "text-muted-foreground"
                  }`}>
                    {effectiveStatus === "ready" ? "Complete" :
                     effectiveStatus === "failed" ? "Failed" :
                     effectiveStatus === "cancelled" ? "Cancelled" :
                     effectiveStatus === "building" ? "Building..." :
                     effectiveStatus === "queued" ? "Waiting..." : effectiveStatus}
                  </div>
                  <div className="text-xs text-muted-foreground font-mono mt-1">
                    {deployment.completed_at ? new Date(deployment.completed_at).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }) + " UTC" : "-"}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Configuration Metadata */}
          <div className="bg-card rounded-lg p-6 border border-border">
            <h3 className="mb-4 text-base font-medium">Configuration</h3>
            <div className="space-y-4 font-sans text-sm">
              <div className="flex justify-between items-center border-b border-border pb-2">
                <span className="text-muted-foreground">Status</span>
                <span className="text-foreground capitalize">{effectiveStatus}</span>
              </div>
              {deployment.build_duration_ms && (
                <div className="flex justify-between items-center border-b border-border pb-2">
                  <span className="text-muted-foreground">Duration</span>
                  <span className="text-foreground font-mono">
                    {Math.round(deployment.build_duration_ms / 1000)}s
                  </span>
                </div>
              )}
              {deployment.artifact_size_bytes && (
                <div className="flex justify-between items-center border-b border-border pb-2">
                  <span className="text-muted-foreground">Artifact size</span>
                  <span className="text-foreground font-mono">
                    {(deployment.artifact_size_bytes / 1024 / 1024).toFixed(1)} MB
                  </span>
                </div>
              )}
              <div className="flex justify-between items-center border-b border-border pb-2">
                <span className="text-muted-foreground">Branch</span>
                <span className="text-foreground font-mono">{deployment.branch}</span>
              </div>
              {deployment.source_repository && (
                <div className="flex justify-between items-center border-b border-border pb-2">
                  <span className="text-muted-foreground">Repository</span>
                  <span className="text-foreground font-mono">
                    {deployment.source_repository}
                  </span>
                </div>
              )}
              <div className="flex justify-between items-center border-b border-border pb-2">
                <span className="text-muted-foreground">Build recipe</span>
                <span className="text-foreground">
                  {deployment.build_manifest_resolved ? "Resolved" : "Pending detection"}
                </span>
              </div>
              {deployment.build_framework && (
                <div className="flex justify-between items-center border-b border-border pb-2">
                  <span className="text-muted-foreground">Framework</span>
                  <span className="text-foreground font-mono">
                    {deployment.build_framework} · {deployment.build_serving_mode}
                  </span>
                </div>
              )}
              {deployment.build_package_manager && (
                <div className="flex justify-between items-center border-b border-border pb-2">
                  <span className="text-muted-foreground">Package manager</span>
                  <span className="text-foreground font-mono">
                    {deployment.build_package_manager}
                    {deployment.build_package_manager_version
                      ? `@${deployment.build_package_manager_version}`
                      : ""}
                  </span>
                </div>
              )}
              <div className="flex justify-between items-center border-b border-border pb-2">
                <span className="text-muted-foreground">Node / root</span>
                <span className="text-foreground font-mono">
                  Node {deployment.build_node_version || "detect"} · {deployment.build_root_directory}
                </span>
              </div>
              {deployment.build_command && (
                <div className="border-b border-border pb-2">
                  <div className="mb-1 text-muted-foreground">Build command</div>
                  <div className="text-foreground font-mono break-all">
                    {deployment.build_command}
                  </div>
                </div>
              )}
              {deployment.build_output_directory && (
                <div className="flex justify-between items-center border-b border-border pb-2">
                  <span className="text-muted-foreground">Output</span>
                  <span className="text-foreground font-mono">
                    {deployment.build_output_directory}
                  </span>
                </div>
              )}
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground">Production</span>
                <span className="text-foreground">{deployment.is_production ? "Yes" : "No"}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
