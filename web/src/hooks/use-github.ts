import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  GitHubInstallationsResponse,
  GitHubReposParams,
  GitHubReposResponse,
} from "@/types/api";

export function useGitHubInstallations() {
  return useQuery({
    queryKey: queryKeys.installations,
    queryFn: () => api.get<GitHubInstallationsResponse>("/github/installations"),
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
