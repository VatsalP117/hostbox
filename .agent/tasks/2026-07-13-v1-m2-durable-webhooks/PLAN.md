# Implementation Plan

## 1. Task Summary

Replace detached webhook processing with a durable SQLite-backed delivery queue and bounded processor.

## 2. Current System Understanding

The current handler verifies signatures, acknowledges immediately, and launches an untracked goroutine. Delivery IDs are not persisted, duplicate deliveries can be executed more than once, and restart recovery is unavailable.

## 3. Scope

### In Scope

- Add the webhook delivery schema, model, repository, registry wiring, processor, handler changes, and focused tests.
- Expose a small router-provider boundary for runtime integration by the orchestrator.

### Out of Scope

- Runtime/main wiring, event-router changes, GitHub feedback, deployment lifecycle, route cleanup, and public redelivery APIs.

## 4. Proposed Technical Approach

Store the raw bounded payload with a unique delivery ID. Claim due queued rows in a SQLite transaction, increment attempts on claim, and use compare-and-set updates for completion/retry. On startup, reset interrupted processing rows to queued. A single bounded dispatcher feeds a fixed worker set; acceptance only inserts and non-blockingly wakes the dispatcher.

## 5. Step-by-Step Execution Plan

1. Add migration/model and repository operations.
2. Test unique insertion, atomic claiming, retries, terminal failure, and recovery.
3. Add the fixed-worker processor and tests for success, duplicates, retry bounds, recovery, and shutdown.
4. Update the webhook handler to bound, verify, validate, persist, then acknowledge.
5. Run focused tests, full Go tests, race checks, vet, and builds.
6. Write the implementation report.

## 6. Test Plan

- Real migration schema coverage.
- Repository idempotency and compare-and-set transitions.
- Processor success, retry/terminal failure, restart recovery, duplicate delivery, and bounded dispatch.
- Handler invalid signature, oversized/malformed requests, durable failure, valid insert, and duplicate acceptance.

## 7. Acceptance Criteria

- HTTP 202 is never returned for a new valid delivery unless durable insertion succeeds.
- A delivery ID is executed at most once concurrently and duplicate acceptance creates no new row.
- Interrupted/retryable rows are recovered with bounded attempts and error metadata.
- Worker and queue sizes are bounded and shutdown can be deadline-limited.

## 8. Risks and Guardrails

- Never log payloads, signatures, or credentials.
- Preserve the webhook endpoint and successful response shape.
- Keep all changes within assigned ownership and add no dependencies.

## 9. Executor Instructions

Implement only this slice, coordinate the constructor/interface contract with the orchestrator, do not edit runtime/event/main integration files, and do not commit independently.
