# Implementation Report

## Summary

Implemented SQLite-backed GitHub webhook delivery intake with unique delivery IDs, bounded payloads, atomic claims, startup recovery, bounded exponential retry, terminal failures, fixed workers, periodic bounded scans, and deadline-aware shutdown. The HTTP handler now verifies signatures and durably accepts a delivery before returning 202; duplicates remain idempotent and persistence failures are not acknowledged.

The event-router boundary now carries the processor context into push, pull-request, and installation handlers so graceful shutdown can cancel downstream work.

## Files Changed

- `migrations/006_github_webhook_deliveries.sql`
- `internal/models/github_webhook_delivery.go`
- `internal/repository/github_webhook_delivery.go`
- `internal/repository/github_webhook_delivery_test.go`
- `internal/repository/repository.go`
- `internal/database/schema_test.go`
- `internal/services/github/delivery_processor.go`
- `internal/services/github/delivery_processor_test.go`
- `internal/services/github/events.go`
- `internal/services/github/events_test.go`
- `internal/api/handlers/github_webhook.go`
- `internal/api/handlers/github_webhook_test.go`
- `.agent/tasks/2026-07-13-v1-m2-durable-webhooks/{TASK,PLAN,IMPLEMENTATION_REPORT}.md`

## Commands Run

- `gofmt` on all changed Go files
- `go test ./internal/database ./internal/repository`
- `go test ./internal/services/github -run 'TestDeliveryProcessor' -count=1`
- `go test ./internal/api/handlers -run 'TestGitHubWebhookHandler' -count=1`
- `go test ./internal/database ./internal/repository ./internal/services/github ./internal/api/handlers -count=1`
- `go test ./...`
- `go test -race ./internal/repository ./internal/services/github ./internal/api/handlers`
- `go vet ./...`
- `go build ./cmd/api ./cmd/cli`

## Tests

All focused and full Go checks passed. Focused race tests passed; the macOS linker emitted existing malformed `LC_DYSYMTAB` warnings for CGO-linked test binaries without causing failures. Vet and API/CLI builds passed with no output.

Coverage added for:

- Unique delivery insertion and duplicate payload preservation.
- Concurrent atomic claim behavior.
- Compare-and-set completion/retry transitions.
- Startup recovery to queued or terminal failed state.
- One execution for duplicate acceptance.
- Transient retry, terminal retry exhaustion, unavailable router, and interrupted delivery recovery.
- Shutdown deadline behavior and cancellation propagation to the router.
- Valid signature persistence, duplicate HTTP acceptance, invalid/missing signature rejection, required headers, oversized payload rejection, and non-acknowledgment of storage failure.

## Deviations From Plan

At the orchestrator's request, `GitHubEventRouter.Route` was changed to accept `context.Context`; this was necessary to make processor shutdown cancel downstream handlers. Runtime/main wiring was intentionally left to the orchestrator.

## Known Risks

- Processing is intentionally at-least-once across a process or machine crash: a crash after an external side effect but before the completed state commit can replay the delivery. The unique delivery record prevents concurrent execution and normal redeliveries; downstream deployment creation retains its existing commit-level deduplication.
- A router implementation that ignores its context can outlive the shutdown deadline, but concurrency remains bounded by the fixed worker count and the row is recovered on restart.
- Raw webhook payloads are retained for recovery. They are capped at 1 MiB and signatures/credentials are never stored or logged. The integrated milestone now prunes completed payloads after 30 days in bounded batches while retaining failed rows for diagnostics.

## Next Steps

- Keep the orchestrator wiring: construct the processor with `repos.GitHubWebhookDelivery` and a `RouterProvider`, pass it into the webhook handler, start it only after the event router is installed, and call deadline-bounded `Shutdown` during server shutdown.
- Include this slice in the milestone review package and review it alongside lifecycle feedback and preview cleanup ordering.
