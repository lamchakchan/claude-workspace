#!/usr/bin/env bash
#
# Persistent dev environment manager for rapid build testing.
#
# Manages a Docker container or Multipass VM that persists across
# edit-compile-test cycles. After initial setup (~2-3 min), the
# deploy subcommand cross-compiles and copies the binary in ~10s.
#
# Usage:
#   bash scripts/dev-env.sh <subcommand> [OPTIONS]
#
# Subcommands:
#   create    Create and provision a new dev environment
#   deploy    Cross-compile and copy binary to existing env (~10s)
#   shell     Open interactive bash shell inside the env
#   test      Deploy + wipe test state + run assertion phases
#   destroy   Remove the dev environment
#   status    Show whether the env exists and its state
#
# Options:
#   --docker       Use Docker (default)
#   --vm           Use Multipass VM
#   --name <n>     Override env name (default: claude-workspace-dev)

set -euo pipefail

# ---------- defaults ----------
MODE="docker"
VM_NAME="claude-workspace-dev"
BINARY_OUT="/tmp/claude-workspace-linux-dev"

# ---------- parse subcommand ----------
if [[ $# -lt 1 ]]; then
    echo "Usage: bash scripts/dev-env.sh <subcommand> [OPTIONS]"
    echo ""
    echo "Subcommands: create, deploy, shell, test, destroy, status"
    echo "Options:     --docker (default), --vm, --name <n>"
    exit 1
fi

SUBCOMMAND="$1"
shift

# ---------- parse flags ----------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --docker)    MODE="docker";    shift ;;
        --vm) MODE="vm"; shift ;;
        --name)      VM_NAME="$2";    shift 2 ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# ---------- shared helpers ----------
source "$(dirname "$0")/lib.sh"

# ---------- dev-env helpers ----------
env_exists() {
    if [[ "$MODE" == "docker" ]]; then
        docker ps -a --format '{{.Names}}' | grep -q "^${VM_NAME}$"
    else
        multipass info "$VM_NAME" &>/dev/null
    fi
}

env_running() {
    if [[ "$MODE" == "docker" ]]; then
        docker ps --format '{{.Names}}' | grep -q "^${VM_NAME}$"
    else
        local state
        state="$(multipass info "$VM_NAME" --format csv 2>/dev/null | tail -1 | cut -d, -f3)"
        [[ "$state" == "Running" ]]
    fi
}

require_env_exists() {
    if ! env_exists; then
        echo -e "${RED}Error: Dev environment '${VM_NAME}' does not exist.${NC}"
        echo "  Create it first: bash scripts/dev-env.sh create --${MODE}"
        exit 1
    fi
}

ensure_running() {
    if ! env_running; then
        echo "  Starting stopped environment '${VM_NAME}'..."
        if [[ "$MODE" == "docker" ]]; then
            docker start "$VM_NAME" >/dev/null
        else
            multipass start "$VM_NAME"
        fi
    fi
}

# ---------- subcommand: create ----------
cmd_create() {
    echo -e "\n${BOLD}=== Creating dev environment ===${NC}"
    echo "  Mode: ${MODE}"
    echo "  Name: ${VM_NAME}"

    # Preflight
    if [[ "$MODE" == "docker" ]]; then
        if ! command -v docker &>/dev/null; then
            echo -e "${RED}Error: docker not found. Install Docker first.${NC}"
            exit 1
        fi
    else
        if ! command -v multipass &>/dev/null; then
            echo -e "${RED}Error: multipass not found. Install with: brew install multipass${NC}"
            exit 1
        fi
    fi

    if ! command -v go &>/dev/null; then
        echo -e "${RED}Error: go not found. Install Go first.${NC}"
        exit 1
    fi

    if env_exists; then
        echo -e "${RED}Error: Environment '${VM_NAME}' already exists.${NC}"
        echo "  Destroy it first: bash scripts/dev-env.sh destroy --${MODE}"
        exit 1
    fi

    # Cross-compile
    echo -e "\n${BOLD}--- Cross-compile ---${NC}"
    cross_compile

    # Create environment
    echo -e "\n${BOLD}--- Create environment ---${NC}"
    if [[ "$MODE" == "docker" ]]; then
        echo "  Starting container '${VM_NAME}' (Ubuntu 24.04)..."
        docker run -d --name "$VM_NAME" ubuntu:24.04 sleep infinity >/dev/null
    else
        echo "  Launching VM '${VM_NAME}' (Ubuntu 24.04, 2 CPUs, 4G RAM, 10G disk)..."
        multipass launch 24.04 --name "$VM_NAME" --cpus 2 --memory 4G --disk 10G
    fi

    # Provision
    echo -e "\n${BOLD}--- Provision ---${NC}"

    if [[ "$MODE" == "docker" ]]; then
        echo "  Creating ubuntu user..."
        docker exec "$VM_NAME" bash -c "useradd -m -s /bin/bash ubuntu" 2>/dev/null || true
    fi

    echo "  Transferring binary..."
    copy_binary

    echo "  Installing prerequisites (git, curl, python3, sudo)..."
    root_exec "apt-get update -qq && apt-get install -y -qq git curl python3 sudo >/dev/null 2>&1"

    if [[ "$MODE" == "docker" ]]; then
        echo "  Configuring passwordless sudo for ubuntu..."
        root_exec "usermod -aG sudo ubuntu && echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers"
    fi

    echo "  Pre-seeding ~/.claude.json..."
    vm_exec 'cat > /home/ubuntu/.claude.json << '\''SEED'\''
{"oauthAccount":{"email":"dev-test@example.com"}}
SEED'

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

    echo -e "\n${GREEN}Dev environment '${VM_NAME}' is ready.${NC}"
    echo "  Deploy:   bash scripts/dev-env.sh deploy --${MODE}"
    echo "  Test:     bash scripts/dev-env.sh test --${MODE}"
    echo "  Shell:    bash scripts/dev-env.sh shell --${MODE}"
    echo "  Destroy:  bash scripts/dev-env.sh destroy --${MODE}"
}

# ---------- subcommand: deploy ----------
cmd_deploy() {
    require_env_exists
    ensure_running

    echo -e "\n${BOLD}=== Deploying to dev environment ===${NC}"

    cross_compile

    echo "  Copying binary..."
    copy_binary

    echo -e "\n${GREEN}Deploy complete.${NC}"
}

# ---------- subcommand: shell ----------
cmd_shell() {
    require_env_exists
    ensure_running

    if [[ "$MODE" == "docker" ]]; then
        exec docker exec -it --user ubuntu -e HOME=/home/ubuntu "$VM_NAME" bash
    else
        exec multipass shell "$VM_NAME"
    fi
}

# ---------- subcommand: test ----------
cmd_test() {
    require_env_exists
    ensure_running

    echo -e "\n${BOLD}=== Deploy + Test ===${NC}"

    # Deploy latest binary
    echo -e "\n${BOLD}--- Deploy ---${NC}"
    cross_compile
    echo "  Copying binary..."
    copy_binary

    # Wipe test state
    echo -e "\n${BOLD}--- Wipe test state ---${NC}"
    echo "  Removing ~/.claude/ and ~/test-project/..."
    vm_exec "rm -rf /home/ubuntu/.claude /home/ubuntu/test-project" || true

    # Re-seed .claude.json (lives outside ~/.claude/)
    vm_exec 'cat > /home/ubuntu/.claude.json << '\''SEED'\''
{"oauthAccount":{"email":"dev-test@example.com"}}
SEED'

    # Run test phases
    run_phase_setup
    run_phase_attach
    run_phase_doctor
    run_phase_upgrade_check

    # Summary
    echo -e "\n${BOLD}=== Summary ===${NC}"
    local total=$((PASS_COUNT + FAIL_COUNT))
    echo -e "  Total: ${total}  ${GREEN}Passed: ${PASS_COUNT}${NC}  ${RED}Failed: ${FAIL_COUNT}${NC}"

    if [[ "$FAIL_COUNT" -gt 0 ]]; then
        echo -e "\n${RED}TESTS FAILED${NC}"
        exit 1
    else
        echo -e "\n${GREEN}ALL TESTS PASSED${NC}"
    fi
}

# ---------- test phases ----------
run_phase_setup() {
    echo -e "\n${BOLD}--- Phase: claude-workspace setup ---${NC}"

    local output
    output=$(vm_exec "claude-workspace setup" 2>&1) || true
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    assert "~/.claude/settings.json exists" \
        vm_exec_quiet "test -f /home/ubuntu/.claude/settings.json"

    assert "~/.claude/CLAUDE.md exists" \
        vm_exec_quiet "test -f /home/ubuntu/.claude/CLAUDE.md"

    assert "claude-workspace is executable in PATH" \
        vm_exec_quiet "test -x /usr/local/bin/claude-workspace"
}

run_phase_attach() {
    echo -e "\n${BOLD}--- Phase: claude-workspace attach ---${NC}"

    # Create a test git repo
    echo "  Creating test project..."
    vm_exec "mkdir -p /home/ubuntu/test-project && cd /home/ubuntu/test-project && git init && git config user.email test@example.com && git config user.name Test && touch README.md && git add . && git commit -m 'init' -q"

    local output
    output=$(vm_exec "claude-workspace attach /home/ubuntu/test-project" 2>&1) || true
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    local project="/home/ubuntu/test-project"

    assert ".claude/settings.json exists" \
        vm_exec_quiet "test -f ${project}/.claude/settings.json"

    assert ".claude/CLAUDE.md exists" \
        vm_exec_quiet "test -f ${project}/.claude/CLAUDE.md"

    assert ".mcp.json exists and is valid JSON" \
        vm_exec_quiet "python3 -c \"import json; json.load(open('${project}/.mcp.json'))\""

    for hook in auto-format.sh block-dangerous-commands.sh enforce-branch-policy.sh validate-secrets.sh; do
        assert "hook ${hook} exists and is executable" \
            vm_exec_quiet "test -x ${project}/.claude/hooks/${hook}"
    done

    assert ".claude/agents/ is non-empty" \
        vm_exec_quiet "test -d ${project}/.claude/agents && [ \"\$(ls -A ${project}/.claude/agents)\" ]"

    assert ".claude/skills/ is non-empty" \
        vm_exec_quiet "test -d ${project}/.claude/skills && [ \"\$(ls -A ${project}/.claude/skills)\" ]"
}

run_phase_doctor() {
    echo -e "\n${BOLD}--- Phase: claude-workspace doctor ---${NC}"

    local project="/home/ubuntu/test-project"
    local output
    output=$(vm_exec "cd ${project} && ANTHROPIC_API_KEY=sk-fake claude-workspace doctor" 2>&1) || true
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    local fail_lines
    fail_lines=$(echo "$output" | grep -c '\[FAIL\]' || true)
    if [[ "$fail_lines" -eq 0 ]]; then
        assert_pass "doctor output contains no [FAIL] lines"
    else
        assert_fail "doctor output contains ${fail_lines} [FAIL] line(s)"
    fi
}

run_phase_upgrade_check() {
    echo -e "\n${BOLD}--- Phase: claude-workspace upgrade --check ---${NC}"

    local exit_code=0
    local output
    output=$(vm_exec "claude-workspace upgrade --check" 2>&1) || exit_code=$?
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    if [[ "$exit_code" -eq 1 ]]; then
        assert_pass "exit code is 1 (update available)"
    else
        if echo "$output" | grep -q "rate limit\|checking for updates"; then
            echo -e "  ${YELLOW}[SKIP]${NC} GitHub API unavailable (rate limited or no network)"
        else
            assert_fail "exit code was ${exit_code}, expected 1 (update available)"
        fi
    fi

    if echo "$output" | grep -q "Current: dev"; then
        assert_pass "output shows 'Current: dev'"
    else
        assert_fail "output missing 'Current: dev'"
    fi

    if echo "$output" | grep -q "Latest:"; then
        assert_pass "output shows 'Latest:' version"
    else
        if echo "$output" | grep -q "rate limit\|checking for updates"; then
            echo -e "  ${YELLOW}[SKIP]${NC} Cannot verify latest version (API unavailable)"
        else
            assert_fail "output missing 'Latest:' version"
        fi
    fi

    if echo "$output" | grep -q "dev build"; then
        assert_pass "output shows dev build warning"
    else
        if echo "$output" | grep -q "rate limit\|checking for updates"; then
            echo -e "  ${YELLOW}[SKIP]${NC} Cannot verify dev warning (API unavailable)"
        else
            assert_fail "output missing dev build warning"
        fi
    fi
}

# ---------- subcommand: destroy ----------
cmd_destroy() {
    if ! env_exists; then
        echo "Dev environment '${VM_NAME}' does not exist. Nothing to destroy."
        exit 0
    fi

    echo -e "Destroying dev environment '${VM_NAME}'..."
    if [[ "$MODE" == "docker" ]]; then
        docker rm -f "$VM_NAME" >/dev/null 2>&1 || true
    else
        multipass delete --purge "$VM_NAME" 2>/dev/null || true
    fi
    echo -e "${GREEN}Destroyed.${NC}"
}

# ---------- subcommand: status ----------
cmd_status() {
    echo -e "${BOLD}Dev environment: ${VM_NAME}${NC}"
    echo "  Mode: ${MODE}"

    if ! env_exists; then
        echo -e "  Status: ${YELLOW}does not exist${NC}"
        return
    fi

    if [[ "$MODE" == "docker" ]]; then
        local state
        state=$(docker inspect --format '{{.State.Status}}' "$VM_NAME" 2>/dev/null)
        echo -e "  Status: ${GREEN}${state}${NC}"
        echo "  Image:  $(docker inspect --format '{{.Config.Image}}' "$VM_NAME" 2>/dev/null)"
    else
        local info
        info=$(multipass info "$VM_NAME" --format csv 2>/dev/null | tail -1)
        local state
        state=$(echo "$info" | cut -d, -f3)
        echo -e "  Status: ${GREEN}${state}${NC}"
        echo "  IPv4:   $(echo "$info" | cut -d, -f4)"
    fi
}

# ---------- dispatch ----------
case "$SUBCOMMAND" in
    create)  cmd_create  ;;
    deploy)  cmd_deploy  ;;
    shell)   cmd_shell   ;;
    test)    cmd_test    ;;
    destroy) cmd_destroy ;;
    status)  cmd_status  ;;
    *)
        echo "Unknown subcommand: $SUBCOMMAND" >&2
        echo "Valid subcommands: create, deploy, shell, test, destroy, status"
        exit 1
        ;;
esac
