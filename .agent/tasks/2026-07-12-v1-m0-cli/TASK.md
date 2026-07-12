# Task

Repair the Hostbox CLI's HTTP contract for v1 Milestone 0 without changing the backend or public API. Audit all existing CLI client methods and commands against `internal/api/routes/routes.go`, prioritize deploy/status/logs/rollback, resolve the production branch for deploys that omit `--branch`, and add focused client tests for routes, methods, request bodies, and response decoding.

The implementation is limited to `cmd/cli/**` and this task artifact folder. It must not modify workflows, scripts, frontend applications, repository ignores, or backend code.
