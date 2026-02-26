#!/usr/bin/env bash
#
# Shared provisioning library for dev-env.sh and smoke-test.sh.
#
# Expects the caller to have already sourced lib.sh (for MODE, VM_NAME,
# vm_exec, root_exec, copy_binary, and color variables).
#
# Functions:
#   provision_environment <email>

provision_environment() {
    local email="${1:?Usage: provision_environment <email>}"

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
{"oauthAccount":{"email":"'"${email}"'"}}
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

    echo "  Provision complete."
}
