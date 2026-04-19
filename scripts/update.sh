#!/usr/bin/env bash
set -euo pipefail

HOSTBOX_DIR="${HOSTBOX_DIR:-/opt/hostbox}"
HOSTBOX_BRANCH="${HOSTBOX_BRANCH:-main}"
RESET_DATA=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}⚠${NC} $*"; }
fatal() { echo -e "${RED}✗${NC} $*" >&2; exit 1; }

usage() {
    cat <<EOF
Usage: bash scripts/update.sh [--fresh]

  --fresh   Recreate Hostbox from current git state and clear runtime data
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --fresh)
            RESET_DATA=1
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            fatal "Unknown argument: $1"
            ;;
    esac
    shift
done

[ -d "${HOSTBOX_DIR}/.git" ] || fatal "No git checkout found at ${HOSTBOX_DIR}"
[ -f "${HOSTBOX_DIR}/.env" ] || fatal "Missing ${HOSTBOX_DIR}/.env"

cd "${HOSTBOX_DIR}"

info "Updating Hostbox source..."
git fetch --depth 1 origin "${HOSTBOX_BRANCH}"
git checkout -B "${HOSTBOX_BRANCH}" "origin/${HOSTBOX_BRANCH}"
ok "Source updated"

mkdir -p "${HOSTBOX_DIR}"/{data/backups,deployments,logs,cache,tmp}
chown -R 1000:1000 \
    "${HOSTBOX_DIR}/data" \
    "${HOSTBOX_DIR}/deployments" \
    "${HOSTBOX_DIR}/logs" \
    "${HOSTBOX_DIR}/cache" \
    "${HOSTBOX_DIR}/tmp"

if [ "${RESET_DATA}" -eq 1 ]; then
    warn "Resetting runtime data under ${HOSTBOX_DIR}"
    docker compose down --remove-orphans || true
    rm -f "${HOSTBOX_DIR}/data/hostbox.db" "${HOSTBOX_DIR}/data/hostbox.db-shm" "${HOSTBOX_DIR}/data/hostbox.db-wal"
    rm -rf "${HOSTBOX_DIR}/deployments"/* "${HOSTBOX_DIR}/logs"/* "${HOSTBOX_DIR}/cache"/* "${HOSTBOX_DIR}/tmp"/*
    ok "Runtime data cleared"
fi

info "Rebuilding Hostbox..."
docker compose up -d --build --remove-orphans
ok "Hostbox updated"
