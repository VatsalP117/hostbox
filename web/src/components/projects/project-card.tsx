import { Badge } from "@/components/ui/badge";
import { frameworkConfig } from "@/lib/constants";
import { timeAgo } from "@/lib/date";
import type { Project } from "@/types/models";
import { GitBranch, Folder } from "lucide-react";

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
      className="cursor-pointer rounded-xl border border-[#e5e2e1]/10 bg-[#201f1f] p-5 transition-colors hover:bg-[#2a2a2a]"
      onClick={onClick}
    >
      {/* Header: Framework Icon + Project Name */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          {/* Framework Icon */}
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[#131313] border border-[#e5e2e1]/10">
            {fw ? (
              <span className="text-[#ADC6FF] text-xs font-bold">
                {fw.label.slice(0, 2).toUpperCase()}
              </span>
            ) : (
              <Folder className="h-5 w-5 text-[#e5e2e1]/50" />
            )}
          </div>
          
          {/* Project Name */}
          <h3 className="font-['Manrope'] text-base font-bold text-[#e5e2e1] truncate">
            {project.name}
          </h3>
        </div>

        {/* Framework Label */}
        <div className="flex items-center gap-2">
          {project.status && (
            <Badge
              variant="outline"
              className={`text-[10px] uppercase tracking-wider ${statusStyles[project.status]}`}
            >
              {project.status}
            </Badge>
          )}
          {fw && (
            <span className="text-[10px] text-[#e5e2e1]/40 uppercase tracking-wider">
              {fw.label}
            </span>
          )}
        </div>
      </div>

      {/* Repo & Branch Info */}
      <div className="mt-4 space-y-2">
        {project.github_repo && (
          <div className="flex items-center gap-2 text-sm text-[#e5e2e1]/50">
            <GitBranch className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">{project.github_repo}</span>
          </div>
        )}

        <div className="flex items-center justify-between">
          <span className="text-xs text-[#e5e2e1]/50">
            Updated {timeAgo(project.updated_at)}
          </span>
          
          {project.production_branch && (
            <Badge
              variant="outline"
              className="text-[10px] border-[#e5e2e1]/20 text-[#e5e2e1]/70"
            >
              {project.production_branch}
            </Badge>
          )}
        </div>
      </div>
    </div>
  );
}
