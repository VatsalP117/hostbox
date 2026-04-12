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
import { ConfirmationDialog } from "@/components/shared/confirmation-dialog";
import { DnsInstructions } from "@/components/domains/dns-instructions";
import { TimeAgo } from "@/components/shared/time-ago";
import { useVerifyDomain, useDeleteDomain } from "@/hooks/use-domains";
import { getApiErrorMessage } from "@/lib/utils";
import type { Domain } from "@/types/models";
import {
  CheckCircle,
  Clock,
  Loader2,
  RefreshCw,
  Trash2,
  Globe,
} from "lucide-react";

interface DomainCardProps {
  domain: Domain;
  projectId: string;
}

export function DomainCard({ domain, projectId }: DomainCardProps) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const verify = useVerifyDomain();
  const deleteDomain = useDeleteDomain();

  const handleVerify = () => {
    verify.mutate(
      { projectId, domainId: domain.id },
      {
        onSuccess: () => toast.success("Domain verified!"),
        onError: (err) => toast.error(getApiErrorMessage(err)),
      },
    );
  };

  const handleDelete = () => {
    deleteDomain.mutate(
      { projectId, domainId: domain.id },
      {
        onSuccess: () => {
          toast.success("Domain removed");
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
              <Globe className="h-4 w-4 text-muted-foreground" />
              <CardTitle className="text-base">{domain.domain}</CardTitle>
              {domain.verified ? (
                <Badge
                  variant="outline"
                  className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                >
                  <CheckCircle className="mr-1 h-3 w-3" />
                  Verified
                </Badge>
              ) : (
                <Badge variant="outline" className="text-amber-600">
                  <Clock className="mr-1 h-3 w-3" />
                  Pending
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-2">
              {!domain.verified && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleVerify}
                  disabled={verify.isPending}
                >
                  {verify.isPending ? (
                    <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <RefreshCw className="mr-2 h-3.5 w-3.5" />
                  )}
                  Verify
                </Button>
              )}
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
        <CardContent>
          <div className="space-y-3">
            {!domain.verified && <DnsInstructions hostname={domain.domain} />}
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              <span>Added <TimeAgo date={domain.created_at} /></span>
              {domain.last_checked_at && (
                <span>
                  Last checked <TimeAgo date={domain.last_checked_at} />
                </span>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      <ConfirmationDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Remove Domain"
        description={`Are you sure you want to remove "${domain.domain}"? This cannot be undone.`}
        confirmLabel="Remove"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteDomain.isPending}
      />
    </>
  );
}
