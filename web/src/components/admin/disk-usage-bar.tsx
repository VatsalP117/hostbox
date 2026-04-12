import { Progress } from "@/components/ui/progress";

interface DiskUsageBarProps {
  usedBytes: number;
  totalBytes: number;
  deploymentBytes: number;
}

export function DiskUsageBar({
  usedBytes,
  totalBytes,
}: DiskUsageBarProps) {
  const percentage = totalBytes > 0 ? (usedBytes / totalBytes) * 100 : 0;

  return (
    <Progress
      value={percentage}
      className="h-2"
    />
  );
}
