import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { routes, queryKeys } from "@/lib/constants";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { ProjectCard } from "@/components/projects/project-card";
import { PaginationControls } from "@/components/shared/pagination-controls";
import type { ProjectListResponse } from "@/types/api";
import { FolderPlus, Plus, Search } from "lucide-react";

export function ProjectsPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");

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

  return (
    <div className="space-y-6">
      <PageHeader title="Projects" description="Manage your deployed projects.">
        <Button onClick={() => navigate(routes.newProject)}>
          <Plus className="mr-2 h-4 w-4" />
          New Project
        </Button>
      </PageHeader>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search projects..."
          value={search}
          onChange={(e) => handleSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 rounded-lg" />
          ))}
        </div>
      ) : !data?.projects?.length ? (
        <EmptyState
          icon={FolderPlus}
          title="No projects yet"
          description="Create your first project to start deploying."
          action={{
            label: "Create your first project",
            onClick: () => navigate(routes.newProject),
          }}
        />
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {data.projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onClick={() => navigate(routes.project(project.id))}
              />
            ))}
          </div>
          {data.pagination && (
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
