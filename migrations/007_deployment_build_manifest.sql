-- Persist the immutable build recipe used for each deployment.
ALTER TABLE deployments ADD COLUMN build_framework TEXT;
ALTER TABLE deployments ADD COLUMN build_serving_mode TEXT;
ALTER TABLE deployments ADD COLUMN build_package_manager TEXT;
ALTER TABLE deployments ADD COLUMN build_package_manager_version TEXT;
ALTER TABLE deployments ADD COLUMN build_node_version TEXT NOT NULL DEFAULT '';
ALTER TABLE deployments ADD COLUMN build_root_directory TEXT NOT NULL DEFAULT '.';
ALTER TABLE deployments ADD COLUMN build_output_directory TEXT;
ALTER TABLE deployments ADD COLUMN build_install_command TEXT;
ALTER TABLE deployments ADD COLUMN build_command TEXT;
ALTER TABLE deployments ADD COLUMN build_lock_file_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE deployments ADD COLUMN build_manifest_resolved INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployments ADD COLUMN source_repository TEXT;
ALTER TABLE deployments ADD COLUMN source_installation_id INTEGER;

-- Preserve the current project settings for deployments that were queued
-- before this migration. Their effective recipe remains unresolved and will
-- be finalized by the worker after checkout.
UPDATE deployments
SET build_framework = (SELECT framework FROM projects WHERE projects.id = deployments.project_id),
    build_node_version = COALESCE((SELECT node_version FROM projects WHERE projects.id = deployments.project_id), '20'),
    build_root_directory = COALESCE((SELECT root_directory FROM projects WHERE projects.id = deployments.project_id), '.'),
    build_output_directory = (SELECT output_directory FROM projects WHERE projects.id = deployments.project_id),
    build_install_command = (SELECT install_command FROM projects WHERE projects.id = deployments.project_id),
    build_command = (SELECT build_command FROM projects WHERE projects.id = deployments.project_id),
    source_repository = (SELECT github_repo FROM projects WHERE projects.id = deployments.project_id),
    source_installation_id = (SELECT github_installation_id FROM projects WHERE projects.id = deployments.project_id);
