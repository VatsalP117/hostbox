import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { routes } from "@/lib/constants";

export function AdminGuard() {
  const user = useAuthStore((s) => s.user);
  if (!user?.is_admin) return <Navigate to={routes.dashboard} replace />;
  return <Outlet />;
}
