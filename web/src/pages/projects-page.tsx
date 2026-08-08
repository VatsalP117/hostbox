import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { routes, queryKeys, frameworkConfig } from "@/lib/constants";
import { EmptyState } from "@/components/shared/empty-state";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { ProjectCard } from "@/components/projects/project-card";
import { PaginationControls } from "@/components/shared/pagination-controls";
import type { ProjectListResponse } from "@/types/api";
import { FolderPlus, Search, ChevronDown, Plus } from "lucide-react";

export function ProjectsPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [frameworkFilter, setFrameworkFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");

  // Simple debounce
  const searchTimeout = useState<ReturnType<typeof setTimeout> | null>(null);
  const handleSearch = (value: string) => {
    setSearch(value);
    if (searchTimeout[0]) clearTimeout(searchTimeout[0]);
    searchTimeout[1](
      setTimeout(() => {
        setDebouncedSearch(value);
        setPage(1);
      }, 300),
    );
  };

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.projects({ page, search: debouncedSearch }),
    queryFn: () =>
      api.get<ProjectListResponse>("/projects", {
        page,
        per_page: 12,
        search: debouncedSearch || undefined,
      }),
  });

  const filteredProjects = data?.projects?.filter((project) => {
    const matchesFramework = frameworkFilter === "all" || project.framework === frameworkFilter;
    const matchesStatus = statusFilter === "all" || project.status === statusFilter;
    return matchesFramework && matchesStatus;
  });

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">Projects</h1>
          <p className="mt-1 text-sm text-muted-foreground">Deploy and manage applications running on your infrastructure.</p>
        </div>
        <button onClick={() => navigate(routes.newProject)} className="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-white bg-white px-4 text-sm font-medium text-black hover:bg-neutral-200">
          <Plus className="h-4 w-4" /> New project
        </button>
      </div>

      {/* Search and Filters */}
      <div className="flex flex-col gap-3 border-y border-border py-4 lg:flex-row lg:items-center">
        {/* Search Bar */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search projects..."
            value={search}
            onChange={(e) => handleSearch(e.target.value)}
            className="bg-black pl-9"
          />
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-2">
          <div className="relative">
            <select
              value={frameworkFilter}
              onChange={(e) => setFrameworkFilter(e.target.value)}
              className="h-9 appearance-none rounded-md border border-input bg-black px-3 pr-9 text-sm text-foreground focus:border-ring focus:outline-none"
            >
              <option value="all">All Frameworks</option>
              {Object.entries(frameworkConfig).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
            <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          </div>
          <div className="relative">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="h-9 appearance-none rounded-md border border-input bg-black px-3 pr-9 text-sm text-foreground focus:border-ring focus:outline-none"
            >
              <option value="all">All Statuses</option>
              <option value="healthy">Healthy</option>
              <option value="building">Building</option>
              <option value="failed">Failed</option>
              <option value="stopped">Not Deployed</option>
            </select>
            <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          </div>
        </div>
      </div>

      {/* Project Grid */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-44 rounded-lg bg-card" />
          ))}
        </div>
      ) : !filteredProjects?.length ? (
        <div className="rounded-lg border border-dashed border-border bg-card/50 p-8">
          <EmptyState
            icon={FolderPlus}
            title="No projects found"
            description={
                search || frameworkFilter !== "all" || statusFilter !== "all"
                  ? "Try adjusting your search or filters."
                  : "Create your first project to start deploying."
            }
            action={
                !search && frameworkFilter === "all" && statusFilter === "all"
                  ? {
                    label: "Create your first project",
                    onClick: () => navigate(routes.newProject),
                  }
                : undefined
            }
          />
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filteredProjects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onClick={() => navigate(routes.project(project.id))}
              />
            ))}
          </div>
          {data?.pagination && (
            <PaginationControls
              pagination={data.pagination}
              onPageChange={setPage}
            />
          )}
        </>
      )}
    </div>
  );
}
