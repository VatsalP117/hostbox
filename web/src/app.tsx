import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";

import { RootLayout } from "@/components/layout/root-layout";
import { AuthLayout } from "@/components/layout/auth-layout";
import { AuthGuard } from "@/components/shared/auth-guard";
import { SetupGuard } from "@/components/shared/setup-guard";
import { AdminGuard } from "@/components/shared/admin-guard";
import { ErrorBoundary } from "@/components/shared/error-boundary";
import { LoadingPage } from "@/components/shared/loading-page";

import { SetupPage } from "@/pages/setup-page";
import { LoginPage } from "@/pages/login-page";
import { ForgotPasswordPage } from "@/pages/forgot-password-page";
import { ResetPasswordPage } from "@/pages/reset-password-page";
import { NotFoundPage } from "@/pages/not-found-page";

const DashboardPage = lazy(() =>
  import("@/pages/dashboard-page").then((m) => ({ default: m.DashboardPage })),
);
const ProjectsPage = lazy(() =>
  import("@/pages/projects-page").then((m) => ({ default: m.ProjectsPage })),
);
const CreateProjectPage = lazy(() =>
  import("@/pages/create-project-page").then((m) => ({
    default: m.CreateProjectPage,
  })),
);
const GitHubSetupPage = lazy(() =>
  import("@/pages/github-setup-page").then((m) => ({
    default: m.GitHubSetupPage,
  })),
);
const GitHubManifestPage = lazy(() =>
  import("@/pages/github-manifest-page").then((m) => ({
    default: m.GitHubManifestPage,
  })),
);
const ProjectDetailPage = lazy(() =>
  import("@/pages/project-detail-page").then((m) => ({
    default: m.ProjectDetailPage,
  })),
);
const DeploymentDetailPage = lazy(() =>
  import("@/pages/deployment-detail-page").then((m) => ({
    default: m.DeploymentDetailPage,
  })),
);
const AdminPage = lazy(() =>
  import("@/pages/admin-page").then((m) => ({ default: m.AdminPage })),
);
const ProfilePage = lazy(() =>
  import("@/pages/profile-page").then((m) => ({ default: m.ProfilePage })),
);

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

function SuspenseWrapper({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<LoadingPage />}>{children}</Suspense>;
}

export function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            {/* Public auth routes */}
            <Route element={<AuthLayout />}>
              <Route
                path="/setup"
                element={
                  <SetupGuard>
                    <SetupPage />
                  </SetupGuard>
                }
              />
              <Route path="/login" element={<LoginPage />} />
              <Route
                path="/forgot-password"
                element={<ForgotPasswordPage />}
              />
              <Route
                path="/reset-password"
                element={<ResetPasswordPage />}
              />
            </Route>

            {/* Protected routes */}
            <Route
              element={
                <AuthGuard>
                  <RootLayout />
                </AuthGuard>
              }
            >
              <Route
                index
                element={
                  <SuspenseWrapper>
                    <DashboardPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/projects"
                element={
                  <SuspenseWrapper>
                    <ProjectsPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/projects/new"
                element={
                  <SuspenseWrapper>
                    <CreateProjectPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/github/setup"
                element={
                  <SuspenseWrapper>
                    <GitHubSetupPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/github/manifest"
                element={
                  <SuspenseWrapper>
                    <GitHubManifestPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/projects/:id"
                element={
                  <SuspenseWrapper>
                    <ProjectDetailPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/projects/:id/deployments/:deploymentId"
                element={
                  <SuspenseWrapper>
                    <DeploymentDetailPage />
                  </SuspenseWrapper>
                }
              />
              <Route
                path="/profile"
                element={
                  <SuspenseWrapper>
                    <ProfilePage />
                  </SuspenseWrapper>
                }
              />

              {/* Admin routes */}
              <Route element={<AdminGuard />}>
                <Route
                  path="/admin"
                  element={
                    <SuspenseWrapper>
                      <AdminPage />
                    </SuspenseWrapper>
                  }
                />
              </Route>
            </Route>

            {/* Catch-all */}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </BrowserRouter>
        <Toaster position="bottom-right" richColors />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
