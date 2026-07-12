# Implementation Report

## Summary

Completed the CLI API-contract repair for v1 Milestone 0. All HTTP paths used by existing CLI resource clients now correspond to registered backend routes, and the affected response envelopes/request field names match the existing handler DTOs.

Key outcomes:

- `deploy` now calls `POST /api/v1/projects/:projectId/deployments/trigger` and resolves an omitted `--branch` from the project's `production_branch` before triggering.
- Specific deployment status uses the global `GET /api/v1/deployments/:id` route.
- `logs` now retrieves and prints the existing log snapshot resource at `GET /api/v1/deployments/:id/logs`; its help text no longer claims unsupported streaming.
- Rollback remains on the registered project rollback route with its required `deployment_id` body.
- Domain deletion/verification and environment-variable deletion now use their registered global-ID routes.
- Environment-variable list/create calls now use `env-vars` rather than the stale `env` path.
- Admin backup creation/list/restore now use the plural `/admin/backups` routes, with `compress` sent as the handler's query parameter.
- `whoami` now unwraps the backend's `{ "user": ... }` response.
- Project creation sends `github_repo` and `output_directory`, exposes the already-supported install command, and honors a non-default production branch through the registered project PATCH route. Stale `--slug` and `--framework` flags were removed because the current API cannot accept those fields.

## Files Changed

- `cmd/cli/internal/client/admin.go`
- `cmd/cli/internal/client/auth.go`
- `cmd/cli/internal/client/deployments.go`
- `cmd/cli/internal/client/domains.go`
- `cmd/cli/internal/client/envvars.go`
- `cmd/cli/internal/client/projects.go`
- `cmd/cli/internal/client/contracts_test.go`
- `cmd/cli/cmd/deploy.go`
- `cmd/cli/cmd/deploy_test.go`
- `cmd/cli/cmd/domains.go`
- `cmd/cli/cmd/env.go`
- `cmd/cli/cmd/projects.go`
- `cmd/cli/cmd/status.go`
- `.agent/tasks/2026-07-12-v1-m0-cli/TASK.md`
- `.agent/tasks/2026-07-12-v1-m0-cli/PLAN.md`
- `.agent/tasks/2026-07-12-v1-m0-cli/IMPLEMENTATION_REPORT.md`

## Commands Run

- `gofmt` on changed Go files
- `go test ./cmd/cli/...`
- `go vet ./cmd/cli/...`
- `go build ./cmd/cli`
- `go test ./...`
- `go vet ./...`
- `go build -o /tmp/hostbox-cli-v1-m0 ./cmd/cli`
- `git diff --check -- cmd/cli .agent/tasks/2026-07-12-v1-m0-cli`

## Tests

All checks passed.

The new HTTP contract tests verify:

- Deployment list/get/trigger/logs/rollback methods and paths.
- Trigger and rollback request bodies.
- Deployment and log response decoding.
- Login and wrapped current-user decoding.
- Project list/get/create/update routes, field names, and response decoding.
- Domain list/create/delete/verify routes.
- Environment-variable list/create/delete routes.
- Admin backup create/list/restore routes and compression query.
- Explicit deploy branches and production-branch fallback behavior.

## Deviations From Plan

None. The implementation stayed within `cmd/cli/**` and this task folder and did not change the backend or public API.

## Known Risks

- `hostbox logs` intentionally retrieves the existing bounded log snapshot (the backend defaults to at most 1,000 lines). It does not follow the SSE stream; real-time SSE support remains later CLI work.
- Setting a non-default branch during `project create` requires a create followed by a PATCH because the current create API always initializes projects with `main`. A PATCH failure can therefore leave the newly created project on `main`, although the CLI now reports that error instead of silently claiming the requested branch was used.
- The CLI still authenticates only with the access token and does not persist the backend's refresh cookie; that pre-existing session-lifecycle limitation is outside this route-contract milestone.

## Next Steps

- Integrate this track with the other Milestone 0 changes and run the combined verification matrix.
- Consider SSE log following, access-token renewal, and an atomic create-project production-branch contract in later v1 milestones.
