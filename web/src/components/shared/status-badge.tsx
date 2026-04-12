import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { statusConfig } from "@/lib/constants";
import type { DeploymentStatus } from "@/types/models";

interface StatusBadgeProps {
  status: DeploymentStatus;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const config = statusConfig[status];

  return (
    <Badge variant="outline" className={cn("gap-1.5", config.className)}>
      <span className={cn("h-1.5 w-1.5 rounded-full", config.dotClassName)} />
      {config.label}
    </Badge>
  );
}
