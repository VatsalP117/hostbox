# Implementation Report

## Summary

Completed the first Milestone 1 core-security/correctness slice through three parallel tracks and an orchestrator review/fix pass.

- Builds now use canonical safe paths, reject unsafe static entries, build and persist the exact SHA, avoid credential-bearing Git URLs/argv/config, and persist detected framework metadata.
- Deployment status changes use database compare-and-set operations; worker and service cancellation remain terminal and skip failure hooks.
- Rollback/promote validate artifacts and activate Caddy synchronously, with failed-state compensation on activation errors.
- Notification webhook URLs are validated before persistence and at all send boundaries.

## Files Changed

- Build/safety: `internal/platform/sanitize`, `internal/worker`, project/deployment repositories and tests.
- Activation: deployment service/types/tests, Caddy adapter/tests, API constructor wiring.
- Notifications: notification handler/service and focused tests.
- Roadmap and engineering-pipeline artifacts.

## Commands Run

- Focused and full Go tests.
- Focused race tests for repositories, worker, deployment/Caddy, handlers, and notifications.
- Go vet and API/CLI builds.
- Existing CLI installer/release contract smoke tests.
- `gofmt` and `git diff --check`.

## Tests

Passed:

- `go test ./cmd/... ./internal/... ./migrations`
- `go vet ./cmd/... ./internal/... ./migrations`
- `go build ./cmd/api ./cmd/cli`
- `go test -race` for all changed security/correctness packages
- Existing release packaging contract tests

Regression coverage includes traversal/symlink escape, static symlink exfiltration, non-regular static entries, exact older-SHA checkout, manual SHA resolution, credential exposure, framework persistence, terminal cancellation, activation inputs/failure compensation, artifact validity, and notification URL persistence/send boundaries.

## Deviations From Plan

- Review added static-copy symlink/non-regular rejection after identifying that canonical project-root checks alone did not protect the direct static copy path.
- Resolved commit persistence was moved to the deployment repository during review.
- Service-level cancellation was also converted to compare-and-set so it cannot overwrite a build that became ready between read and cancel.

## Known Risks

- Short-lived Git credentials remain visible in the Git child process environment to same-UID privileged inspection.
- Filesystem validation remains subject to same-user TOCTOU mutation.
- Caddy and SQLite changes cannot be atomic; compensation has a documented double-failure window.
- Existing Caddy production replacement deletes then adds the route.
- URL validation does not pin DNS at dial time and therefore does not close DNS rebinding/redirect attacks.

## Next Steps

Continue Milestone 1 with durable GitHub webhook intake/status/comments, branch/PR route cleanup, stronger outbound HTTP transport validation, and real Docker/Caddy integration fixtures.
