# Task: Harden Caddy route concurrency

## Objective

Prevent full Caddy configuration syncs from overwriting concurrent incremental route changes, and remove routes created by a successful-build hook when the deployment is cancelled while that hook is running.

## Scope

- Serialize full snapshot/load operations with every incremental route mutation.
- Keep existing public constructors and runtime wiring intact.
- Add an optional post-success cancellation cleanup contract.
- Remove immutable deployment and branch-stable routes for the cancelled deployment.
- Add focused concurrency and cleanup tests, including race-detector coverage.

## Out of scope

- Changes to `cmd/api/main.go`.
- Production-route reconciliation redesign.
- Database schema changes.

## Acceptance criteria

- A full sync holds one shared coordinator from database snapshot through Caddy load.
- Incremental route mutations using the same builder cannot overlap the full sync.
- A cancelled deployment observed after `OnBuildSuccess` triggers route cleanup.
- Composite hooks forward cleanup only to hooks that implement it.
- Focused tests and race tests pass.
