import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  AdminStatsResponse,
  AdminUsersResponse,
  AdminActivityResponse,
  AdminActivityParams,
  AdminSettingsResponse,
  UpdateSettingsRequest,
} from "@/types/api";

export function useAdminStats(enabled = true) {
  return useQuery({
    queryKey: queryKeys.adminStats,
    queryFn: () => api.get<AdminStatsResponse>("/admin/stats"),
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
    enabled,
  });
}

export function useAdminUsers() {
  return useQuery({
    queryKey: queryKeys.adminUsers,
    queryFn: () => api.get<AdminUsersResponse>("/admin/users"),
  });
}

export function useAdminActivity(params?: AdminActivityParams) {
  return useQuery({
    queryKey: queryKeys.adminActivity(params),
    queryFn: () =>
      api.get<AdminActivityResponse>("/admin/activity", {
        page: params?.page,
        per_page: params?.per_page,
        action: params?.action,
        resource_type: params?.resource_type,
      }),
  });
}

export function useAdminSettings() {
  return useQuery({
    queryKey: queryKeys.adminSettings,
    queryFn: () => api.get<AdminSettingsResponse>("/admin/settings"),
  });
}

export function useUpdateAdminSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateSettingsRequest) =>
      api.put<AdminSettingsResponse>("/admin/settings", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminSettings });
    },
  });
}
