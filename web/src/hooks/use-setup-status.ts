import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import type {
  SetupStatusResponse,
  SetupRequest,
  SetupResponse,
} from "@/types/api";
import { useAuthStore } from "@/stores/auth-store";

export function useSetupStatus() {
  return useQuery({
    queryKey: queryKeys.setupStatus,
    queryFn: () => api.get<SetupStatusResponse>("/setup/status"),
    retry: false,
  });
}

export function useSetup() {
  const login = useAuthStore((s) => s.login);
  return useMutation({
    mutationFn: (data: SetupRequest) =>
      api.post<SetupResponse>("/setup", data, true),
    onSuccess: (data) => {
      login(data.access_token, data.user);
    },
  });
}
