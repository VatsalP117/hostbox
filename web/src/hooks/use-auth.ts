import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { useAuthStore } from "@/stores/auth-store";
import { queryKeys } from "@/lib/constants";
import type {
  LoginRequest,
  LoginResponse,
  MeResponse,
  RefreshResponse,
} from "@/types/api";

export function useLogin() {
  const login = useAuthStore((s) => s.login);
  return useMutation({
    mutationFn: (data: LoginRequest) =>
      api.post<LoginResponse>("/auth/login", data, true),
    onSuccess: (data) => {
      login(data.access_token, data.user);
    },
  });
}

export function useLogout() {
  const logout = useAuthStore((s) => s.logout);
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<void>("/auth/logout"),
    onSettled: () => {
      logout();
      queryClient.clear();
    },
  });
}

export function useBootstrapAuth() {
  const login = useAuthStore((s) => s.login);
  const logout = useAuthStore((s) => s.logout);

  return async (): Promise<boolean> => {
    try {
      const refreshRes = await fetch("/api/v1/auth/refresh", {
        method: "POST",
        credentials: "include",
      });
      if (!refreshRes.ok) {
        logout();
        return false;
      }
      const { access_token } = (await refreshRes.json()) as RefreshResponse;
      useAuthStore.getState().setAccessToken(access_token);

      const meRes = await api.get<MeResponse>("/auth/me");
      login(access_token, meRes.user);
      return true;
    } catch {
      logout();
      return false;
    }
  };
}
