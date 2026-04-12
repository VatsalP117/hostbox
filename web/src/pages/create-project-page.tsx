import { PageHeader } from "@/components/shared/page-header";
import { CreateProjectWizard } from "@/components/projects/create-project-wizard";

export function CreateProjectPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="New Project"
        description="Create a new project to deploy."
      />
      <CreateProjectWizard />
    </div>
  );
}
