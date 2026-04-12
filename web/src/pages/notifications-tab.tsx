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
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Notification
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
