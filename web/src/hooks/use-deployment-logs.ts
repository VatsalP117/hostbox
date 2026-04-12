import { useEffect, useRef, useState, useCallback } from "react";
import { useAuthStore } from "@/stores/auth-store";
import type { LogEvent, CompleteEvent, SSEEventType } from "@/types/events";

interface UseDeploymentLogsOptions {
  enabled?: boolean;
}

interface UseDeploymentLogsResult {
  lines: LogEvent[];
  status: string | null;
  error: string | null;
  complete: CompleteEvent | null;
  isConnected: boolean;
  isComplete: boolean;
}

export function useDeploymentLogs(
  deploymentId: string,
  options?: UseDeploymentLogsOptions,
): UseDeploymentLogsResult {
  const [lines, setLines] = useState<LogEvent[]>([]);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [complete, setComplete] = useState<CompleteEvent | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isComplete, setIsComplete] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  const enabled = options?.enabled !== false;

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsConnected(false);
  }, []);

  useEffect(() => {
    if (!deploymentId || !enabled) return;

    const token = useAuthStore.getState().accessToken;
    const params = new URLSearchParams();
    if (token) params.set("token", token);

    const url = `/api/v1/deployments/${deploymentId}/logs/stream?${params.toString()}`;
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    eventSource.addEventListener("log", (e) => {
      try {
        const data = JSON.parse(e.data) as LogEvent;
        setLines((prev) => [...prev, data]);
      } catch {
        // ignore malformed events
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

    eventSource.addEventListener("complete", (e) => {
      try {
        const data = JSON.parse(e.data) as CompleteEvent;
        setComplete(data);
        setIsComplete(true);
        cleanup();
      } catch {
        // ignore
      }
    });

    eventSource.onerror = () => {
      setIsConnected(false);
    };

    return cleanup;
  }, [deploymentId, enabled, cleanup]);

  return { lines, status, error, complete, isConnected, isComplete };
}
