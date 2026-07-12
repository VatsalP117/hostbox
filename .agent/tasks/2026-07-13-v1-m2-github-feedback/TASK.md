# Task

Implement synchronous GitHub deployment-status and pull-request comment feedback for Hostbox deployment lifecycle transitions.

The reporter must support queued, building, ready, failed, and cancelled states; safely validate GitHub metadata; create and persist one GitHub Deployment ID per Hostbox deployment; reuse it for later statuses; and maintain one marker-based pull-request comment for preview deployments. It must expose a dynamic client-provider boundary for later runtime wiring and return all errors to its caller.

Scope is limited to GitHub status/comment/feedback code and the narrow deployment repository method needed to persist `github_deploy_id`. Runtime, worker, deployment-service, handler, route, migration, and application wiring are owned by the orchestrator or other tracks.
