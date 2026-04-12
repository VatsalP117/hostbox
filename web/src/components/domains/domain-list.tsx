import { DomainCard } from "@/components/domains/domain-card";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import type { Domain } from "@/types/models";
import { Globe } from "lucide-react";

interface DomainListProps {
  domains: Domain[] | undefined;
  isLoading: boolean;
  projectId: string;
}

export function DomainList({ domains, isLoading, projectId }: DomainListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 2 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (!domains?.length) {
    return (
      <EmptyState
        icon={Globe}
        title="No domains"
        description="Add a custom domain to your project."
      />
    );
  }

  return (
    <div className="space-y-3">
      {domains.map((domain) => (
        <DomainCard key={domain.id} domain={domain} projectId={projectId} />
      ))}
    </div>
  );
}
