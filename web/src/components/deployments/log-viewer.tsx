import { useRef, useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/shared/copy-button";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { LogEvent } from "@/types/events";
import { ArrowDown } from "lucide-react";

interface LogViewerProps {
  lines: LogEvent[];
  isStreaming?: boolean;
}

export function LogViewer({ lines, isStreaming = false }: LogViewerProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [lines, autoScroll]);

  const handleScroll = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  const fullText = lines.map((l) => l.message).join("\n");

  return (
    <div className="relative rounded-lg border bg-zinc-950 text-zinc-100">
      <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-2">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-zinc-400">Build Logs</span>
          {isStreaming && (
            <span className="flex items-center gap-1 text-xs text-green-400">
              <span className="h-1.5 w-1.5 rounded-full bg-green-400 animate-pulse" />
              Live
            </span>
          )}
        </div>
        <CopyButton value={fullText} />
      </div>

      <ScrollArea
        className="h-[500px]"
        ref={containerRef}
        onScrollCapture={handleScroll}
      >
        <div className="p-4 font-mono text-xs leading-relaxed">
          {lines.length === 0 && (
            <p className="text-zinc-500">Waiting for logs…</p>
          )}
          {lines.map((line, i) => (
            <div key={i} className="flex gap-3 hover:bg-zinc-900/50">
              <span className="select-none text-zinc-600 w-8 text-right shrink-0">
                {line.line || i + 1}
              </span>
              <span className="whitespace-pre-wrap break-all">
                {line.message}
              </span>
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      </ScrollArea>

      {!autoScroll && (
        <Button
          variant="secondary"
          size="sm"
          className="absolute bottom-4 right-4 opacity-80 hover:opacity-100"
          onClick={() => {
            setAutoScroll(true);
            bottomRef.current?.scrollIntoView({ behavior: "smooth" });
          }}
        >
          <ArrowDown className="mr-1 h-3 w-3" />
          Follow logs
        </Button>
      )}
    </div>
  );
}
