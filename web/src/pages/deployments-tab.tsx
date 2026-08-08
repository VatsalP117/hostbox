import { useState } from "react";
import { useDeployments } from "@/hooks/use-deployments";
import { DeploymentList } from "@/components/deployments/deployment-list";
import { PaginationControls } from "@/components/shared/pagination-controls";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { DeploymentStatus } from "@/types/models";

interface DeploymentsTabProps {
  projectId: string;
  productionBranch: string;
}

export function DeploymentsTab({ projectId }: DeploymentsTabProps) {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<DeploymentStatus | "all">(
    "all",
  );

  const { data, isLoading } = useDeployments(projectId, {
    page,
    per_page: 10,
    status: statusFilter === "all" ? undefined : statusFilter,
  });

  return (
    <div className="space-y-5">
      <div className="flex flex-col justify-between gap-3 border-b border-border pb-4 sm:flex-row sm:items-center">
        <div><h2 className="text-base font-medium">Deployment history</h2><p className="mt-1 text-xs text-muted-foreground">Builds created from connected branches and pull requests.</p></div>
        <Select
          value={statusFilter}
          onValueChange={(v) => {
            setStatusFilter(v as DeploymentStatus | "all");
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[170px] bg-black">
            <SelectValue placeholder="Filter by status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="queued">Queued</SelectItem>
            <SelectItem value="building">Building</SelectItem>
            <SelectItem value="ready">Ready</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="cancelled">Cancelled</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <DeploymentList
        deployments={data?.deployments}
        isLoading={isLoading}
        projectId={projectId}
      />

      {data?.pagination && (
        <PaginationControls
          pagination={data.pagination}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}
