import { useState, useEffect } from "react";
import { Navigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { useBootstrapAuth } from "@/hooks/use-auth";
import { routes } from "@/lib/constants";
import { LoadingPage } from "./loading-page";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const [checking, setChecking] = useState(!isAuthenticated);
  const [failed, setFailed] = useState(false);
  const bootstrap = useBootstrapAuth();

  useEffect(() => {
    if (isAuthenticated) return;

    let cancelled = false;
    bootstrap().then((success) => {
      if (cancelled) return;
      if (!success) setFailed(true);
      setChecking(false);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  if (checking) return <LoadingPage />;
  if (failed && !isAuthenticated)
    return <Navigate to={routes.login} replace />;

  return <>{children}</>;
}
