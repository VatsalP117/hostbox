# Implementation Plan

## 1. Task Summary

Align every existing Hostbox CLI API call with the routes and JSON contracts already exposed by the backend, then verify the highest-risk deployment flows with HTTP-level unit tests.

## 2. Current System Understanding

The CLI contains stale nested deployment, domain, and environment-variable URLs; singular admin backup URLs; incomplete auth response decoding; and a deploy command that sends an empty branch to a backend endpoint that requires one. The logs command does not retrieve the registered logs resource.

## 3. Scope

### In Scope

- Correct CLI-only route, method, query, request-body, and response-wrapper mismatches.
- Resolve a missing deploy branch from the linked project's production branch.
- Retrieve deployment status and log snapshots through the existing global deployment routes.
- Align supported project creation fields with the backend request contract.
- Add focused `httptest` coverage for client contracts and decoding.

### Out of Scope

- Backend routes, handlers, DTOs, or public API changes.
- SSE log-follow implementation or deployment state-machine changes.
- New CLI feature areas unrelated to existing commands.
- Workflows, release scripts, web applications, or ignore files.

## 4. Proposed Technical Approach

Keep the generic HTTP client unchanged except where a helper is needed. Update each resource client to use route constants implied by `routes.go` and model the existing response envelopes. Keep branch fallback orchestration in the deploy command by reading the project before calling the trigger endpoint. Convert logs into a truthful snapshot command backed by `GET /deployments/:id/logs`.

## 5. Step-by-Step Execution Plan

1. Inventory all client methods against registered routes and handler DTOs.
2. Repair deployment, domain, environment-variable, admin, auth, and project contracts.
3. Update commands for changed client signatures and branch/log behavior.
4. Add table-driven HTTP tests covering paths, methods, bodies, query parameters, and response envelopes.
5. Run focused CLI tests, then Go formatting, vet, and CLI build checks.
6. Record the final diff and verification results in `IMPLEMENTATION_REPORT.md`.

## 6. Test Plan

- `go test ./cmd/cli/...`
- `go vet ./cmd/cli/...`
- `go build ./cmd/cli`
- `git diff --check -- cmd/cli .agent/tasks/2026-07-12-v1-m0-cli`

## 7. Acceptance Criteria

- Every existing CLI client method calls a route registered in `routes.go` with the correct HTTP method.
- Deploy uses `/deployments/trigger` and never sends an empty branch when `--branch` is omitted.
- Status and logs use global deployment routes and decode their response envelopes.
- Rollback, domain, env-var, and admin backup operations use their registered URLs.
- Focused client tests pass and the CLI vets/builds.

## 8. Risks and Guardrails

- Do not compensate for mismatches by changing backend/public API behavior.
- Do not add dependencies.
- Do not implement SSE following in this Milestone 0 repair.
- Do not touch files outside assigned ownership.

## 9. Executor Instructions

Implement the smallest coherent CLI-only patch. Preserve existing user-facing command names where possible, remove or reject flags the backend cannot honor, and report any remaining unsupported behavior explicitly.
