#!/usr/bin/env bash
#
# Smoke test for claude-workspace.
#
# Launches a fresh Ubuntu 24.04 VM (Multipass) or container (Docker),
# transfers the cross-compiled binary, and exercises setup -> attach -> doctor
# end-to-end.
#
# Usage:
#   bash scripts/smoke-test.sh [OPTIONS]
#
# Options:
#   --docker           Use Docker instead of Multipass (for CI / no nested virt)
#   --keep             Don't delete the VM/container on exit (for debugging)
#   --reuse            Reuse an existing VM/container instead of recreating
#   --skip-claude-cli  Stub the claude binary instead of running the real installer
#   --name <vm>        Override VM/container name (default: claude-workspace-smoke)

set -euo pipefail

# ---------- defaults ----------
MODE="multipass"
VM_NAME="claude-workspace-smoke"
KEEP=false
REUSE=false
SKIP_CLAUDE_CLI=false
BINARY_OUT="/tmp/claude-workspace-linux"
PASS_COUNT=0
FAIL_COUNT=0

# ---------- colors ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# ---------- summary (GitHub Job Summary) ----------
SUMMARY=""
BINARY_SIZE=""

summary_append() {
    SUMMARY="${SUMMARY}$1"$'\n'
}

summary_phase() {
    summary_append ""
    summary_append "## Phase $1: $2"
}

write_summary() {
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        local TOTAL=$((PASS_COUNT + FAIL_COUNT))
        {
            echo "# Smoke Test Results"
            echo ""
            echo "| Result | Count |"
            echo "|--------|-------|"
            echo "| Passed | ${PASS_COUNT} |"
            echo "| Failed | ${FAIL_COUNT} |"
            echo "| Total  | ${TOTAL} |"
            echo ""
            echo "## Environment"
            echo "- **Mode:** ${MODE}"
            echo "- **Target:** linux/${TARGET_GOARCH}"
            echo "- **Binary size:** ${BINARY_SIZE}"
            echo ""
            echo "$SUMMARY"
        } >> "$GITHUB_STEP_SUMMARY"
    fi
}

# ---------- parse flags ----------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --docker)        MODE="docker";       shift ;;
        --keep)          KEEP=true;           shift ;;
        --reuse)         REUSE=true;          shift ;;
        --skip-claude-cli) SKIP_CLAUDE_CLI=true; shift ;;
        --name)          VM_NAME="$2";        shift 2 ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# ---------- helpers ----------
assert_pass() {
    local desc="$1"
    PASS_COUNT=$((PASS_COUNT + 1))
    echo -e "  ${GREEN}[PASS]${NC} $desc"
    summary_append "- :white_check_mark: $desc"
}

assert_fail() {
    local desc="$1"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo -e "  ${RED}[FAIL]${NC} $desc"
    summary_append "- :x: $desc"
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

# Run a command as root inside the VM/container (for provisioning only)
root_exec() {
    if [[ "$MODE" == "docker" ]]; then
        docker exec "$VM_NAME" bash -c "$1"
    else
        multipass exec "$VM_NAME" -- sudo bash -c "$1"
    fi
}

cleanup() {
    if [[ "$KEEP" == true ]]; then
        if [[ "$MODE" == "docker" ]]; then
            echo -e "\n${YELLOW}--keep set. Container '${VM_NAME}' preserved.${NC}"
            echo "  Inspect:  docker exec -it ${VM_NAME} bash"
            echo "  Delete:   docker rm -f ${VM_NAME}"
        else
            echo -e "\n${YELLOW}--keep set. VM '${VM_NAME}' preserved.${NC}"
            echo "  Inspect:  multipass shell ${VM_NAME}"
            echo "  Delete:   multipass delete --purge ${VM_NAME}"
        fi
    else
        if [[ "$MODE" == "docker" ]]; then
            echo -e "\nCleaning up container '${VM_NAME}'..."
            docker rm -f "$VM_NAME" 2>/dev/null || true
        else
            echo -e "\nCleaning up VM '${VM_NAME}'..."
            multipass delete --purge "$VM_NAME" 2>/dev/null || true
        fi
    fi
}

# ========== Phase 1: Preflight ==========
echo -e "\n${BOLD}=== Phase 1: Preflight ===${NC}"
echo "  Mode: ${MODE}"

if [[ "$MODE" == "docker" ]]; then
    if ! command -v docker &>/dev/null; then
        echo -e "${RED}Error: docker not found. Install Docker first.${NC}"
        exit 1
    fi
    echo "  docker: $(docker --version)"
else
    if ! command -v multipass &>/dev/null; then
        echo -e "${RED}Error: multipass not found. Install with: brew install multipass${NC}"
        exit 1
    fi
    echo "  multipass: $(multipass version | head -1)"
fi

if ! command -v go &>/dev/null; then
    echo -e "${RED}Error: go not found. Install Go first.${NC}"
    exit 1
fi
echo "  go: $(go version)"

# ========== Phase 2: Cross-compile ==========
echo -e "\n${BOLD}=== Phase 2: Cross-compile ===${NC}"

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
echo "  Project: $PROJECT_DIR"

# Detect target architecture based on host
HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
    arm64|aarch64) TARGET_GOARCH="arm64" ;;
    x86_64|amd64)  TARGET_GOARCH="amd64" ;;
    *)             echo "  Unsupported host architecture: $HOST_ARCH"; exit 1 ;;
esac

echo "  Target:  linux/${TARGET_GOARCH} -> $BINARY_OUT"

GOOS=linux GOARCH="$TARGET_GOARCH" go build -ldflags "-s -w" -o "$BINARY_OUT" "$PROJECT_DIR"
BINARY_SIZE="$(ls -lh "$BINARY_OUT" | awk '{print $5}')"
echo "  Built: ${BINARY_SIZE}"

# ========== Phase 3: VM/Container lifecycle ==========
echo -e "\n${BOLD}=== Phase 3: VM/Container lifecycle ===${NC}"

# Check if VM/container already exists
VM_EXISTS=false
if [[ "$MODE" == "docker" ]]; then
    if docker ps -a --format '{{.Names}}' | grep -q "^${VM_NAME}$"; then
        VM_EXISTS=true
    fi
else
    if multipass info "$VM_NAME" &>/dev/null; then
        VM_EXISTS=true
    fi
fi

if [[ "$VM_EXISTS" == true && "$REUSE" == false ]]; then
    if [[ "$MODE" == "docker" ]]; then
        echo "  Removing existing container '$VM_NAME'..."
        docker rm -f "$VM_NAME" >/dev/null 2>&1
    else
        echo "  Deleting existing VM '$VM_NAME'..."
        multipass delete --purge "$VM_NAME"
    fi
    VM_EXISTS=false
fi

if [[ "$VM_EXISTS" == false ]]; then
    if [[ "$MODE" == "docker" ]]; then
        echo "  Starting container '$VM_NAME' (Ubuntu 24.04)..."
        docker run -d --name "$VM_NAME" ubuntu:24.04 sleep infinity >/dev/null
    else
        echo "  Launching VM '$VM_NAME' (Ubuntu 24.04, 2 CPUs, 4G RAM, 10G disk)..."
        multipass launch 24.04 --name "$VM_NAME" --cpus 2 --memory 4G --disk 10G
    fi
else
    if [[ "$MODE" == "docker" ]]; then
        echo "  Reusing existing container '$VM_NAME'."
        docker start "$VM_NAME" 2>/dev/null || true
    else
        echo "  Reusing existing VM '$VM_NAME'."
        multipass start "$VM_NAME" 2>/dev/null || true
    fi
fi

trap cleanup EXIT

if [[ "$MODE" == "docker" ]]; then
    echo "  Container ready: $(docker inspect --format '{{.State.Status}}' "$VM_NAME")"
else
    echo "  VM ready: $(multipass info "$VM_NAME" --format csv | tail -1 | cut -d, -f3)"
fi

# ========== Phase 4: Provision the VM/Container ==========
echo -e "\n${BOLD}=== Phase 4: Provision ===${NC}"

if [[ "$MODE" == "docker" ]]; then
    # Create ubuntu user in the container so /home/ubuntu paths work
    echo "  Creating ubuntu user..."
    docker exec "$VM_NAME" bash -c "useradd -m -s /bin/bash ubuntu" 2>/dev/null || true

    # Transfer binary via docker cp
    echo "  Transferring binary..."
    docker cp "$BINARY_OUT" "${VM_NAME}:/home/ubuntu/claude-workspace"
else
    # Transfer binary via multipass
    echo "  Transferring binary..."
    multipass transfer "$BINARY_OUT" "${VM_NAME}:/home/ubuntu/claude-workspace"
fi

# Install binary to PATH
echo "  Installing binary to /usr/local/bin..."
root_exec "cp /home/ubuntu/claude-workspace /usr/local/bin/claude-workspace && chmod +x /usr/local/bin/claude-workspace"

# Install prerequisites
echo "  Installing prerequisites (git, curl)..."
root_exec "apt-get update -qq && apt-get install -y -qq git curl python3 >/dev/null 2>&1"

# Pre-seed ~/.claude.json so setup skips interactive API key flow
echo "  Pre-seeding ~/.claude.json..."
vm_exec 'cat > /home/ubuntu/.claude.json << '\''SEED'\''
{"oauthAccount":{"email":"smoke-test@example.com"}}
SEED'

# Stub claude CLI if requested
if [[ "$SKIP_CLAUDE_CLI" == true ]]; then
    echo "  Creating stub claude CLI..."
    root_exec 'tee /usr/local/bin/claude > /dev/null << '\''STUB'\''
#!/bin/bash
if [[ "$1" == "--version" ]]; then
    echo "claude 1.0.0-stub"
else
    echo "stub: $*"
fi
STUB
chmod +x /usr/local/bin/claude'
fi

echo "  Provision complete."

# ========== Phase 5: Run setup ==========
echo -e "\n${BOLD}=== Phase 5: claude-workspace setup ===${NC}"

SETUP_OUTPUT=$(vm_exec "claude-workspace setup" 2>&1) || true
echo "$SETUP_OUTPUT" | sed 's/^/  | /'

echo ""
echo "  Assertions:"
summary_phase 5 "claude-workspace setup"

# Assert ~/.claude/settings.json exists
assert "~/.claude/settings.json exists" \
    vm_exec_quiet "test -f /home/ubuntu/.claude/settings.json"

# Assert ~/.claude/CLAUDE.md exists
assert "~/.claude/CLAUDE.md exists" \
    vm_exec_quiet "test -f /home/ubuntu/.claude/CLAUDE.md"

# Assert binary is executable in PATH
assert "claude-workspace is executable in PATH" \
    vm_exec_quiet "test -x /usr/local/bin/claude-workspace"

# ========== Phase 6: Run attach ==========
echo -e "\n${BOLD}=== Phase 6: claude-workspace attach ===${NC}"

# Create a dummy git repo
echo "  Creating test project..."
vm_exec "mkdir -p /home/ubuntu/test-project && cd /home/ubuntu/test-project && git init && git config user.email test@example.com && git config user.name Test && touch README.md && git add . && git commit -m 'init' -q"

# Run attach
ATTACH_OUTPUT=$(vm_exec "claude-workspace attach /home/ubuntu/test-project" 2>&1) || true
echo "$ATTACH_OUTPUT" | sed 's/^/  | /'

echo ""
echo "  Assertions:"
summary_phase 6 "claude-workspace attach"

PROJECT="/home/ubuntu/test-project"

# .claude/settings.json
assert ".claude/settings.json exists" \
    vm_exec_quiet "test -f ${PROJECT}/.claude/settings.json"

# .claude/CLAUDE.md
assert ".claude/CLAUDE.md exists" \
    vm_exec_quiet "test -f ${PROJECT}/.claude/CLAUDE.md"

# .mcp.json exists and is valid JSON
assert ".mcp.json exists and is valid JSON" \
    vm_exec_quiet "python3 -c \"import json; json.load(open('${PROJECT}/.mcp.json'))\""

# Hook scripts exist and are executable
for hook in auto-format.sh block-dangerous-commands.sh enforce-branch-policy.sh validate-secrets.sh; do
    assert "hook ${hook} exists and is executable" \
        vm_exec_quiet "test -x ${PROJECT}/.claude/hooks/${hook}"
done

# .claude/agents/ is non-empty
assert ".claude/agents/ is non-empty" \
    vm_exec_quiet "test -d ${PROJECT}/.claude/agents && [ \"\$(ls -A ${PROJECT}/.claude/agents)\" ]"

# .claude/skills/ is non-empty
assert ".claude/skills/ is non-empty" \
    vm_exec_quiet "test -d ${PROJECT}/.claude/skills && [ \"\$(ls -A ${PROJECT}/.claude/skills)\" ]"

# ========== Phase 7: Run doctor ==========
echo -e "\n${BOLD}=== Phase 7: claude-workspace doctor ===${NC}"

DOCTOR_OUTPUT=$(vm_exec "cd ${PROJECT} && ANTHROPIC_API_KEY=sk-fake claude-workspace doctor" 2>&1) || true
echo "$DOCTOR_OUTPUT" | sed 's/^/  | /'

echo ""
echo "  Assertions:"
summary_phase 7 "claude-workspace doctor"

# Assert no [FAIL] lines in output
FAIL_LINES=$(echo "$DOCTOR_OUTPUT" | grep -c '\[FAIL\]' || true)
if [[ "$FAIL_LINES" -eq 0 ]]; then
    assert_pass "doctor output contains no [FAIL] lines"
else
    assert_fail "doctor output contains ${FAIL_LINES} [FAIL] line(s)"
fi

# ========== Phase 8: Summary ==========
echo -e "\n${BOLD}=== Summary ===${NC}"

TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo -e "  Total: ${TOTAL}  ${GREEN}Passed: ${PASS_COUNT}${NC}  ${RED}Failed: ${FAIL_COUNT}${NC}"

write_summary

if [[ "$FAIL_COUNT" -gt 0 ]]; then
    echo -e "\n${RED}SMOKE TEST FAILED${NC}"
    exit 1
else
    echo -e "\n${GREEN}SMOKE TEST PASSED${NC}"
    exit 0
fi
