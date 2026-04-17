import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  DomainListResponse,
  CreateDomainRequest,
  CreateDomainResponse,
  VerifyDomainResponse,
} from "@/types/api";

export function useDomains(projectId: string) {
  return useQuery({
    queryKey: queryKeys.domains(projectId),
    queryFn: () =>
      api.get<DomainListResponse>(`/projects/${projectId}/domains`),
    enabled: !!projectId,
  });
}

export function useAddDomain(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateDomainRequest) =>
      api.post<CreateDomainResponse>(`/projects/${projectId}/domains`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.domains(projectId),
      });
    },
  });
}

export function useVerifyDomain() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      domainId,
      projectId,
    }: {
      domainId: string;
      projectId: string;
    }) =>
      api.post<VerifyDomainResponse>(`/domains/${domainId}/verify`),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.domains(projectId),
      });
    },
  });
}

export function useDeleteDomain() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      domainId,
      projectId,
    }: {
      domainId: string;
      projectId: string;
    }) => api.delete<void>(`/domains/${domainId}`),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.domains(projectId),
      });
    },
  });
}
