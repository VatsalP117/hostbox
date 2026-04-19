import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from "@/components/ui/form";
import { useAdminSettings, useUpdateAdminSettings } from "@/hooks/use-admin";
import { getApiErrorMessage } from "@/lib/utils";
import { Loader2, Save, Users, Box, Clock, Archive } from "lucide-react";

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
      <div className="bg-surface-container-low rounded-xl p-6 space-y-4">
        <Skeleton className="h-8 w-48 bg-surface-container" />
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full rounded-lg bg-surface-container" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-surface-container-low rounded-xl overflow-hidden">
      <div className="p-6 border-b border-outline-variant/15">
        <h3 className="font-headline text-lg font-bold text-foreground">Platform Settings</h3>
        <p className="font-label text-xs text-muted-foreground mt-1 uppercase tracking-wider">
          Configure global platform settings
        </p>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="p-6 space-y-6">
          {/* User Registration Toggle */}
          <FormField
            control={form.control}
            name="registration_enabled"
            render={({ field }) => (
              <FormItem className="flex items-center justify-between p-4 bg-surface-container rounded-lg">
                <div className="space-y-0.5">
                  <div className="flex items-center space-x-2">
                    <Users className="h-4 w-4 text-muted-foreground" />
                    <FormLabel className="font-body text-sm font-medium text-foreground">
                      User Registration
                    </FormLabel>
                  </div>
                  <FormDescription className="font-body text-xs text-muted-foreground">
                    Allow new users to register on the platform
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    className="data-[state=checked]:bg-primary"
                  />
                </FormControl>
              </FormItem>
            )}
          />

          {/* Settings Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Max Projects */}
            <FormField
              control={form.control}
              name="max_projects"
              render={({ field }) => (
                <FormItem className="p-4 bg-surface-container rounded-lg space-y-3">
                  <div className="flex items-center space-x-2">
                    <Box className="h-4 w-4 text-muted-foreground" />
                    <FormLabel className="font-body text-sm font-medium text-foreground">
                      Max Projects per User
                    </FormLabel>
                  </div>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      {...field}
                      className="bg-surface-container-low border-outline-variant/30 font-body text-sm"
                    />
                  </FormControl>
                  <FormMessage className="font-body text-xs" />
                </FormItem>
              )}
            />

            {/* Max Concurrent Builds */}
            <FormField
              control={form.control}
              name="max_concurrent_builds"
              render={({ field }) => (
                <FormItem className="p-4 bg-surface-container rounded-lg space-y-3">
                  <div className="flex items-center space-x-2">
                    <Clock className="h-4 w-4 text-muted-foreground" />
                    <FormLabel className="font-body text-sm font-medium text-foreground">
                      Max Concurrent Builds
                    </FormLabel>
                  </div>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      {...field}
                      className="bg-surface-container-low border-outline-variant/30 font-body text-sm"
                    />
                  </FormControl>
                  <FormMessage className="font-body text-xs" />
                </FormItem>
              )}
            />
          </div>

          {/* Artifact Retention */}
          <FormField
            control={form.control}
            name="artifact_retention_days"
            render={({ field }) => (
              <FormItem className="p-4 bg-surface-container rounded-lg space-y-3">
                <div className="flex items-center space-x-2">
                  <Archive className="h-4 w-4 text-muted-foreground" />
                  <FormLabel className="font-body text-sm font-medium text-foreground">
                    Artifact Retention (days)
                  </FormLabel>
                </div>
                <FormControl>
                  <Input
                    type="number"
                    min={1}
                    {...field}
                    className="bg-surface-container-low border-outline-variant/30 font-body text-sm max-w-xs"
                  />
                </FormControl>
                <FormDescription className="font-body text-xs text-muted-foreground">
                  Number of days to keep build artifacts before automatic deletion
                </FormDescription>
                <FormMessage className="font-body text-xs" />
              </FormItem>
            )}
          />

          {/* Save Button */}
          <div className="pt-4 border-t border-outline-variant/15">
            <Button
              type="submit"
              disabled={update.isPending}
              className="gradient-btn rounded-xl px-6 py-2.5 font-label text-sm font-semibold"
            >
              {update.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              <Save className="mr-2 h-4 w-4" />
              Save Settings
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}
