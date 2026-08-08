import { cn } from "@/lib/utils";
import { Check, Loader2, Circle, XCircle } from "lucide-react";
import type { DeploymentStatus } from "@/types/models";

const phases = [
  { key: "queued", label: "Queued" },
  { key: "clone", label: "Clone" },
  { key: "install", label: "Install" },
  { key: "build", label: "Build" },
  { key: "deploy", label: "Deploy" },
  { key: "complete", label: "Complete" },
] as const;

const runtimePhases = new Set(["clone", "install", "build", "deploy"]);

function normalizePhase(
  phase: string | null,
  status: DeploymentStatus,
): (typeof phases)[number]["key"] {
  if (status === "ready") return "complete";
  if (phase && runtimePhases.has(phase)) return phase as "clone" | "install" | "build" | "deploy";
  if (status === "queued") return "queued";
  return "build";
}

function getPhaseState(
  phaseIndex: number,
  currentPhase: string | null,
  status: DeploymentStatus,
): "done" | "active" | "pending" | "failed" {
  const currentIndex = phases.findIndex((p) => p.key === (currentPhase || "queued"));
  
  if (status === "failed") {
    if (phaseIndex < currentIndex) return "done";
    if (phaseIndex === currentIndex) return "failed";
    return "pending";
  }

  if (status === "ready") return "done";
  
  if (status === "cancelled") {
    if (phaseIndex < currentIndex) return "done";
    return "pending";
  }

  if (phaseIndex < currentIndex) return "done";
  if (phaseIndex === currentIndex) return "active";
  return "pending";
}

interface BuildProgressProps {
  phase: string | null;
  status: DeploymentStatus;
}

export function BuildProgress({ phase, status }: BuildProgressProps) {
  const normalizedPhase = normalizePhase(phase, status);
  const currentPhaseIndex = phases.findIndex((p) => p.key === normalizedPhase);
  const progress = Math.min(
    ((currentPhaseIndex + 1) / phases.length) * 100,
    100
  );

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      {/* Header */}
      <div className="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 className="text-base font-medium">Build progress</h3>
        <span className="text-sm text-muted-foreground font-sans">
          {status === "building"
            ? "Building..."
            : status === "queued"
              ? "Queued"
              : status === "failed"
                ? "Failed"
                : status === "cancelled"
                  ? "Cancelled"
                  : "Complete"}
        </span>
      </div>

      <div className="p-6 space-y-6">
        {/* Progress Bar */}
        <div className="relative">
          <div className="h-1.5 overflow-hidden rounded-full bg-muted">
            <div 
              className="h-full bg-[#52a8ff] transition-all duration-500 ease-out"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        {/* Phase Indicators */}
        <div className="flex items-center justify-between">
          {phases.map((p, i) => {
            const state = getPhaseState(i, normalizedPhase, status);
            const isLast = i === phases.length - 1;
            
            return (
              <div key={p.key} className="flex items-center">
                <div className="flex flex-col items-center gap-2">
                  {/* Phase Circle */}
                  <div className={cn(
                    "w-8 h-8 rounded-full flex items-center justify-center transition-all duration-300",
                    state === "done" && "border border-success bg-success/10",
                    state === "active" && "border border-[#52a8ff] bg-[#52a8ff]/10",
                    state === "failed" && "border border-destructive bg-destructive/10",
                    state === "pending" && "bg-muted border border-border"
                  )}>
                    {state === "done" && (
                      <Check className="h-4 w-4 text-success" />
                    )}
                    {state === "active" && (
                      <Loader2 className="h-4 w-4 animate-spin text-[#52a8ff]" />
                    )}
                    {state === "failed" && (
                      <XCircle className="h-4 w-4 text-destructive" />
                    )}
                    {state === "pending" && (
                      <Circle className="h-4 w-4 text-neutral-700" />
                    )}
                  </div>
                  
                  {/* Phase Label */}
                  <span className={cn(
                    "text-xs font-sans transition-colors duration-300",
                    state === "done" && "text-success",
                    state === "active" && "font-medium text-[#52a8ff]",
                    state === "failed" && "text-destructive",
                    state === "pending" && "text-muted-foreground"
                  )}>
                    {p.label}
                  </span>
                </div>
                
                {/* Connector Line */}
                {!isLast && (
                  <div className={cn(
                    "w-8 h-px mx-2 transition-colors duration-300",
                    state === "done" ? "bg-success" : "bg-border"
                  )} />
                )}
              </div>
            );
          })}
        </div>

        {/* Time Elapsed */}
        <div className="flex items-center justify-between text-sm text-muted-foreground font-sans pt-2 border-t border-border">
          <span>Time elapsed</span>
          <span className="font-mono">
            {status === "queued"
              ? "Waiting..."
              : status === "ready"
                ? "Completed"
                : status === "failed"
                  ? "Stopped"
                  : status === "cancelled"
                    ? "Cancelled"
                    : "In progress..."}
          </span>
        </div>
      </div>
    </div>
  );
}
