# Review

## Scope

Reviewed the complete Milestone 2 diff for correctness, security, migration safety, concurrency, retry behavior, route lifecycle, and user-owned file isolation.

## Findings resolved

- Prevented post-close route resurrection by rechecking durable deployment state after success hooks and invoking route cleanup.
- Serialized Caddy full snapshots/loads with incremental route mutations.
- Added bounded lifecycle feedback retries so transient terminal-state failures are not immediately lost.
- Paginated PR comments so a marker beyond the first 100 comments is reused.
- Recovered router panics into normal delivery retry/failure transitions without losing a fixed worker.
- Added bounded pruning of completed webhook payloads.
- Made DELETE 404 successful and returned cancelled previews on cleanup retries for idempotent partial-failure recovery.
- Cancelled queued/building work on branch/PR cleanup so it cannot later finalize ready.
- Associated push-created previews with later PR events and made commit deduplication branch-scoped, preserving the one-deployment/one-comment invariant regardless of webhook ordering.

## Verification result

Passed:

- `git diff --check`
- `go test ./...`
- `go test -race ./internal/repository ./internal/api/handlers ./internal/services/github ./internal/services/caddy ./internal/services/deployment ./internal/worker`
- `go vet ./...`
- `go build ./cmd/api ./cmd/cli`
- `bash scripts/tests/test-release-packaging.sh`
- `bash scripts/tests/test-install-cli.sh`

The race-linked macOS test binaries emitted the repository's known non-fatal `LC_DYSYMTAB` linker warning; all commands exited successfully.

## Repository hygiene

The unrelated user-owned `.opencodeignore` remains untracked and is excluded from the commit.
