import { X, AlertTriangle, AlertCircle, Info } from "lucide-react";
import { useEffect, useState } from "react";
import type { SystemAlert } from "@/types/models";

interface SystemAlertsProps {
  alerts?: SystemAlert[] | null;
}

const dismissedAlertsStorageKey = "hostbox-dismissed-system-alerts";

export function SystemAlerts({ alerts }: SystemAlertsProps) {
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (typeof window === "undefined") return;

    const stored = window.localStorage.getItem(dismissedAlertsStorageKey);
    if (!stored) return;

    try {
      const parsed = JSON.parse(stored);
      if (Array.isArray(parsed)) {
        setDismissed(new Set(parsed));
      }
    } catch {
      window.localStorage.removeItem(dismissedAlertsStorageKey);
    }
  }, []);

  if (!alerts?.length) return null;

  const visibleAlerts = alerts.filter((alert) => !dismissed.has(`${alert.severity}-${alert.title}`));

  if (visibleAlerts.length === 0) return null;

  const handleDismiss = (key: string) => {
    setDismissed((prev) => {
      const next = new Set(prev).add(key);
      if (typeof window !== "undefined") {
        window.localStorage.setItem(dismissedAlertsStorageKey, JSON.stringify(Array.from(next)));
      }
      return next;
    });
  };

  return (
    <div className="space-y-3">
      {visibleAlerts.map((alert) => {
        const alertKey = `${alert.severity}-${alert.title}`;
        const isError = alert.severity === "error";
        const isWarning = alert.severity === "warning";
        
        const Icon = isError ? AlertCircle : isWarning ? AlertTriangle : Info;
        const statusColor = isError 
          ? "bg-destructive shadow-[0_0_12px_rgba(239,68,68,0.3)]" 
          : isWarning 
            ? "bg-warning shadow-[0_0_12px_rgba(245,158,11,0.3)]" 
            : "bg-primary shadow-[0_0_12px_rgba(173,198,255,0.3)]";

        return (
          <div
            key={alertKey}
            className={`flex items-center justify-between p-4 rounded-xl bg-surface-container-low border-l-4 ${
              isError ? "border-l-destructive" : isWarning ? "border-l-warning" : "border-l-primary"
            } hover:bg-surface-container transition-colors group`}
          >
            <div className="flex items-center space-x-4">
              <div className={`w-2 h-2 rounded-full ${statusColor}`} />
              <div>
                <div className="flex items-center space-x-2">
                  <Icon className={`h-4 w-4 ${isError ? "text-destructive" : isWarning ? "text-warning" : "text-primary"}`} />
                  <h4 className={`font-body text-sm font-medium text-foreground group-hover:text-primary transition-colors`}>
                    {alert.title}
                  </h4>
                </div>
                <p className="font-label text-xs text-muted-foreground mt-0.5">{alert.message}</p>
              </div>
            </div>
            <button
              onClick={() => handleDismiss(alertKey)}
              className="p-1 rounded-md hover:bg-surface-container-high text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Dismiss alert"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
