export interface LogEvent {
  line: number;
  message: string;
  timestamp: string;
}

export interface StatusEvent {
  status: string;
  phase: string;
}

export interface ErrorEvent {
  message: string;
}

export interface CompleteEvent {
  status: "ready" | "failed" | "cancelled";
  duration_ms: number;
  url?: string;
  artifact_size?: number;
  error?: string;
}

export type SSEEventType = "log" | "status" | "error" | "done" | "complete";

export interface SSEMessage {
  id?: string;
  event: SSEEventType;
  data: LogEvent | StatusEvent | ErrorEvent | CompleteEvent;
}
