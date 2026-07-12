# Implementation report

## Changes

- Added a mutation mutex to `ConfigBuilder`, which is already shared by `SyncService` and `RouteManager`.
- `SyncAll` now serializes the database snapshot and Caddy `/load` as one operation.
- Every incremental route-manager mutation now uses the same coordinator; multi-request replacements remain atomic relative to other Hostbox Caddy writes.
- Added optional `PostBuildCancellationHook` support and composite-hook forwarding.
- The executor now checks the durable deployment state after success hooks. If another workflow cancelled the deployment during route activation, it refreshes the local deployment and invokes cleanup before lifecycle reporting.
- The Caddy post-build hook removes both immutable deployment and branch-stable routes on this cancellation path.

## Verification

- `go test ./internal/services/caddy ./internal/worker`
- `go test -race ./internal/services/caddy ./internal/worker`

Both passed. The macOS linker emitted a non-fatal malformed `LC_DYSYMTAB` warning during the race build; the test binaries completed successfully.
