# Implementation Report

## Summary

Completed the first v1 Milestone 0 slice through three parallel executor tracks and an orchestrator integration/review pass.

- Restored reproducible dashboard and marketing checks and added marketing to CI.
- Removed dashboard production dependency findings; retained/documented only major-migration build-tool findings.
- Removed unusable standalone server archives while retaining multi-architecture Docker server images.
- Fixed checksum-verified CLI archive installation and added packaging contract tests to CI.
- Repaired existing CLI deployment/project/domain/env/auth/backup contracts and added focused tests.
- Removed tracked machine-generated binaries/compiler outputs and ignored their regenerated local copies.
- Updated the v1 roadmap with Milestone 0 progress.

## Files Changed

- CI/release: `.github/workflows/test.yml`, `.github/workflows/release.yml`
- Packaging: `scripts/install-cli.sh`, `scripts/tests/*`
- CLI: focused files under `cmd/cli/cmd` and `cmd/cli/internal/client`, plus tests
- Dashboard/marketing manifests, lockfiles, lint/auth/log fixes
- `.gitignore`, removal of `api` and generated `web` outputs
- Roadmap and engineering-pipeline task artifacts

## Commands Run

- Go tests, vet, and API/CLI builds.
- Dashboard and marketing clean installs and production builds.
- Dashboard lint and TypeScript checks.
- Production-only npm audits for both static web deliverables.
- CLI HTTP contract/unit tests.
- Shell syntax, hermetic installer, tampered-checksum, and release-contract tests.
- Release/test workflow YAML parsing and Compose configuration validation.
- `git diff --check`.

## Tests

Passed:

- `go test ./cmd/... ./internal/... ./migrations`
- `go vet ./cmd/... ./internal/... ./migrations`
- `go build ./cmd/api ./cmd/cli`
- Dashboard `npm ci`, lint (zero errors), TypeScript, build, and production audit (zero findings)
- Marketing `npm ci`, build, and production audit (zero findings; build tooling is development-only)
- CLI installer/release contract tests
- Workflow YAML and Compose validation

The full development audit still reports major-migration findings in Vite/Astro tooling. They are documented and not bypassed; clearing them is a separate upgrade task.

## Deviations From Plan

- The auth lint fix was refined during integration to use `useCallback` plus complete effect dependencies rather than preserving mount values in refs.
- Static marketing packages were moved to `devDependencies`, accurately reflecting that only generated static output is deployed.
- Go verification used repository package roots after Node installs because `go test ./...` also traverses an unrelated Go fixture inside `node_modules`; clean CI jobs do not contain Node modules during Go tests.

## Known Risks

- No release/tag was published and no Docker image build was rerun locally; CI retains Docker build gates.
- Remaining development-tool advisories require validated major upgrades.
- CLI logs are snapshots rather than SSE follow mode, and CLI access tokens do not refresh.

## Next Steps

Proceed to Milestone 1: path/symlink isolation, exact-commit builds, deterministic deployment state transitions, framework/serving-mode persistence, cancellation, rollback/promote routing, and related integration tests.
