-- Add GitHub deployment tracking to deployments table.
-- Stores the GitHub Deployment API ID for posting status updates.

ALTER TABLE deployments ADD COLUMN github_deploy_id INTEGER;
