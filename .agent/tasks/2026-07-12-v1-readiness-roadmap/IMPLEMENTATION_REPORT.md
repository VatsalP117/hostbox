# Implementation Report

## Summary

Audited Hostbox's intended v1 specification against the current backend, worker, Caddy, GitHub integration, dashboard, CLI, persistence, packaging, scripts, CI, and operational documentation. Added `docs/V1-READINESS-ROADMAP.md`, which defines the focused static/single-node v1 contract, records repository-proven gaps, organizes the work into prioritized workstreams, and supplies sequencing and release gates.

## Files Changed

- `docs/V1-READINESS-ROADMAP.md`
- `.agent/tasks/2026-07-12-v1-readiness-roadmap/TASK.md`
- `.agent/tasks/2026-07-12-v1-readiness-roadmap/PLAN.md`
- `.agent/tasks/2026-07-12-v1-readiness-roadmap/IMPLEMENTATION_REPORT.md`
- `.agent/tasks/2026-07-12-v1-readiness-roadmap/REVIEW.md`

## Commands Run

- Repository/file/history/status inventory and targeted source searches.
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/api ./cmd/cli`
- `npm ci`, `npm run lint`, `npx tsc --noEmit`, `npm run build` in `web/`.
- `npm ci` in `marketing/`.
- `npm audit --omit=dev --json` in `web/`.
- `docker compose config --quiet`.
- A runtime check of the release workflow's `CGO_ENABLED=0` API build.
- `git diff --check`.

## Tests

### Passed

- Backend package tests.
- Go vet.
- Normal Go API/CLI builds.
- Dashboard TypeScript check.
- Dashboard production build.
- Production Compose configuration parse.
- Documentation whitespace check.

### Current baseline failures documented in the roadmap

- Dashboard lint: two `no-control-regex` errors and five warnings.
- Marketing clean install: Astro 6 / `@astrojs/tailwind` peer dependency conflict.
- Runtime dependency audit: two high and two moderate production findings.
- Release-style API binary: starts but fails when SQLite is opened because it was built with CGO disabled.

## Deviations From Plan

No production code was changed. Full Docker image builds and public DNS/GitHub/ACME VM flows were not rerun locally; the absence of those acceptance tests is itself captured as a v1 release blocker.

## Known Risks

The roadmap is intentionally comprehensive. It should be split into narrow implementation issues before execution, with security and deterministic deployment work preceding UI polish.

## Next Steps

Start Milestone 0 in the roadmap: freeze the v1 support/API contract, restore the green baseline, repair release packaging, and convert workstreams into independently reviewable issues.
