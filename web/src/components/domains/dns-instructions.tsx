import { CopyButton } from "@/components/shared/copy-button";

interface DnsInstructionsProps {
  hostname: string;
}

export function DnsInstructions({ hostname }: DnsInstructionsProps) {
  const isSubdomain = hostname.split(".").length > 2;

  return (
    <div className="rounded-md border bg-muted/50 p-3 space-y-3">
      <p className="text-xs font-medium text-muted-foreground">
        Configure your DNS records:
      </p>

      {isSubdomain ? (
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <p className="text-xs font-medium">CNAME Record</p>
            <CopyButton value={hostname} />
          </div>
          <div className="grid grid-cols-3 gap-2 text-xs">
            <div>
              <span className="text-muted-foreground">Type</span>
              <p className="font-mono">CNAME</p>
            </div>
            <div>
              <span className="text-muted-foreground">Name</span>
              <p className="font-mono">{hostname.split(".")[0]}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Value</span>
              <p className="font-mono">cname.hostbox.dev</p>
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <p className="text-xs font-medium">A Record</p>
            <CopyButton value="76.76.21.21" />
          </div>
          <div className="grid grid-cols-3 gap-2 text-xs">
            <div>
              <span className="text-muted-foreground">Type</span>
              <p className="font-mono">A</p>
            </div>
            <div>
              <span className="text-muted-foreground">Name</span>
              <p className="font-mono">@</p>
            </div>
            <div>
              <span className="text-muted-foreground">Value</span>
              <p className="font-mono">76.76.21.21</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
