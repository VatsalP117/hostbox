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
    <div className="bg-[hsl(0,0%,11%)] rounded-xl border border-[hsl(220,10%,28%)]/10 overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[hsl(220,10%,28%)]/10 flex items-center justify-between">
        <h3 className="font-headline font-bold text-lg">Build Progress</h3>
        <span className="text-sm text-[hsl(30,4%,70%)] font-label">
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
          <div className="h-2 bg-[hsl(0,0%,16%)] rounded-full overflow-hidden">
            <div 
              className="h-full bg-gradient-to-r from-[hsl(220,100%,84%)] to-[hsl(217,91%,65%)] transition-all duration-500 ease-out"
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
                    state === "done" && "bg-[hsl(142,76%,36%)]/20 border border-[hsl(142,76%,56%)]",
                    state === "active" && "bg-[hsl(220,100%,84%)]/20 border border-[hsl(220,100%,84%)]",
                    state === "failed" && "bg-[hsl(0,84%,60%)]/20 border border-[hsl(0,84%,60%)]",
                    state === "pending" && "bg-[hsl(0,0%,16%)] border border-[hsl(220,10%,28%)]"
                  )}>
                    {state === "done" && (
                      <Check className="h-4 w-4 text-[hsl(142,76%,56%)]" />
                    )}
                    {state === "active" && (
                      <Loader2 className="h-4 w-4 text-[hsl(220,100%,84%)] animate-spin" />
                    )}
                    {state === "failed" && (
                      <XCircle className="h-4 w-4 text-[hsl(0,84%,60%)]" />
                    )}
                    {state === "pending" && (
                      <Circle className="h-4 w-4 text-[hsl(220,10%,28%)]" />
                    )}
                  </div>
                  
                  {/* Phase Label */}
                  <span className={cn(
                    "text-xs font-label transition-colors duration-300",
                    state === "done" && "text-[hsl(142,76%,56%)]",
                    state === "active" && "text-[hsl(220,100%,84%)] font-medium",
                    state === "failed" && "text-[hsl(0,84%,60%)]",
                    state === "pending" && "text-[hsl(30,4%,50%)]"
                  )}>
                    {p.label}
                  </span>
                </div>
                
                {/* Connector Line */}
                {!isLast && (
                  <div className={cn(
                    "w-8 h-px mx-2 transition-colors duration-300",
                    state === "done" ? "bg-[hsl(142,76%,56%)]" : "bg-[hsl(220,10%,28%)]/30"
                  )} />
                )}
              </div>
            );
          })}
        </div>

        {/* Time Elapsed */}
        <div className="flex items-center justify-between text-sm text-[hsl(30,4%,70%)] font-label pt-2 border-t border-[hsl(220,10%,28%)]/10">
          <span>Time Elapsed</span>
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
