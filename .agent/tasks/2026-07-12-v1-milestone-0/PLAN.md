# Implementation Plan

## 1. Task Summary

Complete Milestone 0's first vertical slice using parallel agents with non-overlapping file ownership, then integrate, review, and commit the combined result.

## 2. Current System Understanding

The Go backend is green, while dashboard lint, marketing clean install, standalone server release packaging, CLI archive installation, and CLI deployment endpoints have verified failures. Machine-generated artifacts are tracked.

## 3. Scope

### In Scope

- Fix dashboard lint without changing log-viewer behavior.
- Make marketing dependencies reproducibly install/build and add that build to CI.
- Make published release artifacts executable and the CLI installer compatible with archive contents/checksums.
- Align CLI deployment/status/log calls with existing API routes and add client tests.
- Remove the tracked local API binary and stale generated frontend compiler/Vite outputs; update ignores if needed.

### Out of Scope

- Deployment state-machine/backend route redesign.
- Database/auth/security-boundary work from later milestones.
- Publishing or pushing releases.
- DNS, Caddy, or VM changes.

## 4. Proposed Technical Approach

Run three independent executor tracks: frontend/marketing baseline, release packaging, and CLI API contract. Integrate on `agent/v1-milestone-0`, run the complete local verification matrix, create a review package against `main`, and fix only review blockers.

## 5. Step-by-Step Execution Plan

1. Agents inspect and patch only their assigned files, add focused tests, and report outcomes.
2. Remove stale tracked binaries/generated build metadata and update ignore rules.
3. Review each track's diff and interaction with the release workflow.
4. Run Go tests/vet/build, dashboard install/lint/type/build/audit, marketing install/build, Compose validation, CLI installer/release-shape tests, and workflow syntax checks available locally.
5. Write implementation/review artifacts and commit a small coherent Milestone 0 change.

## 6. Test Plan

- `go test ./...`, `go vet ./...`, `go build ./cmd/api ./cmd/cli`
- Dashboard: `npm ci`, `npm run lint`, `npx tsc --noEmit`, `npm run build`
- Marketing: `npm ci`, `npm run build`
- Focused CLI client/command tests
- `docker compose config --quiet`
- Validate release archive naming and installer checksum behavior without publishing
- `git diff --check`

## 7. Acceptance Criteria

- Existing CI commands are green locally, including marketing.
- No release job produces a known-unusable server binary.
- CLI archives install under the `hostbox` command with checksum verification.
- CLI deployment/status/log flows call registered API routes.
- Generated/local binaries are no longer tracked.
- No unrelated or user-owned file is included.

## 8. Risks and Guardrails

- Agents share a worktree; file ownership must not overlap.
- Preserve `.opencodeignore`.
- Do not weaken lint, audit, tests, or checksum validation to make gates pass.
- Do not publish, push, or tag.

## 9. Executor Instructions

Make narrowly scoped edits, add focused tests, run checks, and write a track-specific implementation report. Do not commit independently; the orchestrator will review and commit the integrated result.
