import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState, useCallback } from "react";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/constants";
import { useAuthStore } from "@/stores/auth-store";
import type { LogsResponse } from "@/types/api";
import type { LogEvent, CompleteEvent } from "@/types/events";

interface UseDeploymentLogsOptions {
  enabled?: boolean;
  liveEnabled?: boolean;
}

interface UseDeploymentLogsResult {
  lines: LogEvent[];
  status: string | null;
  error: string | null;
  complete: CompleteEvent | null;
  isConnected: boolean;
  isComplete: boolean;
  isLoadingHistory: boolean;
}

function mergeLogLines(existing: LogEvent[], incoming: LogEvent[]): LogEvent[] {
  if (incoming.length === 0) {
    return existing;
  }

  const mergedByLine = new Map<number, LogEvent>();
  for (const line of existing) {
    mergedByLine.set(line.line, line);
  }

  let changed = false;
  for (const line of incoming) {
    const current = mergedByLine.get(line.line);
    if (!current) {
      mergedByLine.set(line.line, line);
      changed = true;
      continue;
    }

    const next: LogEvent = {
      ...current,
      ...line,
      timestamp: line.timestamp ?? current.timestamp,
    };

    if (
      current.message !== next.message ||
      current.timestamp !== next.timestamp
    ) {
      mergedByLine.set(line.line, next);
      changed = true;
    }
  }

  if (!changed) {
    return existing;
  }

  return Array.from(mergedByLine.values()).sort((a, b) => a.line - b.line);
}

export function useDeploymentLogs(
  deploymentId: string,
  options?: UseDeploymentLogsOptions,
): UseDeploymentLogsResult {
  const queryClient = useQueryClient();
  const [lines, setLines] = useState<LogEvent[]>([]);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [complete, setComplete] = useState<CompleteEvent | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isComplete, setIsComplete] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);
  const maxLineRef = useRef(0);

  const enabled = options?.enabled !== false;
  const liveEnabled = options?.liveEnabled !== false;

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const appendLines = useCallback((incoming: LogEvent[]) => {
    if (incoming.length === 0) {
      return;
    }

    setLines((prev) => {
      const merged = mergeLogLines(prev, incoming);
      if (merged.length > 0) {
        maxLineRef.current = merged[merged.length - 1].line;
      }
      return merged;
    });
  }, []);

  useEffect(() => {
    cleanup();
    setLines([]);
    setStatus(null);
    setError(null);
    setComplete(null);
    setIsComplete(false);
    maxLineRef.current = 0;
  }, [deploymentId, cleanup]);

  const historyQuery = useQuery({
    queryKey: queryKeys.deploymentLogs(deploymentId),
    queryFn: () =>
      api.get<LogsResponse>(`/deployments/${deploymentId}/logs`, { limit: 5000 }),
    enabled: !!deploymentId && enabled,
  });

  useEffect(() => {
    if (!historyQuery.data) {
      return;
    }

    appendLines(
      historyQuery.data.lines.map((message, index) => ({
        line: index + 1,
        message,
      })),
    );
  }, [historyQuery.data, appendLines]);

  useEffect(() => {
    if (!deploymentId || !enabled || !liveEnabled) {
      cleanup();
      return;
    }

    const token = useAuthStore.getState().accessToken;
    const params = new URLSearchParams();
    if (token) params.set("token", token);
    if (maxLineRef.current > 0) {
      params.set("offset", String(maxLineRef.current));
    }

    const url = `/api/v1/deployments/${deploymentId}/logs/stream?${params.toString()}`;
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    eventSource.addEventListener("log", (e) => {
      try {
        const parsed = JSON.parse(e.data) as Partial<LogEvent>;
        appendLines([
          {
            line: parsed.line ?? maxLineRef.current + 1,
            message: parsed.message ?? e.data,
            timestamp: parsed.timestamp,
          },
        ]);
      } catch {
        appendLines([
          {
            line: maxLineRef.current + 1,
            message: e.data,
            timestamp: new Date().toISOString(),
          },
        ]);
      }
    });

    eventSource.addEventListener("status", (e) => {
      try {
        const data = JSON.parse(e.data) as { status: string; phase: string };
        setStatus(data.phase || data.status);
      } catch {
        // ignore
      }
    });

    eventSource.addEventListener("error", (e) => {
      if (e instanceof MessageEvent) {
        try {
          const data = JSON.parse(e.data) as { message: string };
          setError(data.message);
        } catch {
          // ignore
        }
      }
    });

    const handleComplete = (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as CompleteEvent;
        setComplete(data);
        setIsComplete(true);
        void queryClient.invalidateQueries({
          queryKey: queryKeys.deployment(deploymentId),
        });
        void queryClient.invalidateQueries({
          queryKey: queryKeys.deploymentLogs(deploymentId),
        });
        cleanup();
      } catch {
        // ignore
      }
    };

    eventSource.addEventListener("complete", handleComplete);
    eventSource.addEventListener("done", handleComplete);

    eventSource.onerror = () => {
      setIsConnected(false);
    };

    return cleanup;
  }, [appendLines, cleanup, deploymentId, enabled, liveEnabled, queryClient]);

  return {
    lines,
    status,
    error,
    complete,
    isConnected,
    isComplete,
    isLoadingHistory: historyQuery.isLoading,
  };
}
