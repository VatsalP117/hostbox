import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Github, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { queryKeys, routes } from "@/lib/constants";

export function GitHubSetupPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const setupAction = searchParams.get("setup_action");
  const installationId = searchParams.get("installation_id");
  const isUpdate = setupAction === "update";

  useEffect(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.githubStatus });
    queryClient.invalidateQueries({ queryKey: queryKeys.installations });

    const timeout = window.setTimeout(() => {
      navigate(routes.newProject, { replace: true });
    }, 1400);

    return () => window.clearTimeout(timeout);
  }, [navigate, queryClient]);

  return (
    <div className="mx-auto flex min-h-[50vh] max-w-lg items-center">
      <Card className="w-full">
        <CardHeader>
          <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-md border bg-background">
            <Github className="h-5 w-5" />
          </div>
          <CardTitle>GitHub {isUpdate ? "updated" : "connected"}</CardTitle>
          <CardDescription>
            {installationId
              ? `Installation ${installationId} is ready for repository selection.`
              : "Your GitHub App installation is ready for repository selection."}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4 text-green-600" />
            <span>Returning to project setup</span>
            <Loader2 className="h-4 w-4 animate-spin" />
          </div>
          <Button onClick={() => navigate(routes.newProject, { replace: true })}>
            Continue
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
