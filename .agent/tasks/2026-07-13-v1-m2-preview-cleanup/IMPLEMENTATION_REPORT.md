# Implementation Report

## Summary

Implemented complete handler-level preview cleanup for closed pull requests and deleted branches. Caddy can now remove a stable branch route from a project ID plus raw branch name. Both GitHub event paths deactivate matching preview deployments, attempt removal of every immutable route returned by deactivation, always attempt stable-route removal, and return joined contextual errors for durable retry processing.

## Files Changed

- `internal/services/caddy/manager.go`
- `internal/services/caddy/manager_test.go`
- `internal/services/github/interfaces.go`
- `internal/services/github/push_handler.go`
- `internal/services/github/pr_handler.go`
- `internal/services/github/events_test.go`
- `.agent/tasks/2026-07-13-v1-m2-preview-cleanup/TASK.md`
- `.agent/tasks/2026-07-13-v1-m2-preview-cleanup/PLAN.md`
- `.agent/tasks/2026-07-13-v1-m2-preview-cleanup/IMPLEMENTATION_REPORT.md`

## Commands Run

- `gofmt -w` on all changed Go files
- `go test ./internal/services/caddy ./internal/services/github`
- `go test -race ./internal/services/caddy ./internal/services/github`
- `go vet ./internal/services/caddy ./internal/services/github`
- `go test ./...`

## Tests

- Focused Caddy and GitHub tests pass.
- Focused Caddy and GitHub race tests pass.
- Focused Caddy and GitHub vet checks pass.
- Full Go tests pass for all packages except `cmd/api`, which awaits the orchestrator-owned `NewPushHandler` constructor wiring update to pass `routeManager`.
- Added coverage for raw slash-containing branch normalization, PR-close cleanup, deleted-branch cleanup, all-route attempts after failures, joined errors, and deleted non-branch refs.

## Deviations From Plan

None. API entrypoint wiring was intentionally left to the orchestrator as assigned.

## Known Risks

- The current repository implementation returns only deployments that are `ready` at the moment it marks them cancelled. If a route deletion fails, a durable retry may no longer receive that immutable deployment ID. The orchestrator was notified that retry-safe lookup/deactivation semantics are needed outside this slice.
- Current deactivation ignores queued/building previews, which can later complete and recreate routes after branch/PR closure. The orchestrator was notified that cancellation or worker-side terminal-state enforcement is required outside this slice.
- Caddy deletion currently reports an already-missing route as an error. Durable retries are safer if missing-route deletion is treated idempotently; this was raised to the orchestrator because the client file is outside assigned ownership.

## Next Steps

- Update `cmd/api/main.go` to pass `routeManager` into `NewPushHandler`.
- Resolve the retry/cancellation risks above in repository/worker integration.
- Run the umbrella full test, race, vet, and build gates after all parallel slices are integrated.
