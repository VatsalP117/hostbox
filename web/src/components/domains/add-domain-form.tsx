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
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useAddDomain } from "@/hooks/use-domains";
import { getApiErrorMessage } from "@/lib/utils";
import { Loader2 } from "lucide-react";

const schema = z.object({
  hostname: z
    .string()
    .min(1, "Domain is required")
    .regex(
      /^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$/i,
      "Enter a valid domain (e.g. app.example.com)",
    ),
});

type FormValues = z.infer<typeof schema>;

interface AddDomainFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
}

export function AddDomainForm({
  open,
  onOpenChange,
  projectId,
}: AddDomainFormProps) {
  const addDomain = useAddDomain(projectId);
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { hostname: "" },
  });

  const onSubmit = (values: FormValues) => {
    addDomain.mutate(values, {
      onSuccess: () => {
        toast.success("Domain added. Configure your DNS records to verify it.");
        form.reset();
        onOpenChange(false);
      },
      onError: (err) => toast.error(getApiErrorMessage(err)),
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Domain</DialogTitle>
          <DialogDescription>
            Add a custom domain to your project.
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="hostname"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Domain</FormLabel>
                  <FormControl>
                    <Input placeholder="app.example.com" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={addDomain.isPending}>
                {addDomain.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Add Domain
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
