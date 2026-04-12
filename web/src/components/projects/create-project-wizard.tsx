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
import { Label } from "@/components/ui/label";
import {
  BuildSettingsForm,
  type BuildSettingsValues,
} from "@/components/projects/build-settings-form";
import { useCreateProject } from "@/hooks/use-projects";
import { routes } from "@/lib/constants";
import { getApiErrorMessage } from "@/lib/utils";
import { Loader2, ArrowLeft, ArrowRight } from "lucide-react";

type Step = "repo" | "settings";

export function CreateProjectWizard() {
  const navigate = useNavigate();
  const createProject = useCreateProject();
  const [step, setStep] = useState<Step>("repo");
  const [repoUrl, setRepoUrl] = useState("");

  const handleSubmit = (values: BuildSettingsValues) => {
    createProject.mutate(
      {
        name: values.name,
        github_repo: repoUrl || undefined,
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
              Enter a GitHub repository URL or skip to create a project without
              one.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="repo">GitHub Repository</Label>
              <Input
                id="repo"
                placeholder="owner/repository"
                value={repoUrl}
                onChange={(e) => setRepoUrl(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                e.g. vercel/next.js
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
