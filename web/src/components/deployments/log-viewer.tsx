import { useRef, useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { LogEvent } from "@/types/events";
import { ArrowDown, Copy, Download, Maximize2 } from "lucide-react";
import { toast } from "sonner";

interface LogViewerProps {
  lines: LogEvent[];
  isStreaming?: boolean;
}

// ANSI color codes for terminal-style coloring
const ansiColors: Record<string, string> = {
  "30": "text-gray-900",     // Black
  "31": "text-[hsl(0,84%,60%)]",     // Red
  "32": "text-[hsl(142,76%,56%)]",  // Green
  "33": "text-[hsl(38,92%,60%)]",   // Yellow
  "34": "text-[hsl(217,91%,65%)]",   // Blue
  "35": "text-[hsl(292,84%,65%)]",   // Magenta
  "36": "text-[hsl(187,85%,53%)]",   // Cyan
  "37": "text-[hsl(30,4%,90%)]",     // White
  "90": "text-[hsl(30,4%,50%)]",     // Bright Black (Gray)
  "91": "text-[hsl(0,84%,70%)]",     // Bright Red
  "92": "text-[hsl(142,76%,66%)]",   // Bright Green
  "93": "text-[hsl(38,92%,70%)]",    // Bright Yellow
  "94": "text-[hsl(217,91%,75%)]",   // Bright Blue
  "95": "text-[hsl(292,84%,75%)]",   // Bright Magenta
  "96": "text-[hsl(187,85%,63%)]",   // Bright Cyan
  "97": "text-[hsl(30,4%,100%)]",    // Bright White
};

// Parse ANSI codes and convert to styled HTML
function parseAnsi(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  const regex = /\x1b\[([0-9;]*)m/g;
  let lastIndex = 0;
  let match;
  let currentClass = "";
  let key = 0;

  while ((match = regex.exec(text)) !== null) {
    const before = text.slice(lastIndex, match.index);
    if (before) {
      parts.push(
        <span key={key++} className={currentClass}>
          {before}
        </span>
      );
    }

    const codes = match[1].split(";");
    for (const code of codes) {
      if (code === "0") {
        currentClass = "";
      } else if (ansiColors[code]) {
        currentClass = ansiColors[code];
      } else if (code === "1") {
        currentClass += " font-bold";
      }
    }

    lastIndex = regex.lastIndex;
  }

  const remaining = text.slice(lastIndex);
  if (remaining) {
    parts.push(
      <span key={key++} className={currentClass}>
        {remaining}
      </span>
    );
  }

  return parts.length > 0 ? parts : [text];
}

// Strip ANSI codes for plain text copy
function stripAnsi(text: string): string {
  return text.replace(/\x1b\[[0-9;]*m/g, "");
}

export function LogViewer({ lines, isStreaming = false }: LogViewerProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  // Auto-scroll to bottom when new lines arrive
  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [lines, autoScroll]);

  // Handle scroll to detect if user is at bottom
  const handleScroll = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  // Copy logs to clipboard
  const handleCopyLogs = () => {
    const fullText = lines.map((l) => stripAnsi(l.message)).join("\n");
    navigator.clipboard.writeText(fullText);
    toast.success("Logs copied to clipboard");
  };

  // Download logs as file
  const handleDownloadLogs = () => {
    const fullText = lines.map((l) => stripAnsi(l.message)).join("\n");
    const blob = new Blob([fullText], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `build-logs-${new Date().toISOString().slice(0, 10)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
    toast.success("Logs downloaded");
  };

  // Get log level from message
  const getLogLevel = (message: string): { level: string; className: string } => {
    const upper = message.toUpperCase();
    if (upper.includes("[ERROR]") || upper.includes("ERROR:") || upper.includes("FAILED")) {
      return { level: "ERROR", className: "text-[hsl(0,84%,60%)]" };
    }
    if (upper.includes("[WARN]") || upper.includes("WARNING:")) {
      return { level: "WARN", className: "text-[hsl(38,92%,60%)]" };
    }
    if (upper.includes("[INFO]") || upper.includes("INFO:")) {
      return { level: "INFO", className: "text-[hsl(217,91%,65%)]" };
    }
    if (upper.includes("[SUCCESS]") || upper.includes("SUCCESS:")) {
      return { level: "SUCCESS", className: "text-[hsl(142,76%,56%)]" };
    }
    return { level: "LOG", className: "text-[hsl(30,4%,70%)]" };
  };

  // Format timestamp
  const formatTimestamp = (timestamp?: string): string => {
    if (!timestamp) return "";
    const date = new Date(timestamp);
    return date.toLocaleTimeString("en-US", {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  };

  return (
    <div className="bg-[hsl(0,0%,11%)] rounded-xl overflow-hidden shadow-[0_4px_48px_-12px_rgba(229,226,225,0.05)] border border-[hsl(220,10%,28%)]/10">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[hsl(220,10%,28%)]/15 flex items-center justify-between bg-[hsl(0,0%,16%)]/50 backdrop-blur-md">
        <div className="flex items-center gap-3">
          <h3 className="font-headline font-bold text-lg">Build Logs</h3>
          {isStreaming && (
            <span className="flex items-center gap-1.5 text-xs text-[hsl(142,76%,56%)]">
              <span className="h-1.5 w-1.5 rounded-full bg-[hsl(142,76%,56%)] animate-pulse" />
              Live
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleDownloadLogs}
            className="text-[hsl(30,4%,70%)] hover:text-[hsl(30,4%,90%)] transition-colors p-1.5 rounded-lg hover:bg-[hsl(0,0%,16%)]"
            title="Download logs"
          >
            <Download className="h-4 w-4" />
          </button>
          <button
            onClick={handleCopyLogs}
            className="text-[hsl(30,4%,70%)] hover:text-[hsl(30,4%,90%)] transition-colors p-1.5 rounded-lg hover:bg-[hsl(0,0%,16%)]"
            title="Copy logs"
          >
            <Copy className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Log Content */}
      <div className="relative">
        <ScrollArea
          className="h-[500px] bg-[#0a0a0a]"
          ref={containerRef}
          onScrollCapture={handleScroll}
        >
          <div className="p-4 font-[JetBrains_Mono] text-xs leading-relaxed">
            {lines.length === 0 && (
              <p className="text-[hsl(30,4%,50%)] italic">Waiting for logs...</p>
            )}
            {lines.map((line, i) => {
              const { level, className: levelClass } = getLogLevel(line.message);
              const timestamp = formatTimestamp(line.timestamp);
              
              return (
                <div 
                  key={i} 
                  className="flex gap-3 hover:bg-[hsl(0,0%,16%)]/30 transition-colors rounded px-1 -mx-1"
                >
                  {/* Line Number */}
                  <span className="select-none text-[hsl(220,10%,28%)] w-8 text-right shrink-0">
                    {i + 1}
                  </span>
                  
                  {/* Timestamp */}
                  {timestamp && (
                    <span className="select-none text-[hsl(220,10%,28%)] shrink-0">
                      {timestamp}
                    </span>
                  )}
                  
                  {/* Log Level */}
                  <span className={`select-none shrink-0 ${levelClass}`}>
                    [{level}]
                  </span>
                  
                  {/* Message */}
                  <span className="whitespace-pre-wrap break-all text-[hsl(220,10%,75%)]">
                    {parseAnsi(line.message)}
                  </span>
                </div>
              );
            })}
            <div ref={bottomRef} />
          </div>
        </ScrollArea>

        {/* Follow Logs Button */}
        {!autoScroll && (
          <Button
            variant="secondary"
            size="sm"
            className="absolute bottom-4 right-4 bg-[hsl(0,0%,16%)] hover:bg-[hsl(0,0%,20%)] text-[hsl(30,4%,90%)] border border-[hsl(220,10%,28%)]/30 opacity-90 hover:opacity-100"
            onClick={() => {
              setAutoScroll(true);
              bottomRef.current?.scrollIntoView({ behavior: "smooth" });
            }}
          >
            <ArrowDown className="mr-1.5 h-3.5 w-3.5" />
            Follow logs
          </Button>
        )}
      </div>
    </div>
  );
}
