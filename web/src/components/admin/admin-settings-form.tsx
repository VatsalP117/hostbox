import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from "@/components/ui/form";
import { Skeleton } from "@/components/ui/skeleton";
import { useAdminSettings, useUpdateAdminSettings } from "@/hooks/use-admin";
import { getApiErrorMessage } from "@/lib/utils";
import { Loader2 } from "lucide-react";

const settingsSchema = z.object({
  registration_enabled: z.boolean(),
  max_projects: z.coerce.number().min(1),
  max_concurrent_builds: z.coerce.number().min(1),
  artifact_retention_days: z.coerce.number().min(1),
});

type SettingsValues = z.infer<typeof settingsSchema>;

export function AdminSettingsForm() {
  const { data, isLoading } = useAdminSettings();
  const update = useUpdateAdminSettings();

  const form = useForm<SettingsValues>({
    resolver: zodResolver(settingsSchema),
    values: data?.settings
      ? {
          registration_enabled: data.settings.registration_enabled,
          max_projects: data.settings.max_projects,
          max_concurrent_builds: data.settings.max_concurrent_builds,
          artifact_retention_days: data.settings.artifact_retention_days,
        }
      : undefined,
  });

  const onSubmit = (values: SettingsValues) => {
    update.mutate(values, {
      onSuccess: () => toast.success("Settings updated"),
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-6 space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Platform Settings</CardTitle>
        <CardDescription>
          Configure global platform settings.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="registration_enabled"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between rounded-lg border p-3">
                  <div className="space-y-0.5">
                    <FormLabel>User Registration</FormLabel>
                    <FormDescription>
                      Allow new users to register.
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="max_projects"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Max Projects per User</FormLabel>
                  <FormControl>
                    <Input type="number" min={1} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="max_concurrent_builds"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Max Concurrent Builds</FormLabel>
                  <FormControl>
                    <Input type="number" min={1} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="artifact_retention_days"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Artifact Retention (days)</FormLabel>
                  <FormControl>
                    <Input type="number" min={1} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Button type="submit" disabled={update.isPending}>
              {update.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save Settings
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}
