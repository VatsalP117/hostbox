#!/usr/bin/env bash
set -euo pipefail

# Hostbox CLI Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/vatsalpatel/hostbox/main/scripts/install-cli.sh | bash

REPO="vatsalpatel/hostbox"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="hostbox"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
fatal() { echo -e "${RED}✗${NC} $*" >&2; exit 1; }

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      fatal "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)   ARCH="amd64" ;;
        aarch64|arm64)  ARCH="arm64" ;;
        *)              fatal "Unsupported architecture: $ARCH" ;;
    esac

    info "Platform: ${OS}/${ARCH}"
}

get_latest_version() {
    info "Fetching latest version..."
    VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        fatal "Could not determine latest version"
    fi
    info "Latest version: ${VERSION}"
}

download_and_install() {
    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    local filename="hostbox-cli-${OS}-${ARCH}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"

    info "Downloading ${filename}..."
    if ! curl -fsSL "$url" -o "${tmpdir}/${filename}"; then
        fatal "Download failed. Check if release exists: ${url}"
    fi

    info "Installing to ${INSTALL_DIR}..."
    tar -xzf "${tmpdir}/${filename}" -C "$tmpdir"

    # Try to install to INSTALL_DIR, use sudo if needed
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "${tmpdir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
}

main() {
    echo ""
    echo -e "${CYAN}Hostbox CLI Installer${NC}"
    echo ""

    detect_platform
    get_latest_version
    download_and_install

    echo ""
    ok "${BINARY_NAME} ${VERSION} installed to ${INSTALL_DIR}/${BINARY_NAME}"
    echo -e "  Run '${BINARY_NAME} login' to get started"
    echo ""
}

main "$@"
