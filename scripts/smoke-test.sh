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
MODE="vm"
VM_NAME="claude-workspace-smoke"
KEEP=false
REUSE=false
SKIP_CLAUDE_CLI=false
BINARY_OUT="/tmp/claude-workspace-linux"

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

# ---------- shared helpers ----------
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/lib-provision.sh"
source "$(dirname "$0")/lib-phases.sh"

# Override assert_pass/assert_fail to add GitHub Step Summary tracking
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

# Override _phase_skip to add GitHub Step Summary tracking
_phase_skip() {
    local msg="$1"
    echo -e "  ${YELLOW}[SKIP]${NC} $msg"
    summary_append "- :warning: Skipped -- $msg"
}

# Hook: extra assertions for setup phase (symlink + bashrc)
phase_setup_extra() {
    assert "~/.local/bin/claude symlink exists" \
        vm_exec_quiet "test -L /home/ubuntu/.local/bin/claude"

    assert "~/.bashrc contains .local/bin PATH entry" \
        vm_exec_quiet "grep -q '.local/bin' /home/ubuntu/.bashrc"
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

echo "  Project: $PROJECT_DIR"
cross_compile

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
provision_environment "smoke-test@example.com"

# ========== Phase 5: Run setup ==========
echo -e "\n${BOLD}=== Phase 5: claude-workspace setup ===${NC}"
summary_phase 5 "claude-workspace setup"
run_phase_setup

# ========== Phase 6: Run attach ==========
echo -e "\n${BOLD}=== Phase 6: claude-workspace attach ===${NC}"
summary_phase 6 "claude-workspace attach"
run_phase_attach

# ========== Phase 7: Run doctor ==========
echo -e "\n${BOLD}=== Phase 7: claude-workspace doctor ===${NC}"
summary_phase 7 "claude-workspace doctor"
run_phase_doctor

# ========== Phase 8: Run sessions ==========
echo -e "\n${BOLD}=== Phase 8: claude-workspace sessions ===${NC}"
summary_phase 8 "claude-workspace sessions"
run_phase_sessions

# ========== Phase 9: Run upgrade --check ==========
echo -e "\n${BOLD}=== Phase 9: claude-workspace upgrade --check ===${NC}"
summary_phase 9 "claude-workspace upgrade --check"
run_phase_upgrade_check

# ========== Phase 10: Summary ==========
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
