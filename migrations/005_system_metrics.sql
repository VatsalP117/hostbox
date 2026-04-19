-- Persist lightweight system metric snapshots for operator monitoring trends.

CREATE TABLE system_metric_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cpu_usage_percent REAL NOT NULL DEFAULT 0,
    load1 REAL NOT NULL DEFAULT 0,
    load5 REAL NOT NULL DEFAULT 0,
    load15 REAL NOT NULL DEFAULT 0,
    memory_used_bytes INTEGER NOT NULL DEFAULT 0,
    memory_total_bytes INTEGER NOT NULL DEFAULT 0,
    memory_available_bytes INTEGER NOT NULL DEFAULT 0,
    memory_usage_percent REAL NOT NULL DEFAULT 0,
    disk_used_bytes INTEGER NOT NULL DEFAULT 0,
    disk_total_bytes INTEGER NOT NULL DEFAULT 0,
    disk_available_bytes INTEGER NOT NULL DEFAULT 0,
    disk_usage_percent REAL NOT NULL DEFAULT 0,
    active_builds INTEGER NOT NULL DEFAULT 0,
    queued_builds INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_system_metric_snapshots_created_at
    ON system_metric_snapshots(created_at);
