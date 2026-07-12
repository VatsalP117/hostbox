# Task

Repair Hostbox Milestone 0 release packaging so published artifacts are usable and accurately describe the supported distribution model.

## Scope

- Stop publishing the known-unusable `CGO_ENABLED=0` API server binary.
- Continue publishing the server as multi-architecture Docker images.
- Publish CLI archives for Linux and macOS on amd64 and arm64.
- Give every CLI archive a stable `hostbox` executable payload.
- Verify release checksums before the installer extracts or installs the CLI.
- Keep release filenames and installer expectations aligned.

## Constraints

- Do not publish, tag, push, or commit.
- Do not modify application code, test workflow, dashboard, marketing site, or root ignore files.
