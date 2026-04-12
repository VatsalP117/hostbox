import { truncateSha } from "@/lib/utils";
import { GitCommit } from "lucide-react";

interface CommitInfoProps {
  sha: string;
  message?: string | null;
  author?: string | null;
}

export function CommitInfo({ sha, message, author }: CommitInfoProps) {
  return (
    <div className="flex items-center gap-2 text-sm">
      <GitCommit className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
      <code className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
        {truncateSha(sha)}
      </code>
      {message && (
        <span className="text-muted-foreground truncate max-w-[300px]">
          {message.length > 60 ? `${message.slice(0, 60)}…` : message}
        </span>
      )}
      {author && (
        <span className="text-muted-foreground text-xs shrink-0">
          by {author}
        </span>
      )}
    </div>
  );
}
