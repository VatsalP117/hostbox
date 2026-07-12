# Implementation Plan

## 1. Task Summary

Make frontend builds deterministic and path-safe while ensuring cancellation remains terminal and build metadata is accurate.

## 2. Current System Understanding

The worker cloned the latest branch head with a credential-bearing URL, joined root/output paths lexically, did not persist detected framework, and finalized status with unconditional database updates. A cancellation could therefore be overwritten by failed or ready.

## 3. Scope

### In Scope

- Canonical containment for clone, source, output, and log paths.
- Traversal and symlink-escape rejection.
- Exact full-SHA fetch, detached checkout, and verification.
- Manual/empty SHA resolution to the selected branch head and persistence of the resolved full SHA.
- Ephemeral Git authentication without credentials in the remote URL or repository config.
- Detected framework persistence.
- Compare-and-set build-state updates and cancellation-specific hook/SSE behavior.
- Focused and full Go verification.

### Out of Scope

- Schema/public API changes.
- Deployment activation, Caddy implementation, notification handling, workflows, or docs.
- New dependencies.

## 4. Proposed Technical Approach

Resolve paths against canonical existing ancestors, retaining safe non-existent suffixes. Initialize a Git worktree with a public remote and pass the installation credential as a process-scoped Git config header. Fetch either the exact 40-character SHA or the named branch, detach at `FETCH_HEAD`, resolve `HEAD`, verify it, and persist it. Store detected framework alongside existing build metadata. Use atomic status compare-and-set updates for queued-to-building and building-to-terminal transitions.

## 5. Step-by-Step Execution Plan

1. Add reusable canonical path and relative configuration validation.
2. Apply validation to every worker-controlled filesystem boundary.
3. Replace branch-head clone with explicit fetch/checkout/verification.
4. Persist resolved SHA and detected framework.
5. Add atomic deployment update-by-expected-status repository support.
6. Make cancellation emit cancelled state only and skip failure hooks.
7. Add focused tests and run full checks.

## 6. Test Plan

- Traversal, symlink escape, and safe non-existent destination tests.
- Local two-commit Git remote tests for exact older SHA and manual branch-head resolution.
- Invalid abbreviated SHA and credential-exposure tests.
- Framework and resolved-SHA repository persistence tests.
- Compare-and-set cancellation preservation test.
- Worker cancellation test asserting cancelled DB/SSE state and zero failure hooks.
- Focused tests, full `go test`, `go vet`, builds, and focused race tests.

## 7. Acceptance Criteria

- Root/output/deployment paths cannot escape their configured roots lexically or through existing symlinks.
- A supplied full SHA is the exact commit built and recorded.
- Manual/empty deployments record the full resolved branch-head SHA.
- Git credentials are absent from command arguments, remote URL, and persisted config.
- Detected framework is available to routing after the build.
- Cancelled cannot transition to failed/ready and does not emit a failure hook.
- All Go checks pass.

## 8. Risks and Guardrails

- Filesystem containment checks reduce but cannot fully eliminate operating-system-level TOCTOU without descriptor-relative APIs.
- Git credential material exists only in the child process environment for the fetch duration; eliminating same-user process-environment visibility requires a larger privilege/transport design.
- Reject non-sentinel abbreviated SHAs because they cannot be deterministically verified as the recorded full commit.

## 9. Executor Instructions

Keep changes narrow, add tests for each security property, do not add dependencies, do not commit, and document any architectural limitation rather than broadening scope.
