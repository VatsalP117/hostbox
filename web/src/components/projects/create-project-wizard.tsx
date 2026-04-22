import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
  useCreateGitHubManifest,
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
  ExternalLink,
  LockKeyhole,
} from "lucide-react";

type Step = "repo" | "settings";

export function CreateProjectWizard() {
  const navigate = useNavigate();
  const createProject = useCreateProject();
  const [step, setStep] = useState<Step>("repo");
  const [repoUrl, setRepoUrl] = useState("");
  const [installationId, setInstallationId] = useState<string>("");
  const [manualMode, setManualMode] = useState(false);

  const createManifest = useCreateGitHubManifest();
  const { data: githubStatus, isLoading: githubStatusLoading } =
    useGitHubStatus();
  const { data: installations } = useGitHubInstallations(
    !!githubStatus?.configured,
  );
  const { data: repos } = useGitHubRepos(
    installationId
      ? { installation_id: Number(installationId), per_page: 100 }
      : undefined,
  );
  const hasInstallations = !!installations?.installations?.length;
  const canContinue = repoUrl.trim().length > 0;

  const handleConnectGitHub = () => {
    if (githubStatus?.configured && githubStatus.install_url) {
      window.location.href = githubStatus.install_url;
      return;
    }

    createManifest.mutate(undefined, {
      onSuccess: (data) => submitGitHubManifest(data.action_url, data.manifest),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

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
            <CardTitle>Import from GitHub</CardTitle>
            <CardDescription>
              Connect your GitHub account, choose which repositories Hostbox can
              access, then pick a project to deploy.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {hasInstallations ? (
              <div className="space-y-4 rounded-lg border p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-md border bg-background">
                      <Github className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="font-medium text-foreground">
                        GitHub is connected
                      </h3>
                      <p className="text-sm text-muted-foreground">
                        Select an account and repository to deploy.
                      </p>
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleConnectGitHub}
                  >
                    Manage access
                    <ExternalLink className="h-4 w-4" />
                  </Button>
                </div>

                <div className="space-y-2">
                  <Label>GitHub account</Label>
                  <Select
                    value={installationId}
                    onValueChange={(value) => {
                      setInstallationId(value);
                      setRepoUrl("");
                      setManualMode(false);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select an account" />
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

                <div className="space-y-2">
                  <Label>Repository</Label>
                  <Select
                    value={repoUrl}
                    onValueChange={(value) => {
                      setRepoUrl(value);
                      setManualMode(false);
                    }}
                    disabled={!installationId}
                  >
                    <SelectTrigger>
                      <SelectValue
                        placeholder={
                          installationId
                            ? "Select a repository"
                            : "Select an account first"
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {repos?.repos?.map((repo) => (
                        <SelectItem key={repo.full_name} value={repo.full_name}>
                          {repo.full_name}
                          {repo.private ? " private" : ""}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            ) : (
              <div className="rounded-lg border p-5">
                <div className="flex items-start gap-4">
                  <div className="flex h-11 w-11 items-center justify-center rounded-md border bg-background">
                    <Github className="h-5 w-5" />
                  </div>
                  <div className="flex-1 space-y-3">
                    <div>
                      <h3 className="font-medium text-foreground">
                        Connect GitHub
                      </h3>
                      <p className="mt-1 text-sm text-muted-foreground">
                        Hostbox will create a private GitHub App for this
                        instance, then GitHub will ask which repositories you
                        want to deploy.
                      </p>
                    </div>
                    <Button
                      type="button"
                      onClick={handleConnectGitHub}
                      disabled={githubStatusLoading || createManifest.isPending}
                    >
                      {githubStatusLoading || createManifest.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Github className="h-4 w-4" />
                      )}
                      Connect GitHub
                    </Button>
                  </div>
                </div>
              </div>
            )}

            <div className="border-t pt-4">
              {!manualMode ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setManualMode(true);
                    setInstallationId("");
                    setRepoUrl("");
                  }}
                >
                  <LockKeyhole className="h-4 w-4" />
                  Use a public repository URL instead
                </Button>
              ) : (
                <div className="space-y-2">
                  <Label htmlFor="repo">Public GitHub repository</Label>
                  <Input
                    id="repo"
                    placeholder="owner/repository"
                    value={repoUrl}
                    onChange={(e) => setRepoUrl(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Public repositories can be deployed without connecting a
                    GitHub account.
                  </p>
                </div>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => navigate(-1)}>
                Cancel
              </Button>
              <Button
                onClick={() => setStep("settings")}
                disabled={!canContinue}
              >
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

function submitGitHubManifest(
  actionURL: string,
  manifest: Record<string, unknown>,
) {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = actionURL;

  const input = document.createElement("input");
  input.type = "hidden";
  input.name = "manifest";
  input.value = JSON.stringify(manifest);

  form.appendChild(input);
  document.body.appendChild(form);
  form.submit();
}
