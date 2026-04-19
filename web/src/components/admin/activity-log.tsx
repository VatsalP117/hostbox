import { useState } from "react";
import { PaginationControls } from "@/components/shared/pagination-controls";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/empty-state";
import { useAdminActivity } from "@/hooks/use-admin";
import { Activity as ActivityIcon, GitBranch, Rocket, User, Settings, Trash2, Box } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { TimeAgo } from "@/components/shared/time-ago";
import type { Activity } from "@/types/models";

// Activity type icons mapping based on resource_type
const activityIcons: Record<string, React.ElementType> = {
  deployment: Rocket,
  project: GitBranch,
  user: User,
  settings: Settings,
  deletion: Trash2,
  project_env: Box,
  default: ActivityIcon,
};

// Format action for display
function formatAction(action: string): string {
  return action.replace(/[._]/g, " ").replace(/\b\w/g, (l) => l.toUpperCase());
}

interface ActivityRowProps {
  activity: Activity;
}

function ActivityRow({ activity }: ActivityRowProps) {
  const Icon = activityIcons[activity.resource_type] || activityIcons.default;
  const actionLabel = formatAction(activity.action);
  const actorLabel = activity.user_name || (activity.user_id ? `User ${activity.user_id.slice(0, 8)}...` : "System");
  const subjectLabel = activity.project_name || activity.resource_id;

  return (
    <div className="flex items-center space-x-4 p-4 hover:bg-surface-container transition-colors cursor-pointer">
      <div className="w-10 h-10 rounded-full bg-surface-container-high flex items-center justify-center flex-shrink-0">
        <Icon className="h-5 w-5 text-muted-foreground" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <p className="font-body text-sm font-medium text-foreground truncate">
            {actionLabel}
          </p>
          <span className="font-label text-xs text-muted-foreground flex-shrink-0">
            <TimeAgo date={activity.created_at} />
          </span>
        </div>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="font-body text-xs text-muted-foreground">
            {actorLabel}
          </span>
          {subjectLabel && (
            <span className="font-body text-xs text-muted-foreground/80 truncate">
              on {subjectLabel}
            </span>
          )}
          <Badge variant="outline" className="text-[10px] font-label bg-surface-container-high border-outline-variant/30">
            {activity.resource_type}
          </Badge>
        </div>
      </div>
    </div>
  );
}

export function ActivityLog() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminActivity({ page, per_page: 20 });

  if (isLoading) {
    return (
      <div className="bg-surface-container-low rounded-xl p-6 space-y-4">
        <Skeleton className="h-8 w-48 bg-surface-container" />
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg bg-surface-container" />
          ))}
        </div>
      </div>
    );
  }

  if (!data?.activities?.length) {
    return (
      <div className="bg-surface-container-low rounded-xl p-8">
        <EmptyState
          icon={ActivityIcon}
          title="No activity"
          description="Activity will appear here once users start interacting."
        />
      </div>
    );
  }

  return (
    <div className="bg-surface-container-low rounded-xl overflow-hidden">
      <div className="p-6 border-b border-outline-variant/15">
        <h3 className="font-headline text-lg font-bold text-foreground">Activity Log</h3>
        <p className="font-label text-xs text-muted-foreground mt-1 uppercase tracking-wider">
          Recent platform events
        </p>
      </div>

      <div className="divide-y divide-outline-variant/15">
        {data.activities.map((activity) => (
          <ActivityRow key={activity.id} activity={activity} />
        ))}
      </div>

      {data.pagination && data.pagination.total_pages > 1 && (
        <div className="p-4 border-t border-outline-variant/15">
          <PaginationControls pagination={data.pagination} onPageChange={setPage} />
        </div>
      )}
    </div>
  );
}
