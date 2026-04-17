import { useState } from "react";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { useBulkImportEnvVars } from "@/hooks/use-env-vars";
import { parseEnvFile, getApiErrorMessage } from "@/lib/utils";
import { Loader2, Upload } from "lucide-react";

interface EnvVarImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
}

export function EnvVarImportDialog({
  open,
  onOpenChange,
  projectId,
}: EnvVarImportDialogProps) {
  const [content, setContent] = useState("");
  const bulkImport = useBulkImportEnvVars(projectId);
  const parsed = content ? parseEnvFile(content) : [];

  const handleImport = () => {
    if (!parsed.length) return;

    bulkImport.mutate(
      { env_vars: parsed, scope: "all" },
      {
        onSuccess: (data) => {
          toast.success(
            `Imported ${data.created} new, updated ${data.updated} existing`,
          );
          setContent("");
          onOpenChange(false);
        },
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (event) => {
      setContent(event.target?.result as string);
    };
    reader.readAsText(file);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Import .env File</DialogTitle>
          <DialogDescription>
            Paste your .env file contents or upload a file.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <input
              type="file"
              accept=".env,.env.*"
              className="hidden"
              id="env-file-upload"
              onChange={handleFileUpload}
            />
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                document.getElementById("env-file-upload")?.click()
              }
            >
              <Upload className="mr-2 h-3.5 w-3.5" />
              Upload .env file
            </Button>
          </div>

          <Textarea
            placeholder={`# Paste your .env contents\nAPI_KEY=your-api-key\nDATABASE_URL=postgres://...`}
            className="font-mono text-xs h-40"
            value={content}
            onChange={(e) => setContent(e.target.value)}
          />

          {parsed.length > 0 && (
            <div className="rounded-md border p-3 space-y-2">
              <p className="text-xs font-medium text-muted-foreground">
                Preview ({parsed.length} variables)
              </p>
              <div className="flex flex-wrap gap-1">
                {parsed.map((v, i) => (
                  <Badge key={i} variant="secondary" className="text-xs font-mono">
                    {v.key}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            onClick={handleImport}
            disabled={!parsed.length || bulkImport.isPending}
          >
            {bulkImport.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Import {parsed.length} Variables
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
