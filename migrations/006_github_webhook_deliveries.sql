-- Durable, idempotent intake for GitHub webhook deliveries.

CREATE TABLE github_webhook_deliveries (
    delivery_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload BLOB NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TEXT NOT NULL,
    last_error TEXT,
    processing_started_at TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_github_webhook_deliveries_due
    ON github_webhook_deliveries(status, next_attempt_at, created_at);
