import { Navigate } from "react-router-dom";
import { routes } from "@/lib/constants";
import { LoadingPage } from "./loading-page";
import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "@/lib/constants";

export function SetupGuard({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.setupStatus,
    queryFn: async () => {
      const res = await fetch("/api/v1/setup/status");
      if (!res.ok) return { setup_complete: false };
      return res.json() as Promise<{ setup_complete: boolean }>;
    },
    retry: false,
  });

  if (isLoading) return <LoadingPage />;
  if (data?.setup_complete) return <Navigate to={routes.login} replace />;
  return <>{children}</>;
}
