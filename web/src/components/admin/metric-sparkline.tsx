import { cn } from "@/lib/utils";
import type { MetricPoint } from "@/types/models";

interface MetricSparklineProps {
  points: MetricPoint[];
  className?: string;
}

export function MetricSparkline({
  points,
  className,
}: MetricSparklineProps) {
  if (points.length < 2) {
    return (
      <div
        className={cn(
          "flex h-20 items-center justify-center rounded-md border border-dashed text-xs text-muted-foreground",
          className,
        )}
      >
        Waiting for more samples
      </div>
    );
  }

  const values = points.map((point) => point.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const path = points
    .map((point, index) => {
      const x = (index / (points.length - 1)) * 100;
      const y = 100 - ((point.value - min) / range) * 100;
      return `${index === 0 ? "M" : "L"} ${x} ${y}`;
    })
    .join(" ");

  return (
    <svg
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      className={cn("h-20 w-full overflow-visible", className)}
      aria-hidden="true"
    >
      <path
        d={path}
        fill="none"
        stroke="currentColor"
        strokeWidth="3"
        vectorEffect="non-scaling-stroke"
        className="text-primary"
      />
    </svg>
  );
}
