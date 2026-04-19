#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

for env_file in "${ROOT_DIR}/.env" "${ROOT_DIR}/.env.local"; do
  if [ -f "${env_file}" ]; then
    set -a
    # shellcheck disable=SC1090
    . "${env_file}"
    set +a
  fi
done

if [ -z "${STITCH_API_KEY:-}" ]; then
  echo "STITCH_API_KEY is not set. Add it to your local .env or export it before starting Copilot CLI." >&2
  exit 1
fi

exec env STITCH_API_KEY="${STITCH_API_KEY}" npx --yes @_davideast/stitch-mcp proxy
