# Implementation Report

## Summary

Implemented canonical build-path containment, deterministic exact-commit checkout, ephemeral GitHub authentication, resolved-SHA and framework persistence, and atomic terminal build-state transitions. Cancellation now remains cancelled, publishes a cancelled completion event, and never invokes the failure hook.

## Files Changed

- `internal/platform/sanitize/sanitize.go` and tests: canonical containment, symlink rejection, safe relative path normalization.
- `internal/worker/executor.go` and tests: safe filesystem boundaries, exact Git checkout, credential handling, metadata persistence, cancellation behavior.
- `internal/repository/project.go` and tests: framework build metadata and resolved deployment SHA persistence.
- `internal/repository/deployment.go` and tests: atomic compare-and-set deployment updates.
- `.agent/tasks/2026-07-12-v1-m1-build-integrity/*`: task, plan, and report.

## Commands Run

- `go test ./internal/platform/sanitize ./internal/repository ./internal/worker`
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/api ./cmd/cli`
- `go test -race ./internal/platform/sanitize ./internal/repository ./internal/worker`

## Tests

All focused and full checks passed. Race tests passed with benign macOS linker warnings about `LC_DYSYMTAB` in sqlite-linked test binaries.

## Deviations From Plan

The deployment repository received one narrowly scoped compare-and-set update method after orchestration approval. This was required to guarantee terminal cancellation under database races; read-then-write logic was insufficient.

## Known Risks

- Installation credentials are process-scoped and are not placed in argv, URLs, logs, or repository config. They remain present in the Git child process environment during fetch; stronger isolation requires an architectural privilege boundary.
- Filesystem validation is canonical at check time. As with path-based APIs generally, a malicious same-user process able to mutate directory components concurrently could attempt a TOCTOU race.

## Next Steps

- Review the integrated Milestone 1 diff and rerun checks after the other parallel tracks finish.
- Consider a future dedicated credential helper/isolated Git process if the threat model includes hostile same-UID processes.
