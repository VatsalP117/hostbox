import { Badge } from "@/components/ui/badge";
import { frameworkConfig } from "@/lib/constants";
import type { Framework } from "@/types/models";
import {
  Hexagon,
  Zap,
  Atom,
  Rocket,
  Circle,
  Triangle,
  Flame,
  FileText,
  Globe,
  HelpCircle,
} from "lucide-react";

const iconMap: Record<string, React.ElementType> = {
  Hexagon,
  Zap,
  Atom,
  Rocket,
  Circle,
  Triangle,
  Flame,
  FileText,
  Globe,
  HelpCircle,
};

interface FrameworkBadgeProps {
  framework: Framework | null;
}

export function FrameworkBadge({ framework }: FrameworkBadgeProps) {
  if (!framework) return null;

  const config = frameworkConfig[framework];
  const Icon = iconMap[config.icon] ?? HelpCircle;

  return (
    <Badge variant="secondary" className="gap-1">
      <Icon className="h-3 w-3" />
      {config.label}
    </Badge>
  );
}
