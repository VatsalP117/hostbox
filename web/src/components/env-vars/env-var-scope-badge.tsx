import { Badge } from "@/components/ui/badge";
import type { EnvVarScope } from "@/types/models";

const scopeStyles: Record<EnvVarScope, { label: string; className: string }> = {
  all: {
    label: "All",
    className: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  },
  production: {
    label: "Production",
    className:
      "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  },
  preview: {
    label: "Preview",
    className:
      "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400",
  },
};

interface EnvVarScopeBadgeProps {
  scope: EnvVarScope;
}

export function EnvVarScopeBadge({ scope }: EnvVarScopeBadgeProps) {
  const config = scopeStyles[scope];
  return (
    <Badge variant="outline" className={config.className}>
      {config.label}
    </Badge>
  );
}
