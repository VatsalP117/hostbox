import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import {
  BuildSettingsForm,
  type BuildSettingsValues,
} from "@/components/projects/build-settings-form";
import {
  useGitHubInstallations,
  useGitHubRepos,
  useGitHubStatus,
} from "@/hooks/use-github";
import { useCreateProject } from "@/hooks/use-projects";
import { routes } from "@/lib/constants";
import { getApiErrorMessage } from "@/lib/utils";
import {
  Loader2,
  ArrowLeft,
  ArrowRight,
  Github,
  RefreshCw,
  ExternalLink,
} from "lucide-react";

type Step = "repo" | "settings";

export function CreateProjectWizard() {
  const navigate = useNavigate();
  const createProject = useCreateProject();
  const [step, setStep] = useState<Step>("repo");
  const [repoUrl, setRepoUrl] = useState("");
  const [installationId, setInstallationId] = useState<string>("");

  const { data: githubStatus } = useGitHubStatus();
  const {
    data: installations,
    isFetching: installationsFetching,
    refetch: refetchInstallations,
  } = useGitHubInstallations(!!githubStatus?.configured);
  const { data: repos } = useGitHubRepos(
    installationId
      ? { installation_id: Number(installationId), per_page: 100 }
      : undefined,
  );
  const hasInstallations = !!installations?.installations?.length;

  const handleSubmit = (values: BuildSettingsValues) => {
    createProject.mutate(
      {
        name: values.name,
        github_repo: repoUrl || undefined,
        github_installation_id: installationId
          ? Number(installationId)
          : undefined,
        build_command: values.build_command || undefined,
        install_command: values.install_command || undefined,
        output_directory: values.output_directory || undefined,
        root_directory: values.root_directory || undefined,
        node_version: values.node_version || undefined,
      },
      {
        onSuccess: (data) => {
          toast.success("Project created successfully");
          navigate(routes.project(data.project.id));
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      {/* Step indicator */}
      <div className="flex items-center gap-2 text-sm">
        <span
          className={
            step === "repo"
              ? "font-semibold text-foreground"
              : "text-muted-foreground"
          }
        >
          1. Repository
        </span>
        <ArrowRight className="h-3 w-3 text-muted-foreground" />
        <span
          className={
            step === "settings"
              ? "font-semibold text-foreground"
              : "text-muted-foreground"
          }
        >
          2. Build Settings
        </span>
      </div>

      {step === "repo" && (
        <Card>
          <CardHeader>
            <CardTitle>Connect Repository</CardTitle>
            <CardDescription>
              Pick a repository from a GitHub App installation or enter one
              manually.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {githubStatus?.configured && !hasInstallations ? (
              <Alert>
                <Github className="h-4 w-4" />
                <AlertTitle>No GitHub account connected</AlertTitle>
                <AlertDescription className="space-y-3">
                  <p>
                    Install the Hostbox GitHub App to select private
                    repositories and enable push deployments.
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {githubStatus.install_url ? (
                      <Button asChild size="sm">
                        <a href={githubStatus.install_url}>
                          <Github className="h-4 w-4" />
                          Connect GitHub
                          <ExternalLink className="h-4 w-4" />
                        </a>
                      </Button>
                    ) : (
                      <p className="text-xs text-muted-foreground">
                        Set GITHUB_APP_SLUG to enable the install link.
                      </p>
                    )}
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => refetchInstallations()}
                      disabled={installationsFetching}
                    >
                      {installationsFetching ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <RefreshCw className="h-4 w-4" />
                      )}
                      Refresh
                    </Button>
                  </div>
                </AlertDescription>
              </Alert>
            ) : null}

            {githubStatus && !githubStatus.configured ? (
              <Alert>
                <Github className="h-4 w-4" />
                <AlertTitle>GitHub App not configured</AlertTitle>
                <AlertDescription>
                  Add GITHUB_APP_ID, GITHUB_APP_SLUG, GITHUB_APP_PEM, and
                  GITHUB_WEBHOOK_SECRET to enable private repository selection.
                </AlertDescription>
              </Alert>
            ) : null}

            {hasInstallations ? (
              <>
                <div className="space-y-2">
                  <Label>GitHub Installation</Label>
                  <Select
                    value={installationId}
                    onValueChange={(value) => {
                      setInstallationId(value);
                      setRepoUrl("");
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select an installation" />
                    </SelectTrigger>
                    <SelectContent>
                      {installations.installations.map((installation) => (
                        <SelectItem
                          key={installation.id}
                          value={String(installation.id)}
                        >
                          {installation.account} ({installation.target_type})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {installationId && (
                  <div className="space-y-2">
                    <Label>Repository</Label>
                    <Select value={repoUrl} onValueChange={setRepoUrl}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a repository" />
                      </SelectTrigger>
                      <SelectContent>
                        {repos?.repos?.map((repo) => (
                          <SelectItem key={repo.full_name} value={repo.full_name}>
                            {repo.full_name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </>
            ) : null}

            <div className="space-y-2">
              <Label htmlFor="repo">GitHub Repository (manual)</Label>
              <Input
                id="repo"
                placeholder="owner/repository"
                value={repoUrl}
                onChange={(e) => setRepoUrl(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Use this when GitHub App integration is not configured or you
                want to test against a public repository directly.
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => navigate(-1)}>
                Cancel
              </Button>
              <Button onClick={() => setStep("settings")}>
                Continue
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === "settings" && (
        <Card>
          <CardHeader>
            <CardTitle>Build Settings</CardTitle>
            <CardDescription>
              Configure how your project is built and deployed.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <BuildSettingsForm onSubmit={handleSubmit}>
              <div className="flex justify-between pt-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setStep("repo")}
                >
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back
                </Button>
                <Button type="submit" disabled={createProject.isPending}>
                  {createProject.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Create Project
                </Button>
              </div>
            </BuildSettingsForm>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
