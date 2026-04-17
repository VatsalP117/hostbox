import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  DeploymentListParams,
  DeploymentListResponse,
  DeploymentResponse,
  TriggerDeployRequest,
} from "@/types/api";

export function useDeployments(projectId: string, params?: DeploymentListParams) {
  return useQuery({
    queryKey: queryKeys.deployments(projectId, params),
    queryFn: () =>
      api.get<DeploymentListResponse>(`/projects/${projectId}/deployments`, {
        page: params?.page,
        per_page: params?.per_page,
        status: params?.status,
        branch: params?.branch,
      }),
    enabled: !!projectId,
  });
}

export function useDeployment(id: string) {
  return useQuery({
    queryKey: queryKeys.deployment(id),
    queryFn: () => api.get<DeploymentResponse>(`/deployments/${id}`),
    enabled: !!id,
  });
}

export function useTriggerDeployment(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: TriggerDeployRequest) =>
      api.post<DeploymentResponse>(
        `/projects/${projectId}/deployments/trigger`,
        data,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["deployments", projectId],
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.project(projectId) });
    },
  });
}

export function useCancelDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      api.post<DeploymentResponse>(`/deployments/${id}/cancel`),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.deployment(id) });
      queryClient.invalidateQueries({ queryKey: ["deployments"] });
    },
  });
}

export function useRollbackDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectId,
      deploymentId,
    }: {
      projectId: string;
      deploymentId: string;
    }) =>
      api.post<DeploymentResponse>(`/projects/${projectId}/rollback`, {
        deployment_id: deploymentId,
      }),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.deployments(projectId),
      });
    },
  });
}

export function useRedeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (projectId: string) =>
      api.post<DeploymentResponse>(`/projects/${projectId}/redeploy`),
    onSuccess: (_, projectId) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.deployments(projectId),
      });
    },
  });
}
