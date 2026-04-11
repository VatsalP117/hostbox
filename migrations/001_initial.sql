-- 001_initial.sql
-- Hostbox initial schema

-- Users
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    is_admin INTEGER NOT NULL DEFAULT 0,
    email_verified INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Sessions (refresh tokens)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    user_agent TEXT,
    ip_address TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Projects
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    github_repo TEXT,
    github_installation_id INTEGER,
    production_branch TEXT NOT NULL DEFAULT 'main',
    framework TEXT,
    build_command TEXT,
    install_command TEXT,
    output_directory TEXT,
    root_directory TEXT DEFAULT '/',
    node_version TEXT DEFAULT '20',
    auto_deploy INTEGER NOT NULL DEFAULT 1,
    preview_deployments INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_projects_owner_id ON projects(owner_id);
CREATE INDEX idx_projects_github_repo ON projects(github_repo);

-- Deployments
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    commit_sha TEXT NOT NULL,
    commit_message TEXT,
    commit_author TEXT,
    branch TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    is_production INTEGER NOT NULL DEFAULT 0,
    deployment_url TEXT,
    artifact_path TEXT,
    artifact_size_bytes INTEGER,
    log_path TEXT,
    error_message TEXT,
    is_rollback INTEGER NOT NULL DEFAULT 0,
    rollback_source_id TEXT REFERENCES deployments(id),
    github_pr_number INTEGER,
    build_duration_ms INTEGER,
    started_at TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_deployments_project_id ON deployments(project_id);
CREATE INDEX idx_deployments_project_status ON deployments(project_id, status);
CREATE INDEX idx_deployments_project_branch ON deployments(project_id, branch);
CREATE INDEX idx_deployments_created_at ON deployments(created_at);

-- Custom domains
CREATE TABLE domains (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    domain TEXT UNIQUE NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0,
    verified_at TEXT,
    last_checked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_domains_project_id ON domains(project_id);

-- Environment variables
CREATE TABLE env_vars (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    encrypted_value BLOB NOT NULL,
    is_secret INTEGER NOT NULL DEFAULT 0,
    scope TEXT NOT NULL DEFAULT 'all',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(project_id, key, scope)
);
CREATE INDEX idx_env_vars_project_id ON env_vars(project_id);

-- Notification configurations
CREATE TABLE notification_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    webhook_url TEXT NOT NULL,
    events TEXT NOT NULL DEFAULT 'all',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Activity log
CREATE TABLE activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);
CREATE INDEX idx_activity_log_resource ON activity_log(resource_type, resource_id);

-- Platform settings (key-value store)
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Default settings
INSERT INTO settings (key, value) VALUES
    ('setup_complete', 'false'),
    ('registration_enabled', 'false'),
    ('max_projects', '50'),
    ('max_deployments_per_project', '20'),
    ('artifact_retention_days', '30'),
    ('max_concurrent_builds', '1');
