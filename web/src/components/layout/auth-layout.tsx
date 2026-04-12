import { Outlet } from "react-router-dom";

export function AuthLayout() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-md space-y-6">
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 32 32"
              fill="none"
              className="h-8 w-8"
            >
              <path
                d="M8 12L16 8L24 12V20L16 24L8 20V12Z"
                stroke="currentColor"
                strokeWidth="1.5"
                fill="none"
                className="text-primary-foreground"
              />
              <path
                d="M16 8V24"
                stroke="currentColor"
                strokeWidth="1.5"
                className="text-primary-foreground"
              />
              <path
                d="M8 12L16 16L24 12"
                stroke="currentColor"
                strokeWidth="1.5"
                className="text-primary-foreground"
              />
            </svg>
          </div>
          <h1 className="text-2xl font-bold tracking-tight">Hostbox</h1>
          <p className="text-sm text-muted-foreground">
            Self-hosted deployment platform
          </p>
        </div>
        <Outlet />
      </div>
    </div>
  );
}
