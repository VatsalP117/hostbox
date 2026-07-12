#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

FAKE_BIN="${TEST_DIR}/bin"
FIXTURES="${TEST_DIR}/fixtures"
INSTALL_DIR="${TEST_DIR}/install"
mkdir -p "$FAKE_BIN" "$FIXTURES" "$INSTALL_DIR"

cat > "${FIXTURES}/hostbox" <<'EOF'
#!/usr/bin/env sh
echo "hostbox test binary"
EOF
chmod +x "${FIXTURES}/hostbox"
tar -czf "${FIXTURES}/hostbox-cli-linux-amd64.tar.gz" -C "$FIXTURES" hostbox

if command -v sha256sum >/dev/null 2>&1; then
    CHECKSUM=$(sha256sum "${FIXTURES}/hostbox-cli-linux-amd64.tar.gz" | awk '{print $1}')
else
    CHECKSUM=$(shasum -a 256 "${FIXTURES}/hostbox-cli-linux-amd64.tar.gz" | awk '{print $1}')
fi
printf '%s  %s\n' "$CHECKSUM" "hostbox-cli-linux-amd64.tar.gz" > "${FIXTURES}/checksums.txt"

cat > "${FAKE_BIN}/uname" <<'EOF'
#!/usr/bin/env sh
case "$1" in
    -s) echo Linux ;;
    -m) echo x86_64 ;;
    *) exit 1 ;;
esac
EOF

cat > "${FAKE_BIN}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

url=""
output=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        -o)
            output=$2
            shift 2
            ;;
        -*) shift ;;
        *)
            url=$1
            shift
            ;;
    esac
done

case "$url" in
    */releases/latest)
        printf '{"tag_name":"v1.0.0"}\n'
        ;;
    */hostbox-cli-linux-amd64.tar.gz)
        cp "${HOSTBOX_TEST_FIXTURES}/hostbox-cli-linux-amd64.tar.gz" "$output"
        ;;
    */checksums.txt)
        cp "${HOSTBOX_TEST_CHECKSUMS:-${HOSTBOX_TEST_FIXTURES}/checksums.txt}" "$output"
        ;;
    *)
        echo "unexpected URL: $url" >&2
        exit 1
        ;;
esac
EOF
chmod +x "${FAKE_BIN}/uname" "${FAKE_BIN}/curl"

PATH="${FAKE_BIN}:$PATH" \
    HOSTBOX_TEST_FIXTURES="$FIXTURES" \
    INSTALL_DIR="$INSTALL_DIR" \
    bash "${ROOT_DIR}/scripts/install-cli.sh" >/dev/null

if [ ! -x "${INSTALL_DIR}/hostbox" ]; then
    echo "installer did not create an executable hostbox binary" >&2
    exit 1
fi

if [ "$("${INSTALL_DIR}/hostbox")" != "hostbox test binary" ]; then
    echo "installed binary is not the archive payload" >&2
    exit 1
fi

printf '%064d  %s\n' 0 "hostbox-cli-linux-amd64.tar.gz" > "${FIXTURES}/bad-checksums.txt"
rm -f "${INSTALL_DIR}/hostbox"

if PATH="${FAKE_BIN}:$PATH" \
    HOSTBOX_TEST_FIXTURES="$FIXTURES" \
    HOSTBOX_TEST_CHECKSUMS="${FIXTURES}/bad-checksums.txt" \
    INSTALL_DIR="$INSTALL_DIR" \
    bash "${ROOT_DIR}/scripts/install-cli.sh" >"${TEST_DIR}/bad-checksum.log" 2>&1; then
    echo "installer accepted an invalid checksum" >&2
    exit 1
fi

if [ -e "${INSTALL_DIR}/hostbox" ]; then
    echo "installer wrote a binary after checksum verification failed" >&2
    exit 1
fi

grep -q "Checksum verification failed" "${TEST_DIR}/bad-checksum.log"
echo "CLI installer tests passed"
