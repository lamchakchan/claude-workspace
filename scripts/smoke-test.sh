#!/usr/bin/env bash
#
# Smoke test for claude-workspace using Multipass.
#
# Launches a fresh Ubuntu 24.04 VM, transfers the cross-compiled binary,
# and exercises setup -> attach -> doctor end-to-end.
#
# Usage:
#   bash scripts/smoke-test.sh [OPTIONS]
#
# Options:
#   --keep             Don't delete the VM on exit (for debugging)
#   --reuse            Reuse an existing VM instead of recreating
#   --skip-claude-cli  Stub the claude binary instead of running the real installer
#   --name <vm>        Override VM name (default: claude-workspace-smoke)

set -euo pipefail

# ---------- defaults ----------
VM_NAME="claude-workspace-smoke"
KEEP=false
REUSE=false
SKIP_CLAUDE_CLI=false
BINARY_OUT="/tmp/claude-workspace-linux-amd64"
PASS_COUNT=0
FAIL_COUNT=0

# ---------- colors ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# ---------- parse flags ----------
while [[ $# -gt 0 ]]; do
    case "$1" in
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

vm_exec() {
    multipass exec "$VM_NAME" -- bash -c "$1"
}

vm_exec_quiet() {
    multipass exec "$VM_NAME" -- bash -c "$1" >/dev/null 2>&1
}

cleanup() {
    if [[ "$KEEP" == true ]]; then
        echo -e "\n${YELLOW}--keep set. VM '${VM_NAME}' preserved.${NC}"
        echo "  Inspect:  multipass shell ${VM_NAME}"
        echo "  Delete:   multipass delete --purge ${VM_NAME}"
    else
        echo -e "\nCleaning up VM '${VM_NAME}'..."
        multipass delete --purge "$VM_NAME" 2>/dev/null || true
    fi
}

# ========== Phase 1: Preflight ==========
echo -e "\n${BOLD}=== Phase 1: Preflight ===${NC}"

if ! command -v multipass &>/dev/null; then
    echo -e "${RED}Error: multipass not found. Install with: brew install multipass${NC}"
    exit 1
fi
echo "  multipass: $(multipass version | head -1)"

if ! command -v go &>/dev/null; then
    echo -e "${RED}Error: go not found. Install Go first.${NC}"
    exit 1
fi
echo "  go: $(go version)"

# ========== Phase 2: Cross-compile ==========
echo -e "\n${BOLD}=== Phase 2: Cross-compile ===${NC}"

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
echo "  Project: $PROJECT_DIR"
echo "  Target:  linux/amd64 -> $BINARY_OUT"

GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o "$BINARY_OUT" "$PROJECT_DIR"
echo "  Built: $(ls -lh "$BINARY_OUT" | awk '{print $5}')"

# ========== Phase 3: VM lifecycle ==========
echo -e "\n${BOLD}=== Phase 3: VM lifecycle ===${NC}"

# Check if VM already exists
VM_EXISTS=false
if multipass info "$VM_NAME" &>/dev/null; then
    VM_EXISTS=true
fi

if [[ "$VM_EXISTS" == true && "$REUSE" == false ]]; then
    echo "  Deleting existing VM '$VM_NAME'..."
    multipass delete --purge "$VM_NAME"
    VM_EXISTS=false
fi

if [[ "$VM_EXISTS" == false ]]; then
    echo "  Launching VM '$VM_NAME' (Ubuntu 24.04, 2 CPUs, 2G RAM, 10G disk)..."
    multipass launch 24.04 --name "$VM_NAME" --cpus 2 --memory 2G --disk 10G
else
    echo "  Reusing existing VM '$VM_NAME'."
    # Ensure it's running
    multipass start "$VM_NAME" 2>/dev/null || true
fi

trap cleanup EXIT

echo "  VM ready: $(multipass info "$VM_NAME" --format csv | tail -1 | cut -d, -f3)"

# ========== Phase 4: Provision the VM ==========
echo -e "\n${BOLD}=== Phase 4: Provision ===${NC}"

# Transfer binary
echo "  Transferring binary..."
multipass transfer "$BINARY_OUT" "${VM_NAME}:/home/ubuntu/claude-workspace"

# Install binary to PATH
echo "  Installing binary to /usr/local/bin..."
vm_exec "sudo cp /home/ubuntu/claude-workspace /usr/local/bin/claude-workspace && sudo chmod +x /usr/local/bin/claude-workspace"

# Install prerequisites
echo "  Installing prerequisites (git, curl)..."
vm_exec "sudo apt-get update -qq && sudo apt-get install -y -qq git curl >/dev/null 2>&1"

# Pre-seed ~/.claude.json so setup skips interactive API key flow
echo "  Pre-seeding ~/.claude.json..."
vm_exec 'cat > /home/ubuntu/.claude.json << '\''SEED'\''
{"oauthAccount":{"email":"smoke-test@example.com"}}
SEED'

# Stub claude CLI if requested
if [[ "$SKIP_CLAUDE_CLI" == true ]]; then
    echo "  Creating stub claude CLI..."
    vm_exec 'sudo tee /usr/local/bin/claude > /dev/null << '\''STUB'\''
#!/bin/bash
if [[ "$1" == "--version" ]]; then
    echo "claude 1.0.0-stub"
else
    echo "stub: $*"
fi
STUB
sudo chmod +x /usr/local/bin/claude'
fi

echo "  Provision complete."

# ========== Phase 5: Run setup ==========
echo -e "\n${BOLD}=== Phase 5: claude-workspace setup ===${NC}"

SETUP_OUTPUT=$(vm_exec "claude-workspace setup" 2>&1) || true
echo "$SETUP_OUTPUT" | sed 's/^/  | /'

echo ""
echo "  Assertions:"

# Assert ~/.claude/settings.json exists
assert "~/.claude/settings.json exists" \
    vm_exec_quiet "test -f /home/ubuntu/.claude/settings.json"

# Assert ~/.claude/CLAUDE.md exists
assert "~/.claude/CLAUDE.md exists" \
    vm_exec_quiet "test -f /home/ubuntu/.claude/CLAUDE.md"

# Assert binary is executable in PATH
assert "claude-workspace is executable in PATH" \
    vm_exec_quiet "which claude-workspace && test -x /usr/local/bin/claude-workspace"

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

if [[ "$FAIL_COUNT" -gt 0 ]]; then
    echo -e "\n${RED}SMOKE TEST FAILED${NC}"
    exit 1
else
    echo -e "\n${GREEN}SMOKE TEST PASSED${NC}"
    exit 0
fi
