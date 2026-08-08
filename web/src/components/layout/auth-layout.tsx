import { Outlet } from "react-router-dom";
import { BrandMark } from "@/components/shared/brand-mark";

export function AuthLayout() {
  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-black p-4">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,#1f1f1f_1px,transparent_1px),linear-gradient(to_bottom,#1f1f1f_1px,transparent_1px)] bg-[size:64px_64px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_35%,#000_20%,transparent_100%)]" />
      <div className="relative w-full max-w-sm space-y-6">
        <div className="text-center">
          <div className="mx-auto mb-5 flex h-11 w-11 items-center justify-center rounded-lg border border-[#2a2a2a] bg-[#0e0e0e]">
            <BrandMark className="h-7 w-7 text-[#52a8ff]" />
          </div>
          <h1 className="text-xl font-semibold tracking-tight">Hostbox</h1>
          <p className="text-sm text-muted-foreground">
            The deployment platform for your own VM.
          </p>
        </div>
        <Outlet />
      </div>
    </div>
  );
}
