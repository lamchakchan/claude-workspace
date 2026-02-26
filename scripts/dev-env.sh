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
source "$(dirname "$0")/lib-provision.sh"
source "$(dirname "$0")/lib-phases.sh"

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
        echo "Dev environment '${VM_NAME}' already exists, skipping creation."
        ensure_running
        return
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
    provision_environment "dev-test@example.com"

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
