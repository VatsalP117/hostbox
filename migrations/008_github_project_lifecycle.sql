-- Persist the GitHub repository identity and connection state used by webhook
-- fan-out. Multiple Hostbox projects may intentionally target one repository.

ALTER TABLE projects ADD COLUMN github_repository_id INTEGER;
ALTER TABLE projects ADD COLUMN github_connection_status TEXT NOT NULL DEFAULT 'disconnected'
    CHECK (github_connection_status IN ('active', 'suspended', 'access_removed', 'disconnected'));

UPDATE projects
SET github_connection_status = CASE
    WHEN github_installation_id IS NOT NULL AND github_repo IS NOT NULL THEN 'active'
    ELSE 'disconnected'
END;

CREATE INDEX idx_projects_github_source
    ON projects(github_installation_id, github_repo);

CREATE INDEX idx_projects_github_repository_id
    ON projects(github_repository_id);
