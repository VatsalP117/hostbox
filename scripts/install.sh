#!/usr/bin/env bash
set -euo pipefail

# Hostbox Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/VatsalP117/hostbox/main/scripts/install.sh | bash

HOSTBOX_DIR="${HOSTBOX_DIR:-/opt/hostbox}"
REPO="VatsalP117/hostbox"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}⚠${NC} $*"; }
err()   { echo -e "${RED}✗${NC} $*" >&2; }
fatal() { err "$@"; exit 1; }

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        info "Detected: ${PRETTY_NAME:-$ID}"
    else
        warn "Could not detect OS — proceeding anyway"
    fi
}

detect_docker_gid() {
    if [ ! -S /var/run/docker.sock ]; then
        DOCKER_GID=998
        warn "Docker socket not found at /var/run/docker.sock; defaulting DOCKER_GID=${DOCKER_GID}"
        return
    fi

    DOCKER_GID="$(stat -c '%g' /var/run/docker.sock 2>/dev/null || stat -f '%g' /var/run/docker.sock 2>/dev/null || true)"
    if [ -z "${DOCKER_GID}" ]; then
        DOCKER_GID=998
        warn "Could not detect Docker socket group; defaulting DOCKER_GID=${DOCKER_GID}"
        return
    fi

    ok "Detected Docker socket group: ${DOCKER_GID}"
}

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        fatal "This script must be run as root (or with sudo)"
    fi
}

check_prerequisites() {
    local missing=()

    command -v docker &>/dev/null || missing+=("docker")
    command -v curl &>/dev/null   || missing+=("curl")
    command -v git &>/dev/null    || missing+=("git")

    if [ ${#missing[@]} -gt 0 ]; then
        info "Missing prerequisites: ${missing[*]}"
        install_prerequisites "${missing[@]}"
    else
        ok "All prerequisites found"
    fi

    # Check docker compose (v2 plugin)
    if ! docker compose version &>/dev/null; then
        info "Installing docker compose plugin..."
        apt-get install -y -qq docker-compose-plugin 2>/dev/null || true
    fi
}

install_prerequisites() {
    for pkg in "$@"; do
        case "$pkg" in
            docker)
                info "Installing Docker..."
                curl -fsSL https://get.docker.com | sh
                systemctl enable --now docker
                ok "Docker installed"
                ;;
            curl)
                apt-get update -qq && apt-get install -y -qq curl
                ;;
            git)
                apt-get update -qq && apt-get install -y -qq git
                ;;
        esac
    done
}

setup_directory() {
    info "Setting up ${HOSTBOX_DIR}..."
    mkdir -p "${HOSTBOX_DIR}"/{data/backups,deployments,logs,cache,tmp}
    ok "Directory structure created"
}

download_source() {
    info "Downloading Hostbox source..."
    local branch="${HOSTBOX_BRANCH:-main}"
    local repo_url="https://github.com/${REPO}.git"

    if [ -d "${HOSTBOX_DIR}/.git" ]; then
        git -C "${HOSTBOX_DIR}" fetch --depth 1 origin "${branch}"
        git -C "${HOSTBOX_DIR}" checkout -B "${branch}" "origin/${branch}"
        ok "Hostbox source updated"
        return
    fi

    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "${tmpdir}"' RETURN

    git clone --depth 1 --branch "${branch}" "${repo_url}" "${tmpdir}/repo"
    cp -a "${tmpdir}/repo/." "${HOSTBOX_DIR}/"
    ok "Hostbox source downloaded"
}

configure() {
    echo ""
    echo -e "${CYAN}═══ Hostbox Configuration ═══${NC}"
    echo ""

    read -rp "Root domain (e.g., example.com): " DOMAIN
    [ -z "$DOMAIN" ] && fatal "Domain is required"

    read -rp "Dashboard host [hostbox.${DOMAIN}]: " DASHBOARD_INPUT
    DASHBOARD_DOMAIN="${DASHBOARD_INPUT:-hostbox.${DOMAIN}}"

    read -rp "Email for SSL certificates (Let's Encrypt): " ACME_EMAIL
    [ -z "$ACME_EMAIL" ] && fatal "Email is required"

    read -rp "DNS provider for wildcard SSL [none/cloudflare/route53/digitalocean]: " DNS_PROVIDER
    DNS_PROVIDER="${DNS_PROVIDER:-none}"

    DNS_CONFIG=""
    if [ "$DNS_PROVIDER" = "cloudflare" ]; then
        read -rp "Cloudflare API token: " CF_TOKEN
        DNS_CONFIG="CF_API_TOKEN=${CF_TOKEN}"
    elif [ "$DNS_PROVIDER" = "route53" ]; then
        read -rp "AWS Access Key ID: " AWS_KEY
        read -rp "AWS Secret Access Key: " AWS_SECRET
        read -rp "AWS Hosted Zone ID: " AWS_ZONE_ID
        DNS_CONFIG="AWS_ACCESS_KEY_ID=${AWS_KEY}
AWS_SECRET_ACCESS_KEY=${AWS_SECRET}
AWS_HOSTED_ZONE_ID=${AWS_ZONE_ID}"
    elif [ "$DNS_PROVIDER" = "digitalocean" ]; then
        read -rp "DigitalOcean API token: " DO_TOKEN
        DNS_CONFIG="DO_AUTH_TOKEN=${DO_TOKEN}"
    fi

    echo ""
}

generate_secrets() {
    JWT_SECRET=$(openssl rand -hex 32)
    ENCRYPTION_KEY=$(openssl rand -hex 32)
    WEBHOOK_SECRET=$(openssl rand -hex 32)
}

generate_env() {
    local env_file="${HOSTBOX_DIR}/.env"

    cat > "$env_file" <<EOF
# Hostbox Configuration
# Generated on $(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Platform
PLATFORM_DOMAIN=${DOMAIN}
DASHBOARD_DOMAIN=${DASHBOARD_DOMAIN}
PLATFORM_HTTPS=true
PLATFORM_NAME=Hostbox

# Authentication
JWT_SECRET=${JWT_SECRET}
ENCRYPTION_KEY=${ENCRYPTION_KEY}

# Database
DATABASE_PATH=${HOSTBOX_DIR}/data/hostbox.db
DEPLOYMENTS_DIR=${HOSTBOX_DIR}/deployments
LOGS_DIR=${HOSTBOX_DIR}/logs
CACHE_DIR=${HOSTBOX_DIR}/cache
BACKUP_DIR=${HOSTBOX_DIR}/data/backups
CLONE_BASE_DIR=${HOSTBOX_DIR}/tmp
DEPLOYMENT_BASE_DIR=${HOSTBOX_DIR}/deployments
LOG_BASE_DIR=${HOSTBOX_DIR}/logs
DOCKER_GID=${DOCKER_GID}

# GitHub App (configure after installation via setup wizard)
GITHUB_APP_ID=
GITHUB_APP_SLUG=
GITHUB_APP_PEM=
GITHUB_WEBHOOK_SECRET=${WEBHOOK_SECRET}

# SSL / ACME
ACME_EMAIL=${ACME_EMAIL}

# DNS Provider
DNS_PROVIDER=${DNS_PROVIDER}
${DNS_CONFIG}

# Logging
LOG_LEVEL=info
EOF

    chmod 600 "$env_file"
    ok "Environment file generated"
}

start_hostbox() {
    info "Starting Hostbox..."
    cd "${HOSTBOX_DIR}"
    docker compose up -d --build --remove-orphans
    ok "Containers started"
}

wait_for_health() {
    info "Waiting for Hostbox to be ready..."
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if docker compose -f "${HOSTBOX_DIR}/docker-compose.yml" exec -T hostbox wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/api/v1/health &>/dev/null; then
            ok "Hostbox is ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    warn "Health check timed out — check logs with: docker compose -f ${HOSTBOX_DIR}/docker-compose.yml logs"
    return 1
}

detect_server_ip() {
    # Try to detect the server's public IP
    SERVER_IP=$(curl -sf https://api.ipify.org 2>/dev/null || curl -sf https://ifconfig.me 2>/dev/null || echo "<your-server-ip>")
}

print_success() {
    detect_server_ip

    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  Hostbox installed successfully! 🚀${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  Dashboard:  ${CYAN}https://${DASHBOARD_DOMAIN}${NC}"
    echo ""
    echo -e "  ${YELLOW}DNS Setup Required:${NC}"
    echo -e "  Point these records to ${SERVER_IP}:"
    echo -e "    A     ${DOMAIN}        → ${SERVER_IP}"
    echo -e "    A     *.${DOMAIN}      → ${SERVER_IP}"

    # Check if the dashboard host falls under the wildcard zone
    CASE_INSENSITIVE_DOMAIN="${DOMAIN,,}"
    CASE_INSENSITIVE_DASHDOMAIN="${DASHBOARD_DOMAIN,,}"
    if [[ "${CASE_INSENSITIVE_DASHDOMAIN}" == *".${CASE_INSENSITIVE_DOMAIN}" ]]; then
        echo -e "  (${DASHBOARD_DOMAIN} is covered by the wildcard record)"
    else
        echo -e "  ${YELLOW}⚠${NC}  Dashboard host ${DASHBOARD_DOMAIN} is outside ${DOMAIN} — add a separate DNS record:"
        echo -e "    A     ${DASHBOARD_DOMAIN}  → ${SERVER_IP}"
    fi
    echo ""
    echo -e "  ${YELLOW}Next Steps:${NC}"
    echo -e "  1. Configure DNS records above"
    echo -e "  2. Open https://${DASHBOARD_DOMAIN} to create your admin account"
    echo -e "  3. (Optional) Configure GitHub App in Settings"
    echo ""
    echo -e "  ${YELLOW}Useful Commands:${NC}"
    echo -e "    Logs:    cd ${HOSTBOX_DIR} && docker compose logs -f"
    echo -e "    Stop:    cd ${HOSTBOX_DIR} && docker compose down"
    echo -e "    Update:  cd ${HOSTBOX_DIR} && bash scripts/update.sh"
    echo -e "    Fresh:   cd ${HOSTBOX_DIR} && bash scripts/update.sh --fresh"
    echo -e "    Backup:  hostbox admin backup"
    echo ""
}

main() {
    echo ""
    echo -e "${CYAN}Hostbox Installer${NC}"
    echo ""

    check_root
    detect_os
    check_prerequisites
    detect_docker_gid
    setup_directory
    download_source
    configure
    generate_secrets
    generate_env
    start_hostbox
    wait_for_health || true
    print_success
}

main "$@"
