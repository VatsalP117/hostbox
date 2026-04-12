import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { frameworkConfig } from "@/lib/constants";
import { timeAgo } from "@/lib/date";
import type { Project } from "@/types/models";
import { GitBranch, ExternalLink } from "lucide-react";

interface ProjectCardProps {
  project: Project;
  onClick: () => void;
}

export function ProjectCard({ project, onClick }: ProjectCardProps) {
  const fw = project.framework
    ? frameworkConfig[project.framework]
    : null;

  return (
    <Card
      className="cursor-pointer transition-colors hover:bg-accent/50"
      onClick={onClick}
    >
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <CardTitle className="text-base">{project.name}</CardTitle>
          {fw && (
            <Badge variant="secondary" className="text-xs">
              {fw.label}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {project.github_repo && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <GitBranch className="h-3 w-3" />
            <span>{project.github_repo}</span>
          </div>
        )}
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>{timeAgo(project.updated_at)}</span>
          {project.production_branch && (
            <Badge variant="outline" className="text-[10px]">
              {project.production_branch}
            </Badge>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
