# Implementation Plan

## 1. Task Summary

Make rollback and promote activate the selected artifact on the production route synchronously, with artifact validation and explicit activation-failure handling.

## 2. Current System Understanding

Rollback and promote currently create ready production deployment records that reuse an earlier artifact, but leave Caddy unchanged. Caddy already exposes `RouteManager.UpdateProductionRoute`, and the deployment service has access to the project metadata needed to call it. The database and Caddy cannot participate in one transaction.

## 3. Scope

### In Scope

- A small internal deployment activation interface and Caddy adapter.
- Rollback/promote artifact-directory existence and non-empty validation.
- Synchronous activation after deployment creation and before a successful return.
- Best-effort compensation that marks the new record failed when activation fails.
- Constructor wiring in `cmd/api/main.go`.
- Deployment service and Caddy adapter tests.

### Out of Scope

- Public API/DTO changes.
- Database schema or repository changes.
- Caddy infrastructure redesign or transactional route storage.
- Worker, handler, notification, workflow, and documentation changes.

## 4. Proposed Technical Approach

Define `ProductionActivator` in the deployment service package with a compact activation value. Implement it in Caddy as an adapter over the existing route manager. Before creating a rollback/promote record, require a non-empty artifact path that identifies a readable, non-empty directory. Create the record, invoke activation synchronously, and return only after activation succeeds. If activation fails, mark the just-created record failed and return the activation error; log any compensation failure without hiding the primary failure.

## 5. Step-by-Step Execution Plan

1. Add the deployment activation interface/value and inject it through `NewService`.
2. Add shared artifact validation and activation helpers to rollback/promote.
3. Add a Caddy adapter and wire it in the API process.
4. Extend service tests for successful calls, invalid artifacts, and activation failure state.
5. Add a Caddy adapter test proving delegation builds the production route.
6. Run focused tests, formatting, full Go tests, vet, and builds.

## 6. Test Plan

- Rollback and promote call activation with project slug/ID, artifact, and framework.
- Missing, blank, non-existent, empty, and non-directory artifact paths are rejected before record creation and activation.
- Activation errors are returned and the created record is marked failed.
- The Caddy adapter delegates to the existing production-route behavior.
- Full Go tests, vet, and API build pass.

## 7. Acceptance Criteria

- Successful rollback/promote immediately updates the production route.
- Invalid artifacts create no new record and perform no activation.
- Activation failure cannot produce an API success and does not leave the new record ready.
- Public contracts and schema are unchanged.

## 8. Risks and Guardrails

- Database persistence and Caddy activation are not atomic. Compensation prevents the new record from remaining active in database-driven sync after an activation failure, but a compensation write failure remains a logged operational risk.
- Use the existing Caddy route-update behavior; do not broaden into routing infrastructure redesign.
- Do not touch files outside the assigned paths.

## 9. Executor Instructions

Keep changes focused, add deterministic tests, run focused checks before full checks, and document all results in `IMPLEMENTATION_REPORT.md`. Do not commit.
