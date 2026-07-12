# Implementation Plan

## 1. Task Summary

Close several verified P0 correctness/security gaps in parallel while preserving the static single-node architecture.

## 2. Current System Understanding

Builds clone a branch head without verifying the recorded SHA and accept unchecked root/output paths. Cancelled executors can overwrite `cancelled` with `failed`. Rollback/promote create ready DB rows but do not activate production routing. Notification URL validation exists but is not called by handlers.

## 3. Scope

### In Scope

- Constrain build root/output paths to intended roots, including traversal/symlink tests.
- Fetch/checkout/verify the exact deployment commit and avoid persisting credentials in Git remotes/arguments where practical.
- Persist detected framework so Caddy receives correct SPA/static behavior.
- Prevent cancelled builds from ending failed or emitting failure hooks.
- Activate rollback/promote through a tested internal routing abstraction.
- Enforce existing outbound webhook validation on notification create/update/test paths with tests.

### Out of Scope

- Durable webhook queue, Docker network policy, schema redesign, auth/session changes, domain verification, public route changes.

## 4. Proposed Technical Approach

Use three non-overlapping executor tracks: build integrity/worker, deployment activation/service wiring, and notification URL enforcement. Prefer small internal interfaces and pure helpers with focused tests. Integrate and run all Go checks plus existing frontend/packaging smoke checks.

## 5. Step-by-Step Execution Plan

1. Implement safe canonical build paths, exact SHA checkout/verification, framework persistence, and cancellation terminal-state handling.
2. Add an activation interface to deployment rollback/promote and wire the existing Caddy route path.
3. Apply notification URL validation consistently before persistence/send.
4. Review security/error handling and cross-track constructor interactions.
5. Run focused and full Go tests/race/vet/build; run existing web/marketing/package contract smoke checks.

## 6. Test Plan

- Focused worker/sanitize/repository tests including traversal, symlink, exact commit, cancellation.
- Deployment service tests asserting activation and activation failure behavior.
- Handler tests for private/loopback/insecure webhook rejection and valid HTTPS acceptance.
- `go test ./cmd/... ./internal/... ./migrations`, vet, builds, and race where feasible.
- Existing Node builds and packaging contracts as integration smoke checks.

## 7. Acceptance Criteria

- A deployment cannot escape its checkout/output roots and builds the SHA it records.
- Cancel is terminal and does not become failed.
- Successful rollback/promote immediately invoke production activation; activation failure is surfaced without a false success response.
- Unsafe notification targets are rejected on create/update/test.
- No unrelated files or `.opencodeignore` are committed.

## 8. Risks and Guardrails

- Avoid shell-string interpolation of untrusted refs/SHAs.
- Preserve existing public API shapes.
- Do not introduce schema changes or new dependencies.
- Keep agents within assigned files because the worktree is shared.

## 9. Executor Instructions

Add focused tests, report exact security decisions, and do not commit independently. Escalate through `CHANGE_REQUEST.md` rather than broadening scope.
