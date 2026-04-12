import { NotificationCard } from "@/components/notifications/notification-card";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import type { NotificationConfig } from "@/types/models";
import { Bell } from "lucide-react";

interface NotificationListProps {
  notifications: NotificationConfig[] | undefined;
  isLoading: boolean;
  projectId: string;
}

export function NotificationList({
  notifications,
  isLoading,
  projectId,
}: NotificationListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 2 }).map((_, i) => (
          <Skeleton key={i} className="h-28 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (!notifications?.length) {
    return (
      <EmptyState
        icon={Bell}
        title="No notifications"
        description="Add a notification to be alerted about deployment events."
      />
    );
  }

  return (
    <div className="space-y-3">
      {notifications.map((n) => (
        <NotificationCard
          key={n.id}
          notification={n}
          projectId={projectId}
        />
      ))}
    </div>
  );
}
