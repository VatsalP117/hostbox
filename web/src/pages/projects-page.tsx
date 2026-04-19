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
import { FolderPlus, Search, ChevronDown } from "lucide-react";

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
    <div className="space-y-8">
      {/* Page Header */}
      <div className="space-y-2">
        <h1 className="font-['Manrope'] text-4xl font-extrabold tracking-tight text-[#e5e2e1] md:text-5xl">
          Projects
        </h1>
        <p className="max-w-2xl text-[#e5e2e1]/50">
          Manage your sovereign deployments. Monitor status, branches, and routing rules across all hosted applications.
        </p>
      </div>

      {/* Search and Filters */}
      <div className="rounded-xl bg-[#201f1f]/50 p-4 space-y-4">
        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-[#e5e2e1]/50" />
          <Input
            placeholder="Search projects..."
            value={search}
            onChange={(e) => handleSearch(e.target.value)}
            className="border-[#e5e2e1]/10 bg-transparent pl-12 text-[#e5e2e1] placeholder:text-[#e5e2e1]/50 focus:border-[#ADC6FF]/30"
          />
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-3">
          <div className="relative">
            <select
              value={frameworkFilter}
              onChange={(e) => setFrameworkFilter(e.target.value)}
              className="appearance-none rounded-lg border border-[#e5e2e1]/30 bg-[#201f1f] px-4 py-2 pr-10 text-sm font-medium text-[#e5e2e1] focus:border-[#ADC6FF] focus:outline-none"
            >
              <option value="all">All Frameworks</option>
              {Object.entries(frameworkConfig).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
            <ChevronDown className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#e5e2e1]/50 pointer-events-none" />
          </div>
          <div className="relative">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="appearance-none rounded-lg border border-[#e5e2e1]/30 bg-[#201f1f] px-4 py-2 pr-10 text-sm font-medium text-[#e5e2e1] focus:border-[#ADC6FF] focus:outline-none"
            >
              <option value="all">All Statuses</option>
              <option value="healthy">Healthy</option>
              <option value="building">Building</option>
              <option value="failed">Failed</option>
              <option value="stopped">Not Deployed</option>
            </select>
            <ChevronDown className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#e5e2e1]/50 pointer-events-none" />
          </div>
        </div>
      </div>

      {/* Project Grid */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 rounded-xl bg-[#201f1f]" />
          ))}
        </div>
      ) : !filteredProjects?.length ? (
        <div className="rounded-xl bg-[#201f1f]/50 p-8">
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
