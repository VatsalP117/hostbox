import { useAuthStore } from "@/stores/auth-store";
import type { ApiError } from "@/types/api";

class ApiClient {
  private baseUrl = "/api/v1";
  private refreshPromise: Promise<string> | null = null;

  private getToken(): string | null {
    return useAuthStore.getState().accessToken;
  }

  private setToken(token: string): void {
    useAuthStore.getState().setAccessToken(token);
  }

  private clearAuth(): void {
    useAuthStore.getState().logout();
  }

  private async refreshToken(): Promise<string> {
    if (this.refreshPromise) return this.refreshPromise;

    this.refreshPromise = fetch(`${this.baseUrl}/auth/refresh`, {
      method: "POST",
      credentials: "include",
    })
      .then(async (res) => {
        if (!res.ok) throw new Error("Refresh failed");
        const data = await res.json();
        this.setToken(data.access_token);
        return data.access_token as string;
      })
      .finally(() => {
        this.refreshPromise = null;
      });

    return this.refreshPromise;
  }

  async request<T>(
    method: string,
    path: string,
    options?: {
      body?: unknown;
      params?: Record<string, string | number | undefined>;
      skipAuth?: boolean;
    },
  ): Promise<T> {
    const url = new URL(`${this.baseUrl}${path}`, window.location.origin);

    if (options?.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.set(key, String(value));
        }
      });
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    const token = this.getToken();
    if (token && !options?.skipAuth) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    let res = await fetch(url.toString(), {
      method,
      headers,
      credentials: "include",
      body: options?.body ? JSON.stringify(options.body) : undefined,
    });

    // Auto-refresh on 401
    if (res.status === 401 && !options?.skipAuth) {
      try {
        const newToken = await this.refreshToken();
        headers["Authorization"] = `Bearer ${newToken}`;
        res = await fetch(url.toString(), {
          method,
          headers,
          credentials: "include",
          body: options?.body ? JSON.stringify(options.body) : undefined,
        });
      } catch {
        this.clearAuth();
        window.location.href = "/login";
        throw new Error("Session expired");
      }
    }

    if (!res.ok) {
      const error: ApiError = await res.json().catch(() => ({
        error: {
          code: "UNKNOWN",
          message: res.statusText,
        },
      }));
      throw error;
    }

    // Handle 204 No Content
    if (res.status === 204) {
      return {} as T;
    }

    return res.json() as Promise<T>;
  }

  get<T>(path: string, params?: Record<string, string | number | undefined>) {
    return this.request<T>("GET", path, { params });
  }

  post<T>(path: string, body?: unknown, skipAuth?: boolean) {
    return this.request<T>("POST", path, { body, skipAuth });
  }

  patch<T>(path: string, body?: unknown) {
    return this.request<T>("PATCH", path, { body });
  }

  put<T>(path: string, body?: unknown) {
    return this.request<T>("PUT", path, { body });
  }

  delete<T>(path: string) {
    return this.request<T>("DELETE", path);
  }
}

export const api = new ApiClient();
