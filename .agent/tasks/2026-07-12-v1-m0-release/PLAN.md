# Implementation Plan

## 1. Task Summary

Make GitHub Releases a reliable CLI distribution channel while retaining container images as the only supported server artifact.

## 2. Current System Understanding

The release matrix builds API and CLI binaries with CGO disabled. The API requires CGO-backed SQLite and is unusable. CLI archives contain platform-suffixed executables, but the installer expects an extracted file named `hostbox`. Although releases contain a checksum manifest, the installer does not download or verify it.

## 3. Scope

### In Scope

- Release workflow CLI matrix and artifacts.
- CLI installer archive/checksum behavior.
- Focused installer validation.

### Out of Scope

- Server packaging outside the existing Docker image jobs.
- Application behavior or dependencies.
- Release publication.

## 4. Proposed Technical Approach

Remove the standalone API build from the release matrix and make the job CLI-specific. Cross-compile the CLI as `hostbox`, package it under the existing platform-specific archive names, and checksum those archives. Update the installer to download the checksum manifest, select the exact archive entry, calculate SHA-256 with the native Linux or macOS utility, compare before extraction, validate the payload, and install with executable permissions.

## 5. Step-by-Step Execution Plan

1. Narrow the binary matrix to CLI builds.
2. Align the archive payload and installer binary name.
3. Add checksum download and verification before extraction.
4. Add a hermetic installer success/failure test.
5. Validate shell syntax, workflow syntax, release shape, and all four cross-builds.

## 6. Test Plan

- `bash -n scripts/install-cli.sh scripts/tests/test-install-cli.sh`
- `bash scripts/tests/test-install-cli.sh`
- `bash scripts/tests/test-release-packaging.sh`
- Parse `.github/workflows/release.yml` as YAML.
- Cross-build `./cmd/cli` for linux/darwin and amd64/arm64 with CGO disabled.
- Inspect the diff and run `git diff --check`.

## 7. Acceptance Criteria

- No standalone server archive is built or attached.
- Existing multi-architecture server Docker image publication remains intact.
- Four CLI archives are produced with matching installer filenames.
- Each archive contains an executable named `hostbox`.
- An absent, malformed, or mismatched checksum prevents installation.

## 8. Risks and Guardrails

- SHA tooling differs between Linux and macOS; support `sha256sum` and `shasum -a 256`.
- Keep Bash 3.2 compatibility for the macOS-provided shell.
- Do not publish a release during validation.

## 9. Executor Instructions

Restrict changes to the assigned workflow, installer, focused tests, and task artifacts. Report validation results and remaining tradeoffs without committing.
