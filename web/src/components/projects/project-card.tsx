import { Badge } from "@/components/ui/badge";
import { frameworkConfig } from "@/lib/constants";
import { timeAgo } from "@/lib/date";
import type { Project } from "@/types/models";
import { GitBranch, Folder, ArrowUpRight } from "lucide-react";

const statusStyles = {
  healthy: "bg-primary/10 text-primary border-primary/30",
  building: "bg-warning/10 text-warning border-warning/30",
  failed: "bg-destructive/10 text-destructive border-destructive/30",
  stopped: "bg-surface-container-high text-muted-foreground border-outline-variant/30",
} as const;

interface ProjectCardProps {
  project: Project;
  onClick: () => void;
}

export function ProjectCard({ project, onClick }: ProjectCardProps) {
  const fw = project.framework
    ? frameworkConfig[project.framework]
    : null;

  return (
    <div
      className="group cursor-pointer rounded-lg border border-border bg-card p-5 transition-[background-color,border-color] hover:border-foreground/20 hover:bg-[#141414]"
      onClick={onClick}
    >
      {/* Header: Framework Icon + Project Name */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          {/* Framework Icon */}
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-black">
            {fw ? (
              <span className="text-xs font-semibold text-foreground">
                {fw.label.slice(0, 2).toUpperCase()}
              </span>
            ) : (
              <Folder className="h-4 w-4 text-muted-foreground" />
            )}
          </div>
          
          {/* Project Name */}
          <h3 className="truncate text-sm font-medium text-foreground">
            {project.name}
          </h3>
        </div>

        {/* Framework Label */}
        <div className="flex items-center gap-2">
          {project.status && (
            <Badge
              variant="outline"
              className={`text-[10px] capitalize ${statusStyles[project.status]}`}
            >
              {project.status}
            </Badge>
          )}
          {fw && (
            <span className="text-xs text-muted-foreground">
              {fw.label}
            </span>
          )}
        </div>
      </div>

      {/* Repo & Branch Info */}
      <div className="mt-6 space-y-3">
        {project.github_repo && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <GitBranch className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">{project.github_repo}</span>
          </div>
        )}

        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">
            Updated {timeAgo(project.updated_at)}
          </span>
          
          {project.production_branch && (
            <Badge
              variant="outline"
              className="border-border text-[10px] text-muted-foreground"
            >
              {project.production_branch}
            </Badge>
          )}
        </div>
      </div>
      <div className="mt-4 flex items-center justify-end border-t border-border pt-3 text-xs text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100">
        Open project <ArrowUpRight className="ml-1 h-3.5 w-3.5" />
      </div>
    </div>
  );
}
