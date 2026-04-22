import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  CompleteGitHubManifestRequest,
  GitHubInstallationsResponse,
  GitHubManifestResponse,
  GitHubReposParams,
  GitHubReposResponse,
  GitHubStatusResponse,
} from "@/types/api";

export function useGitHubStatus() {
  return useQuery({
    queryKey: queryKeys.githubStatus,
    queryFn: () => api.get<GitHubStatusResponse>("/github/status"),
  });
}

export function useGitHubInstallations(enabled = true) {
  return useQuery({
    queryKey: queryKeys.installations,
    queryFn: () => api.get<GitHubInstallationsResponse>("/github/installations"),
    enabled,
  });
}

export function useCreateGitHubManifest() {
  return useMutation({
    mutationFn: () => api.post<GitHubManifestResponse>("/github/manifest"),
  });
}

export function useCompleteGitHubManifest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CompleteGitHubManifestRequest) =>
      api.post<GitHubStatusResponse>("/github/manifest/complete", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.githubStatus });
      queryClient.invalidateQueries({ queryKey: queryKeys.installations });
    },
  });
}

export function useGitHubRepos(params?: GitHubReposParams) {
  return useQuery({
    queryKey: params?.installation_id
      ? queryKeys.repos(params.installation_id)
      : ["github-repos", "disabled"],
    queryFn: () =>
      api.get<GitHubReposResponse>("/github/repos", {
        installation_id: params?.installation_id,
        page: params?.page,
        per_page: params?.per_page,
      }),
    enabled: !!params?.installation_id,
  });
}
