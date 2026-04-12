import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  NotificationListResponse,
  CreateNotificationRequest,
  UpdateNotificationRequest,
} from "@/types/api";
import type { NotificationConfig } from "@/types/models";

export function useNotifications(projectId: string) {
  return useQuery({
    queryKey: queryKeys.notifications(projectId),
    queryFn: () =>
      api.get<NotificationListResponse>(
        `/projects/${projectId}/notifications`,
      ),
    enabled: !!projectId,
  });
}

export function useCreateNotification(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateNotificationRequest) =>
      api.post<{ notification: NotificationConfig }>(
        `/projects/${projectId}/notifications`,
        data,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.notifications(projectId),
      });
    },
  });
}

export function useUpdateNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      projectId: string;
      data: UpdateNotificationRequest;
    }) =>
      api.patch<{ notification: NotificationConfig }>(
        `/notifications/${id}`,
        data,
      ),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.notifications(projectId),
      });
    },
  });
}

export function useDeleteNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
    }: {
      id: string;
      projectId: string;
    }) => api.delete<void>(`/notifications/${id}`),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.notifications(projectId),
      });
    },
  });
}

export function useTestNotification() {
  return useMutation({
    mutationFn: (id: string) =>
      api.post<{ success: boolean }>(`/notifications/${id}/test`),
  });
}
