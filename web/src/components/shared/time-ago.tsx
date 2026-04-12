import { timeAgo } from "@/lib/date";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { formatDate } from "@/lib/date";

interface TimeAgoProps {
  date: string;
}

export function TimeAgo({ date }: TimeAgoProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            {timeAgo(date)}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{formatDate(date)}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
