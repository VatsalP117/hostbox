import { Siren, TriangleAlert } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import type { SystemAlert } from "@/types/models";

interface SystemAlertsProps {
  alerts?: SystemAlert[] | null;
}

export function SystemAlerts({ alerts }: SystemAlertsProps) {
  if (!alerts?.length) return null;

  return (
    <div className="space-y-3">
      {alerts.map((alert) => {
        const Icon = alert.severity === "error" ? Siren : TriangleAlert;
        return (
          <Alert
            key={`${alert.severity}-${alert.title}`}
            variant={alert.severity === "error" ? "destructive" : "default"}
          >
            <Icon className="h-4 w-4" />
            <AlertTitle>{alert.title}</AlertTitle>
            <AlertDescription>{alert.message}</AlertDescription>
          </Alert>
        );
      })}
    </div>
  );
}
