import { Skeleton } from "@/components/ui/skeleton";

export function LoadingPage() {
  return (
    <div className="flex h-screen w-full items-center justify-center">
      <div className="space-y-4 w-full max-w-md px-4">
        <Skeleton className="mx-auto h-10 w-10 rounded-lg" />
        <Skeleton className="mx-auto h-4 w-32" />
        <Skeleton className="mx-auto h-3 w-48" />
      </div>
    </div>
  );
}
