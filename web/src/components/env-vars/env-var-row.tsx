import { useState } from "react";
import { toast } from "sonner";
import { TableRow, TableCell } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { EnvVarScopeBadge } from "@/components/env-vars/env-var-scope-badge";
import { ConfirmationDialog } from "@/components/shared/confirmation-dialog";
import { useDeleteEnvVar } from "@/hooks/use-env-vars";
import { getApiErrorMessage } from "@/lib/utils";
import type { EnvVar } from "@/types/models";
import { Eye, EyeOff, Trash2 } from "lucide-react";

interface EnvVarRowProps {
  envVar: EnvVar;
  projectId: string;
}

export function EnvVarRow({ envVar, projectId }: EnvVarRowProps) {
  const [revealed, setRevealed] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const deleteEnvVar = useDeleteEnvVar();

  const handleDelete = () => {
    deleteEnvVar.mutate(
      { id: envVar.id, projectId },
      {
        onSuccess: () => {
          toast.success(`Removed ${envVar.key}`);
          setDeleteOpen(false);
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const displayValue = envVar.is_secret && !revealed
    ? "••••••••"
    : envVar.value;

  return (
    <>
      <TableRow>
        <TableCell className="font-mono text-sm">{envVar.key}</TableCell>
        <TableCell>
          <div className="flex items-center gap-2">
            <code className="text-xs text-muted-foreground max-w-[300px] truncate">
              {displayValue}
            </code>
            {envVar.is_secret && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => setRevealed(!revealed)}
              >
                {revealed ? (
                  <EyeOff className="h-3.5 w-3.5" />
                ) : (
                  <Eye className="h-3.5 w-3.5" />
                )}
              </Button>
            )}
          </div>
        </TableCell>
        <TableCell>
          <EnvVarScopeBadge scope={envVar.scope} />
        </TableCell>
        <TableCell>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-destructive"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </TableCell>
      </TableRow>

      <ConfirmationDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Environment Variable"
        description={`Are you sure you want to delete "${envVar.key}"? This will take effect on the next deployment.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteEnvVar.isPending}
      />
    </>
  );
}
