import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FrameworkSelect } from "@/components/projects/framework-select";
import { nodeVersions } from "@/lib/constants";
import type { Framework } from "@/types/models";

const buildSettingsSchema = z.object({
  name: z.string().min(1, "Project name is required"),
  framework: z.string().optional(),
  build_command: z.string().optional(),
  install_command: z.string().optional(),
  output_directory: z.string().optional(),
  root_directory: z.string().optional(),
  node_version: z.string().optional(),
});

export type BuildSettingsValues = z.infer<typeof buildSettingsSchema>;

interface BuildSettingsFormProps {
  defaultValues?: Partial<BuildSettingsValues>;
  onSubmit: (values: BuildSettingsValues) => void;
  submitLabel?: string;
  isPending?: boolean;
  children?: React.ReactNode;
}

export function BuildSettingsForm({
  defaultValues,
  onSubmit,
  children,
}: BuildSettingsFormProps) {
  const form = useForm<BuildSettingsValues>({
    resolver: zodResolver(buildSettingsSchema),
    defaultValues: {
      name: "",
      framework: "",
      build_command: "",
      install_command: "",
      output_directory: "",
      root_directory: "",
      node_version: "20",
      ...defaultValues,
    },
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Project Name</FormLabel>
              <FormControl>
                <Input placeholder="my-project" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="framework"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Framework</FormLabel>
              <FormControl>
                <FrameworkSelect
                  value={field.value as Framework | ""}
                  onChange={field.onChange}
                />
              </FormControl>
              <FormDescription>
                Auto-detect will analyze your project to determine the framework.
              </FormDescription>
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="build_command"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Build Command</FormLabel>
              <FormControl>
                <Input placeholder="npm run build" {...field} />
              </FormControl>
              <FormDescription>
                Leave empty to use the framework default.
              </FormDescription>
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="install_command"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Install Command</FormLabel>
              <FormControl>
                <Input placeholder="npm install" {...field} />
              </FormControl>
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="output_directory"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Output Directory</FormLabel>
              <FormControl>
                <Input placeholder="dist" {...field} />
              </FormControl>
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="root_directory"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Root Directory</FormLabel>
              <FormControl>
                <Input placeholder="./" {...field} />
              </FormControl>
              <FormDescription>
                The directory where your project source lives.
              </FormDescription>
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="node_version"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Node.js Version</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {nodeVersions.map((v) => (
                    <SelectItem key={v} value={v}>
                      Node.js {v}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormItem>
          )}
        />

        {children}
      </form>
    </Form>
  );
}
