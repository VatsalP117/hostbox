import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  EnvVarListResponse,
  CreateEnvVarRequest,
  UpdateEnvVarRequest,
  BulkImportEnvVarRequest,
  BulkImportEnvVarResponse,
} from "@/types/api";
import type { EnvVar } from "@/types/models";

export function useEnvVars(projectId: string) {
  return useQuery({
    queryKey: queryKeys.envVars(projectId),
    queryFn: () =>
      api.get<EnvVarListResponse>(`/projects/${projectId}/env-vars`),
    enabled: !!projectId,
  });
}

export function useCreateEnvVar(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateEnvVarRequest) =>
      api.post<{ env_var: EnvVar }>(`/projects/${projectId}/env-vars`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.envVars(projectId),
      });
    },
  });
}

export function useUpdateEnvVar() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      projectId: string;
      data: UpdateEnvVarRequest;
    }) => api.patch<{ env_var: EnvVar }>(`/env-vars/${id}`, data),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.envVars(projectId),
      });
    },
  });
}

export function useDeleteEnvVar() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
    }: {
      id: string;
      projectId: string;
    }) => api.delete<void>(`/env-vars/${id}`),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.envVars(projectId),
      });
    },
  });
}

export function useBulkImportEnvVars(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BulkImportEnvVarRequest) =>
      api.post<BulkImportEnvVarResponse>(
        `/projects/${projectId}/env-vars/bulk`,
        data,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.envVars(projectId),
      });
    },
  });
}
