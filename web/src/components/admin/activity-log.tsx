import { useState } from "react";
import { ActivityRow } from "@/components/admin/activity-row";
import { PaginationControls } from "@/components/shared/pagination-controls";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/empty-state";
import { useAdminActivity } from "@/hooks/use-admin";
import { Activity as ActivityIcon } from "lucide-react";

export function ActivityLog() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminActivity({ page, per_page: 20 });

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (!data?.activities?.length) {
    return (
      <EmptyState
        icon={ActivityIcon}
        title="No activity"
        description="Activity will appear here once users start interacting."
      />
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        {data.activities.map((activity) => (
          <ActivityRow key={activity.id} activity={activity} />
        ))}
      </div>
      {data.pagination && (
        <PaginationControls pagination={data.pagination} onPageChange={setPage} />
      )}
    </div>
  );
}
