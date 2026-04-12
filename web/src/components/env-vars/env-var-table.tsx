import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { EnvVarRow } from "@/components/env-vars/env-var-row";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import type { EnvVar } from "@/types/models";
import { Key } from "lucide-react";

interface EnvVarTableProps {
  envVars: EnvVar[] | undefined;
  isLoading: boolean;
  projectId: string;
}

export function EnvVarTable({
  envVars,
  isLoading,
  projectId,
}: EnvVarTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (!envVars?.length) {
    return (
      <EmptyState
        icon={Key}
        title="No environment variables"
        description="Add environment variables for your project."
      />
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Key</TableHead>
            <TableHead>Value</TableHead>
            <TableHead>Scope</TableHead>
            <TableHead className="w-[80px]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {envVars.map((envVar) => (
            <EnvVarRow key={envVar.id} envVar={envVar} projectId={projectId} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
