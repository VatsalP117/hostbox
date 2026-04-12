import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { EventToggleList } from "@/components/notifications/event-toggle-list";
import { useCreateNotification } from "@/hooks/use-notifications";
import { getApiErrorMessage } from "@/lib/utils";
import type { NotificationChannel, NotificationEvent } from "@/types/models";
import { Loader2 } from "lucide-react";

const schema = z.object({
  channel: z.enum(["discord", "slack", "webhook"]),
  webhook_url: z.string().url("Enter a valid URL"),
});

type FormValues = z.infer<typeof schema>;

interface AddNotificationFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
}

export function AddNotificationForm({
  open,
  onOpenChange,
  projectId,
}: AddNotificationFormProps) {
  const createNotification = useCreateNotification(projectId);
  const [selectedEvents, setSelectedEvents] = useState<NotificationEvent[]>([
    "deploy_started",
    "deploy_success",
    "deploy_failed",
  ]);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { channel: "discord", webhook_url: "" },
  });

  const onSubmit = (values: FormValues) => {
    createNotification.mutate(
      {
        channel: values.channel as NotificationChannel,
        webhook_url: values.webhook_url,
        events: selectedEvents,
      },
      {
        onSuccess: () => {
          toast.success("Notification added");
          form.reset();
          setSelectedEvents(["deploy_started", "deploy_success", "deploy_failed"]);
          onOpenChange(false);
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Notification</DialogTitle>
          <DialogDescription>
            Get notified about deployment events via your preferred channel.
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="channel"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Channel</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="discord">Discord</SelectItem>
                      <SelectItem value="slack">Slack</SelectItem>
                      <SelectItem value="webhook">Webhook</SelectItem>
                    </SelectContent>
                  </Select>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="webhook_url"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Webhook URL</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="https://discord.com/api/webhooks/..."
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="space-y-2">
              <FormLabel>Events</FormLabel>
              <EventToggleList
                selected={selectedEvents}
                onChange={setSelectedEvents}
              />
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={createNotification.isPending}>
                {createNotification.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Add Notification
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
