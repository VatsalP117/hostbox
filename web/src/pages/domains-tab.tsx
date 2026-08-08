import { useState } from "react";
import { useDomains } from "@/hooks/use-domains";
import { DomainList } from "@/components/domains/domain-list";
import { AddDomainForm } from "@/components/domains/add-domain-form";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

interface DomainsTabProps {
  projectId: string;
}

export function DomainsTab({ projectId }: DomainsTabProps) {
  const [addOpen, setAddOpen] = useState(false);
  const { data, isLoading } = useDomains(projectId);

  return (
    <div className="space-y-5">
      <div className="flex flex-col justify-between gap-3 border-b border-border pb-4 sm:flex-row sm:items-center">
        <div><h2 className="text-base font-medium">Domains</h2><p className="mt-1 text-xs text-muted-foreground">Route production traffic and manage automatic TLS.</p></div>
        <Button onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add domain
        </Button>
      </div>

      <DomainList
        domains={data?.domains}
        isLoading={isLoading}
        projectId={projectId}
      />

      <AddDomainForm
        open={addOpen}
        onOpenChange={setAddOpen}
        projectId={projectId}
      />
    </div>
  );
}
