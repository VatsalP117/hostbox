#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
WORKFLOW="${ROOT_DIR}/.github/workflows/release.yml"
INSTALLER="${ROOT_DIR}/scripts/install-cli.sh"

for suffix in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do
    grep -Fq -- "suffix: ${suffix}" "$WORKFLOW"
done

if grep -Fq -- "./cmd/api" "$WORKFLOW"; then
    echo "release workflow still builds a standalone API server" >&2
    exit 1
fi

grep -Fq -- "file: docker/Dockerfile" "$WORKFLOW"
grep -Fq -- "platforms: linux/amd64,linux/arm64" "$WORKFLOW"
grep -Fq -- "-o hostbox ./cmd/cli" "$WORKFLOW"
grep -Fq -- 'tar czf hostbox-cli-${{ matrix.suffix }}.tar.gz hostbox' "$WORKFLOW"
grep -Fq -- 'filename="hostbox-cli-${OS}-${ARCH}.tar.gz"' "$INSTALLER"
grep -Fq -- 'checksums_url=' "$INSTALLER"

echo "Release packaging contract tests passed"
