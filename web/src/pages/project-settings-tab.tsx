import { useState } from "react";
import { toast } from "sonner";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { ConfirmationDialog } from "@/components/shared/confirmation-dialog";
import {
  BuildSettingsForm,
  type BuildSettingsValues,
} from "@/components/projects/build-settings-form";
import { useUpdateProject, useDeleteProject } from "@/hooks/use-projects";
import { getApiErrorMessage } from "@/lib/utils";
import { routes } from "@/lib/constants";
import { useNavigate } from "react-router-dom";
import type { Project } from "@/types/models";
import { Loader2, Trash2 } from "lucide-react";

interface ProjectSettingsTabProps {
  project: Project;
}

export function ProjectSettingsTab({ project }: ProjectSettingsTabProps) {
  const navigate = useNavigate();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const updateProject = useUpdateProject(project.id);
  const deleteProject = useDeleteProject(project.id);

  const handleBuildSettings = (values: BuildSettingsValues) => {
    updateProject.mutate(
      {
        name: values.name,
        build_command: values.build_command || undefined,
        install_command: values.install_command || undefined,
        output_directory: values.output_directory || undefined,
        root_directory: values.root_directory || undefined,
        node_version: values.node_version || undefined,
      },
      {
        onSuccess: () => toast.success("Build settings updated"),
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const handleToggle = (field: "auto_deploy" | "preview_deployments", value: boolean) => {
    updateProject.mutate(
      { [field]: value },
      {
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const handleDelete = () => {
    deleteProject.mutate(undefined, {
      onSuccess: () => {
        toast.success("Project deleted");
        navigate(routes.projects);
      },
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  return (
    <div className="space-y-6">
      {/* Build Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Build Settings</CardTitle>
          <CardDescription>
            Configure how your project is built and deployed.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <BuildSettingsForm
            defaultValues={{
              name: project.name,
              framework: project.framework ?? "",
              build_command: project.build_command ?? "",
              install_command: project.install_command ?? "",
              output_directory: project.output_directory ?? "",
              root_directory: project.root_directory,
              node_version: project.node_version,
            }}
            onSubmit={handleBuildSettings}
          >
            <Button type="submit" disabled={updateProject.isPending}>
              {updateProject.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save Changes
            </Button>
          </BuildSettingsForm>
        </CardContent>
      </Card>

      {/* Deploy Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Deploy Settings</CardTitle>
          <CardDescription>
            Configure automatic deployment behavior.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label>Auto Deploy</Label>
              <p className="text-xs text-muted-foreground">
                Automatically deploy when changes are pushed to the production
                branch.
              </p>
            </div>
            <Switch
              checked={project.auto_deploy}
              onCheckedChange={(v) => handleToggle("auto_deploy", v)}
            />
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label>Preview Deployments</Label>
              <p className="text-xs text-muted-foreground">
                Create preview deployments for pull requests.
              </p>
            </div>
            <Switch
              checked={project.preview_deployments}
              onCheckedChange={(v) => handleToggle("preview_deployments", v)}
            />
          </div>
          <div className="space-y-2">
            <Label>Production Branch</Label>
            <Input value={project.production_branch} disabled />
            <p className="text-xs text-muted-foreground">
              The branch that triggers production deployments.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>
            These actions are irreversible. Please be certain.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Separator className="mb-4" />
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Delete Project</p>
              <p className="text-xs text-muted-foreground">
                Permanently delete this project and all its data.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete Project
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConfirmationDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Project"
        description={`Are you sure you want to delete "${project.name}"? This will remove all deployments, domains, and environment variables. This action cannot be undone.`}
        confirmLabel="Delete Project"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteProject.isPending}
      />
    </div>
  );
}
