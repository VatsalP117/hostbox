import { cn } from "@/lib/utils";
import { Check, Loader2, Circle } from "lucide-react";
import type { DeploymentStatus } from "@/types/models";

const steps = [
  { key: "clone", label: "Clone" },
  { key: "install", label: "Install" },
  { key: "build", label: "Build" },
  { key: "deploy", label: "Deploy" },
] as const;

type Phase = (typeof steps)[number]["key"];

function getStepState(
  stepIndex: number,
  currentPhase: string | null,
  status: DeploymentStatus,
): "done" | "active" | "pending" | "failed" {
  if (status === "failed") {
    const phaseIndex = steps.findIndex((s) => s.key === currentPhase);
    if (stepIndex < phaseIndex) return "done";
    if (stepIndex === phaseIndex) return "failed";
    return "pending";
  }

  if (status === "ready") return "done";
  if (status === "cancelled") {
    const phaseIndex = steps.findIndex((s) => s.key === currentPhase);
    if (stepIndex < phaseIndex) return "done";
    return "pending";
  }

  const phaseIndex = steps.findIndex((s) => s.key === currentPhase);
  if (stepIndex < phaseIndex) return "done";
  if (stepIndex === phaseIndex) return "active";
  return "pending";
}

interface BuildProgressProps {
  phase: string | null;
  status: DeploymentStatus;
}

export function BuildProgress({ phase, status }: BuildProgressProps) {
  return (
    <div className="flex items-center gap-1">
      {steps.map((step, i) => {
        const state = getStepState(i, phase, status);
        return (
          <div key={step.key} className="flex items-center gap-1">
            {i > 0 && (
              <div
                className={cn(
                  "h-px w-6",
                  state === "done" ? "bg-green-500" : "bg-border",
                )}
              />
            )}
            <div className="flex items-center gap-1.5">
              {state === "done" && (
                <Check className="h-4 w-4 text-green-500" />
              )}
              {state === "active" && (
                <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />
              )}
              {state === "pending" && (
                <Circle className="h-4 w-4 text-muted-foreground" />
              )}
              {state === "failed" && (
                <Circle className="h-4 w-4 text-red-500" />
              )}
              <span
                className={cn(
                  "text-xs",
                  state === "active" && "font-medium text-blue-500",
                  state === "done" && "text-green-600",
                  state === "failed" && "text-red-500",
                  state === "pending" && "text-muted-foreground",
                )}
              >
                {step.label}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}
