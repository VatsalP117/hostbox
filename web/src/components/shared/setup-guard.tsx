import { Navigate } from "react-router-dom";
import { routes } from "@/lib/constants";
import { LoadingPage } from "./loading-page";
import { useSetupStatus } from "@/hooks/use-setup-status";

export function SetupGuard({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = useSetupStatus();

  if (isLoading) return <LoadingPage />;
  if (data && !data.setup_required) return <Navigate to={routes.login} replace />;
  return <>{children}</>;
}
