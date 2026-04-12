import { useState } from "react";
import { useEnvVars } from "@/hooks/use-env-vars";
import { EnvVarTable } from "@/components/env-vars/env-var-table";
import { AddEnvVarForm } from "@/components/env-vars/add-env-var-form";
import { EnvVarImportDialog } from "@/components/env-vars/env-var-import-dialog";
import { Button } from "@/components/ui/button";
import { Plus, Upload } from "lucide-react";

interface EnvVarsTabProps {
  projectId: string;
}

export function EnvVarsTab({ projectId }: EnvVarsTabProps) {
  const [addOpen, setAddOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const { data, isLoading } = useEnvVars(projectId);

  return (
    <div className="space-y-4">
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={() => setImportOpen(true)}>
          <Upload className="mr-2 h-4 w-4" />
          Import .env
        </Button>
        <Button onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Variable
        </Button>
      </div>

      <EnvVarTable
        envVars={data?.env_vars}
        isLoading={isLoading}
        projectId={projectId}
      />

      <AddEnvVarForm
        open={addOpen}
        onOpenChange={setAddOpen}
        projectId={projectId}
      />

      <EnvVarImportDialog
        open={importOpen}
        onOpenChange={setImportOpen}
        projectId={projectId}
      />
    </div>
  );
}
