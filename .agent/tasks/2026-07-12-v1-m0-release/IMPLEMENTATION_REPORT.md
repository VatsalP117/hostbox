# Implementation Report

## Summary

- Replaced the release binary matrix with a CLI-only matrix for Linux and macOS on amd64 and arm64.
- Removed the unusable `CGO_ENABLED=0` standalone API server build and its release archives. The existing multi-architecture Docker image jobs remain the supported server distribution.
- Standardized each platform-specific CLI archive to contain one installable executable named `hostbox` while retaining the public archive names `hostbox-cli-<os>-<arch>.tar.gz`.
- Updated the installer to fetch `checksums.txt`, require an exact valid SHA-256 entry for the selected archive, verify it before extraction, validate the archive payload, and install it with mode `0755`.
- Added focused tests for successful installation, checksum-tampering rejection, platform coverage, Docker server packaging retention, and workflow/installer filename agreement.

## Files Changed

- `.github/workflows/release.yml`
- `scripts/install-cli.sh`
- `scripts/tests/test-install-cli.sh`
- `scripts/tests/test-release-packaging.sh`
- `.agent/tasks/2026-07-12-v1-m0-release/TASK.md`
- `.agent/tasks/2026-07-12-v1-m0-release/PLAN.md`
- `.agent/tasks/2026-07-12-v1-m0-release/IMPLEMENTATION_REPORT.md`

## Commands Run

- `bash -n scripts/install-cli.sh scripts/tests/test-install-cli.sh scripts/tests/test-release-packaging.sh`
- `bash scripts/tests/test-install-cli.sh`
- `bash scripts/tests/test-release-packaging.sh`
- `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/release.yml")'`
- Four `CGO_ENABLED=0 go build ./cmd/cli` cross-builds for `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`
- `file` against all four cross-built executables
- `git diff --check`

## Tests

- Shell syntax: passed.
- Hermetic installer success case: passed; installed archive payload is executable as `hostbox`.
- Tampered checksum case: passed; installer exits before writing the destination binary.
- Release packaging contract test: passed; verifies four targets, stable archive/payload naming, absence of an API binary build, and retention of multi-architecture Docker server builds.
- Release workflow YAML parse: passed.
- CLI cross-build matrix: passed; Linux outputs are static ELF executables and macOS outputs are Mach-O executables for their requested architectures.
- Whitespace/error diff validation: passed.

## Deviations From Plan

None.

## Known Risks

- The checksum manifest and archive are hosted by the same GitHub Release. Verification protects against corruption, partial downloads, and archive/manifest mismatch, but it is not an independent signing mechanism.
- The workflow was validated structurally and parsed locally, but was not executed by GitHub Actions because publishing, pushing, and tagging are outside this task.
- The server has no standalone native artifact after this change. This is deliberate: the existing GHCR Docker images are the supported server artifact until a portable CGO-enabled packaging design is implemented and verified.

## Next Steps

- Run the complete integrated Milestone 0 verification matrix after all parallel tracks are combined.
- Have the reviewer inspect the integrated diff and GitHub Actions result before any release tag is created.
