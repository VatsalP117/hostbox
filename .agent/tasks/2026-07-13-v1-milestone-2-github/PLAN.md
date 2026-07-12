# Implementation Plan

## 1. Task Summary

Replace detached webhook goroutines with durable bounded processing, connect the existing GitHub deployment/comment clients to real deployment transitions, and remove every preview route when a branch or PR closes.

## 2. Current System Understanding

The webhook handler verifies signatures but acknowledges before a detached goroutine processes the event. Delivery IDs are not persisted. Status/comment helpers exist but are unused. PR close removes immutable deployment routes only, branch deletion is ignored, and branch-stable routes lack a removal API.

## 3. Scope

### In Scope

- Add a webhook-delivery table/repository with unique delivery IDs, payload/event/status, attempt/error metadata, timestamps, and recovery queries.
- Add a bounded processor that durably accepts before 202, recovers queued/processing work after restart, applies bounded retries, and marks terminal outcomes.
- Treat duplicate delivery IDs idempotently.
- Report queued/building/ready/failed/cancelled deployment states to one GitHub Deployment ID.
- Create/update one marked PR comment for preview lifecycle states.
- Remove immutable and branch-stable routes and deactivate previews on PR close and branch deletion.
- Add migration, repository, processor, feedback, route lifecycle, and integration tests.

### Out of Scope

- Dashboard retry UI, public redelivery endpoint, fork PR support, repository rename/transfer, installation suspension UX, multi-project-per-repository redesign.
- New dependencies, auth changes, or external queue infrastructure.

## 4. Proposed Technical Approach

Use SQLite as the durable queue to preserve the lightweight single-process architecture. A bounded in-process worker claims rows atomically and periodically scans recoverable work. GitHub feedback is exposed behind a small internal lifecycle interface consumed by the deployment service/worker. Route cleanup extends the existing Caddy manager abstraction. Three agents own persistence/processor, feedback, and route cleanup; the orchestrator owns shared runtime/main/deployment/worker integration.

## 5. Step-by-Step Execution Plan

1. Implement migration, model/repository, durable processor, and webhook handler acceptance/idempotency.
2. Implement feedback reporter using existing status/comment clients and repository metadata.
3. Implement branch route removal and push/PR cleanup behavior.
4. Integrate runtime client/router access, lifecycle calls, startup recovery, and graceful shutdown.
5. Review failure ordering, retry semantics, duplicate handling, secrets/logs, and terminal states.
6. Run migration/repository, handler/processor, GitHub/Caddy, worker/deployment, full Go, race, vet, build, and existing packaging checks.

## 6. Test Plan

- Migration/schema and repository state-transition/idempotency tests.
- Handler tests proving signature failure is not stored and 202 follows durable acceptance.
- Processor tests for success, duplicate delivery, transient retry, terminal failure, queue saturation/recovery, and restart recovery.
- Feedback tests for one GitHub deployment reused across states and one PR comment updated.
- Push deletion/PR close route cleanup tests including branch-stable routes.
- Full `go test`, focused `-race`, vet, and API/CLI builds.

## 7. Acceptance Criteria

- No valid webhook is acknowledged unless its payload is durably stored.
- Duplicate GitHub delivery IDs do not create duplicate deployments.
- Restarted Hostbox retries recoverable deliveries with bounded attempts and observable last errors.
- Deployment feedback reuses the persisted GitHub deployment ID and updates the marked PR comment.
- Closed/deleted branches have no immutable or branch-stable active routes and cannot be resurrected by reconciliation.
- No unrelated/user-owned file is committed.

## 8. Risks and Guardrails

- Payloads can contain repository metadata; never log/store signatures or credentials, and bound request payload size.
- Claims/status changes must be atomic to avoid concurrent duplicate processing.
- Do not block webhook HTTP requests on GitHub event execution after durable persistence.
- Keep retry/backoff bounded and shutdown-aware; no unbounded goroutines/timers.
- Preserve existing public webhook response shape where practical.

## 9. Executor Instructions

Stay within assigned ownership, add focused tests and task reports, and do not commit independently. Escalate through a change request rather than broadening schema or API scope.
