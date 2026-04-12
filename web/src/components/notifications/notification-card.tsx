import { useState } from "react";
import { toast } from "sonner";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { ConfirmationDialog } from "@/components/shared/confirmation-dialog";
import {
  useUpdateNotification,
  useDeleteNotification,
  useTestNotification,
} from "@/hooks/use-notifications";
import { getApiErrorMessage } from "@/lib/utils";
import type { NotificationConfig } from "@/types/models";
import { Loader2, Trash2, Send, MessageSquare, Hash, Globe } from "lucide-react";

const channelIcons: Record<string, React.ElementType> = {
  discord: MessageSquare,
  slack: Hash,
  webhook: Globe,
};

const channelLabels: Record<string, string> = {
  discord: "Discord",
  slack: "Slack",
  webhook: "Webhook",
};

interface NotificationCardProps {
  notification: NotificationConfig;
  projectId: string;
}

export function NotificationCard({
  notification,
  projectId,
}: NotificationCardProps) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const updateNotification = useUpdateNotification();
  const deleteNotification = useDeleteNotification();
  const testNotification = useTestNotification();

  const ChannelIcon = channelIcons[notification.channel] ?? Globe;
  const events = notification.events ? notification.events.split(",") : [];

  const handleToggle = (enabled: boolean) => {
    updateNotification.mutate(
      { id: notification.id, projectId, data: { enabled } },
      {
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const handleTest = () => {
    testNotification.mutate(notification.id, {
      onSuccess: () => toast.success("Test notification sent"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  const handleDelete = () => {
    deleteNotification.mutate(
      { id: notification.id, projectId },
      {
        onSuccess: () => {
          toast.success("Notification removed");
          setDeleteOpen(false);
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  return (
    <>
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ChannelIcon className="h-4 w-4 text-muted-foreground" />
              <CardTitle className="text-base">
                {channelLabels[notification.channel] ?? notification.channel}
              </CardTitle>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                checked={notification.enabled}
                onCheckedChange={handleToggle}
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handleTest}
                disabled={testNotification.isPending}
              >
                {testNotification.isPending ? (
                  <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Send className="mr-2 h-3.5 w-3.5" />
                )}
                Test
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-destructive"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-xs text-muted-foreground font-mono truncate">
            {notification.webhook_url}
          </p>
          {events.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {events.map((event) => (
                <Badge key={event} variant="secondary" className="text-[10px]">
                  {event.replace(/_/g, " ")}
                </Badge>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmationDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Remove Notification"
        description="Are you sure you want to remove this notification configuration?"
        confirmLabel="Remove"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteNotification.isPending}
      />
    </>
  );
}
