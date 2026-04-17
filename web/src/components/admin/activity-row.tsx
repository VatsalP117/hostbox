import { TimeAgo } from "@/components/shared/time-ago";
import { Badge } from "@/components/ui/badge";
import type { Activity } from "@/types/models";

interface ActivityRowProps {
  activity: Activity;
}

export function ActivityRow({ activity }: ActivityRowProps) {
  const actionLabel = activity.action.replace(/[._]/g, " ");

  return (
    <div className="flex items-center justify-between rounded-md border p-3">
      <div className="flex items-center gap-3 min-w-0">
        <div className="min-w-0">
          <p className="text-sm font-medium capitalize">{actionLabel}</p>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{activity.user_id ? `User ${activity.user_id}` : "System"}</span>
            {activity.resource_id && (
              <>
                <span>·</span>
                <span className="truncate">{activity.resource_id}</span>
              </>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3 shrink-0">
        <Badge variant="outline" className="text-[10px]">
          {activity.resource_type}
        </Badge>
        <TimeAgo date={activity.created_at} />
      </div>
    </div>
  );
}
