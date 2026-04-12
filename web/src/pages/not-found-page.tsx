import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { routes } from "@/lib/constants";

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="text-center space-y-4">
        <h1 className="text-6xl font-bold text-muted-foreground">404</h1>
        <h2 className="text-xl font-semibold">Page not found</h2>
        <p className="text-sm text-muted-foreground">
          The page you're looking for doesn't exist.
        </p>
        <Link to={routes.dashboard}>
          <Button>Back to Dashboard</Button>
        </Link>
      </div>
    </div>
  );
}
