import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  ProjectListParams,
  ProjectListResponse,
  ProjectDetailResponse,
  CreateProjectRequest,
  UpdateProjectRequest,
} from "@/types/api";
import type { Project } from "@/types/models";

export function useProjects(params?: ProjectListParams) {
  return useQuery({
    queryKey: queryKeys.projects(params),
    queryFn: () =>
      api.get<ProjectListResponse>("/projects", {
        page: params?.page,
        per_page: params?.per_page,
        search: params?.search,
      }),
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: queryKeys.project(id),
    queryFn: () => api.get<ProjectDetailResponse>(`/projects/${id}`),
    enabled: !!id,
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateProjectRequest) =>
      api.post<{ project: Project }>("/projects", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });
}

export function useUpdateProject(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateProjectRequest) =>
      api.patch<{ project: Project }>(`/projects/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.project(id) });
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });
}

export function useDeleteProject(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.delete<void>(`/projects/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });
}
