import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { routes } from "@/lib/constants";
import { BrandMark } from "@/components/shared/brand-mark";

export function NotFoundPage() {
  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-black p-4">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,#1f1f1f_1px,transparent_1px),linear-gradient(to_bottom,#1f1f1f_1px,transparent_1px)] bg-[size:64px_64px] [mask-image:radial-gradient(ellipse_50%_45%_at_50%_50%,#000_20%,transparent_100%)]" />
      <div className="relative space-y-4 text-center">
        <BrandMark className="mx-auto h-8 w-8 text-[#52a8ff]" />
        <p className="font-mono text-xs text-muted-foreground">ERROR 404</p>
        <h1 className="text-2xl font-semibold">Page not found</h1>
        <p className="text-sm text-muted-foreground">
          The page you're looking for doesn't exist.
        </p>
        <Link to={routes.dashboard} className="inline-block">
          <Button>Back to dashboard</Button>
        </Link>
      </div>
    </div>
  );
}
