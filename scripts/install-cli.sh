#!/usr/bin/env bash
set -euo pipefail

# Hostbox CLI Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/VatsalP117/hostbox/main/scripts/install-cli.sh | bash

REPO="VatsalP117/hostbox"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="hostbox"
TMPDIR_INSTALL=""

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
fatal() { echo -e "${RED}✗${NC} $*" >&2; exit 1; }

cleanup() {
    if [ -n "$TMPDIR_INSTALL" ]; then
        rm -rf "$TMPDIR_INSTALL"
    fi
}

trap cleanup EXIT

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
    TMPDIR_INSTALL=$(mktemp -d)

    local filename="hostbox-cli-${OS}-${ARCH}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"
    local checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    info "Downloading ${filename}..."
    if ! curl -fsSL "$url" -o "${TMPDIR_INSTALL}/${filename}"; then
        fatal "Download failed. Check if release exists: ${url}"
    fi

    info "Verifying checksum..."
    if ! curl -fsSL "$checksums_url" -o "${TMPDIR_INSTALL}/checksums.txt"; then
        fatal "Could not download release checksums"
    fi

    local expected_checksum
    expected_checksum=$(awk -v file="$filename" '$2 == file || $2 == ("*" file) { print $1; exit }' "${TMPDIR_INSTALL}/checksums.txt")
    if [[ ! "$expected_checksum" =~ ^[[:xdigit:]]{64}$ ]]; then
        fatal "No valid checksum found for ${filename}"
    fi

    local actual_checksum
    if command -v sha256sum >/dev/null 2>&1; then
        actual_checksum=$(sha256sum "${TMPDIR_INSTALL}/${filename}" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual_checksum=$(shasum -a 256 "${TMPDIR_INSTALL}/${filename}" | awk '{print $1}')
    else
        fatal "A SHA-256 tool is required (sha256sum or shasum)"
    fi

    expected_checksum=$(printf '%s' "$expected_checksum" | tr '[:upper:]' '[:lower:]')
    actual_checksum=$(printf '%s' "$actual_checksum" | tr '[:upper:]' '[:lower:]')
    if [ "$actual_checksum" != "$expected_checksum" ]; then
        fatal "Checksum verification failed for ${filename}"
    fi

    tar -xzf "${TMPDIR_INSTALL}/${filename}" -C "$TMPDIR_INSTALL"
    if [ ! -f "${TMPDIR_INSTALL}/${BINARY_NAME}" ]; then
        fatal "Release archive does not contain ${BINARY_NAME}"
    fi

    info "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        install -m 0755 "${TMPDIR_INSTALL}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo install -m 0755 "${TMPDIR_INSTALL}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi
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
