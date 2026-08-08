import { useState } from "react";
import { useNotifications } from "@/hooks/use-notifications";
import { NotificationList } from "@/components/notifications/notification-list";
import { AddNotificationForm } from "@/components/notifications/add-notification-form";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

interface NotificationsTabProps {
  projectId: string;
}

export function NotificationsTab({ projectId }: NotificationsTabProps) {
  const [addOpen, setAddOpen] = useState(false);
  const { data, isLoading } = useNotifications(projectId);

  return (
    <div className="space-y-5">
      <div className="flex flex-col justify-between gap-3 border-b border-border pb-4 sm:flex-row sm:items-center">
        <div><h2 className="text-base font-medium">Notifications</h2><p className="mt-1 text-xs text-muted-foreground">Send deployment events to your team and external systems.</p></div>
        <Button onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add notification
        </Button>
      </div>

      <NotificationList
        notifications={data?.notifications}
        isLoading={isLoading}
        projectId={projectId}
      />

      <AddNotificationForm
        open={addOpen}
        onOpenChange={setAddOpen}
        projectId={projectId}
      />
    </div>
  );
}
