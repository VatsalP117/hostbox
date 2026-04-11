-- Track lock file hash for build cache invalidation
ALTER TABLE projects ADD COLUMN lock_file_hash TEXT DEFAULT '';

-- Track the detected package manager for cache invalidation
ALTER TABLE projects ADD COLUMN detected_package_manager TEXT DEFAULT '';

-- Index for finding active production deployment per project
CREATE INDEX IF NOT EXISTS idx_deployments_production
    ON deployments(project_id, is_production, status)
    WHERE is_production = TRUE AND status = 'ready';

-- Index for deduplication queries (find queued/building for project+branch)
CREATE INDEX IF NOT EXISTS idx_deployments_dedup
    ON deployments(project_id, branch, status)
    WHERE status IN ('queued', 'building');
