import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { AlertCircle, Github, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useCompleteGitHubManifest } from "@/hooks/use-github";
import { routes } from "@/lib/constants";
import { getApiErrorMessage } from "@/lib/utils";

export function GitHubManifestPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const completeManifest = useCompleteGitHubManifest();
  const { error, isPending, isSuccess, mutate } = completeManifest;
  const code = searchParams.get("code") ?? "";
  const state = searchParams.get("state") ?? "";

  useEffect(() => {
    if (!code || !state || isPending || isSuccess) {
      return;
    }

    mutate(
      { code, state },
      {
        onSuccess: (status) => {
          if (status.install_url) {
            window.location.href = status.install_url;
            return;
          }
          navigate(routes.newProject, { replace: true });
        },
      },
    );
  }, [code, isPending, isSuccess, mutate, navigate, state]);

  const hasMissingParams = !code || !state;
  const errorMessage = hasMissingParams
    ? "GitHub did not send the connection code Hostbox expected."
    : error
      ? getApiErrorMessage(error)
      : "";

  return (
    <div className="mx-auto flex min-h-[50vh] max-w-lg items-center">
      <Card className="w-full">
        <CardHeader>
          <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-md border bg-background">
            <Github className="h-5 w-5" />
          </div>
          <CardTitle>
            {errorMessage ? "GitHub connection failed" : "Connecting GitHub"}
          </CardTitle>
          <CardDescription>
            {errorMessage
              ? "The GitHub App was not activated in Hostbox."
              : "Hostbox is activating your GitHub App and will send you to choose repositories."}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {errorMessage ? (
            <div className="flex gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{errorMessage}</span>
            </div>
          ) : (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>Finishing secure GitHub setup</span>
            </div>
          )}
          <Button
            variant={errorMessage ? "default" : "outline"}
            onClick={() => navigate(routes.newProject, { replace: true })}
          >
            Back to project setup
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
