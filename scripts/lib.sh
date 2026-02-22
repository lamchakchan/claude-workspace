#!/usr/bin/env bash
#
# Shared helper library for dev-env.sh and smoke-test.sh.
#
# Expects the caller to set these before sourcing:
#   MODE        — "docker" or "multipass"
#   VM_NAME     — container/VM name
#   BINARY_OUT  — path for cross-compiled binary

# ---------- colors ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# ---------- counters ----------
PASS_COUNT=0
FAIL_COUNT=0

# ---------- project root ----------
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ---------- VM execution ----------
vm_exec() {
    if [[ "$MODE" == "docker" ]]; then
        docker exec --user ubuntu -e HOME=/home/ubuntu "$VM_NAME" bash -c "$1"
    else
        multipass exec "$VM_NAME" -- bash -c "$1"
    fi
}

vm_exec_quiet() {
    vm_exec "$1" >/dev/null 2>&1
}

root_exec() {
    if [[ "$MODE" == "docker" ]]; then
        docker exec "$VM_NAME" bash -c "$1"
    else
        multipass exec "$VM_NAME" -- sudo bash -c "$1"
    fi
}

# ---------- cross-compilation ----------
cross_compile() {
    HOST_ARCH="$(uname -m)"
    case "$HOST_ARCH" in
        arm64|aarch64) TARGET_GOARCH="arm64" ;;
        x86_64|amd64)  TARGET_GOARCH="amd64" ;;
        *)             echo "Unsupported host architecture: $HOST_ARCH"; exit 1 ;;
    esac

    echo "  Cross-compiling for linux/${TARGET_GOARCH}..."
    GOOS=linux GOARCH="$TARGET_GOARCH" go build -ldflags "-s -w" -o "$BINARY_OUT" "$PROJECT_DIR"
    BINARY_SIZE="$(ls -lh "$BINARY_OUT" | awk '{print $5}')"
    echo "  Built: ${BINARY_SIZE}"
}

# ---------- binary transfer ----------
copy_binary() {
    if [[ "$MODE" == "docker" ]]; then
        docker cp "$BINARY_OUT" "${VM_NAME}:/home/ubuntu/claude-workspace"
    else
        multipass transfer "$BINARY_OUT" "${VM_NAME}:/home/ubuntu/claude-workspace"
    fi
    root_exec "cp /home/ubuntu/claude-workspace /usr/local/bin/claude-workspace && chmod +x /usr/local/bin/claude-workspace"
}

# ---------- assertions ----------
assert_pass() {
    local desc="$1"
    PASS_COUNT=$((PASS_COUNT + 1))
    echo -e "  ${GREEN}[PASS]${NC} $desc"
}

assert_fail() {
    local desc="$1"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo -e "  ${RED}[FAIL]${NC} $desc"
}

assert() {
    local desc="$1"
    shift
    if "$@"; then
        assert_pass "$desc"
    else
        assert_fail "$desc"
    fi
}
