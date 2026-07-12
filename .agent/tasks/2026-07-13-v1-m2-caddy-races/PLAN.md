# Plan

1. Add a lightweight mutation mutex to the shared Caddy `ConfigBuilder`.
2. Hold it across `SyncAll` snapshot queries, config construction, and load.
3. Hold it across each complete `RouteManager` operation, including delete-then-add replacements.
4. Add an optional worker cancellation-cleanup hook and composite forwarding.
5. Re-read deployment state after the success hook; if cancelled, refresh the in-memory state and invoke cleanup.
6. Implement Caddy cleanup for immutable and branch-stable routes.
7. Add deterministic serialization and cleanup tests; run focused tests with the race detector.

No schema, public API, authentication, or infrastructure changes are authorized by this plan.
