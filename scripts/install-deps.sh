#!/usr/bin/env bash
set -euo pipefail

# ---------- install-deps.sh ----------
# Detects and installs Go and CUE dependencies for building claude-workspace.
#
# Reads go.mod for the minimum Go version, detects OS/arch and available
# package managers, then installs missing tools via the best available method.
#
# Usage:
#   bash scripts/install-deps.sh           # install all deps
#   bash scripts/install-deps.sh --go      # install Go only
#   bash scripts/install-deps.sh --cue     # install CUE only
#   bash scripts/install-deps.sh --check   # verify only, no install

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ---------- colors ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# ---------- flags ----------
INSTALL_GO=false
INSTALL_CUE=false
INSTALL_GOLANGCI_LINT=false
CHECK_ONLY=false

if [[ $# -eq 0 ]]; then
    INSTALL_GO=true
    INSTALL_CUE=true
else
    for arg in "$@"; do
        case "$arg" in
            --go)              INSTALL_GO=true ;;
            --cue)             INSTALL_CUE=true ;;
            --golangci-lint)   INSTALL_GOLANGCI_LINT=true ;;
            --check) CHECK_ONLY=true; INSTALL_GO=true; INSTALL_CUE=true ;;
            *)       echo -e "${RED}Unknown flag: $arg${NC}"; exit 1 ;;
        esac
    done
fi

# ---------- detect OS and architecture ----------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

# ---------- read minimum Go version from go.mod ----------
read_go_version() {
    local gomod="$PROJECT_DIR/go.mod"
    if [[ ! -f "$gomod" ]]; then
        echo "1.24"
        return
    fi
    local ver
    ver=$(grep -E '^go [0-9]+\.[0-9]+' "$gomod" | head -1 | awk '{print $2}')
    if [[ -z "$ver" ]]; then
        echo "1.24"
    else
        echo "$ver"
    fi
}

MIN_GO_VERSION="$(read_go_version)"

# ---------- version comparison ----------
# Returns 0 if $1 >= $2, 1 otherwise
version_gte() {
    local have="$1" need="$2"
    # Split on dots
    local have_major have_minor need_major need_minor
    have_major="${have%%.*}"
    have_minor="${have#*.}"
    have_minor="${have_minor%%.*}"
    need_major="${need%%.*}"
    need_minor="${need#*.}"
    need_minor="${need_minor%%.*}"

    if (( have_major > need_major )); then
        return 0
    elif (( have_major == need_major && have_minor >= need_minor )); then
        return 0
    fi
    return 1
}

# ---------- detect package managers ----------
has_brew()    { command -v brew &>/dev/null; }
has_port()    { command -v port &>/dev/null; }
has_apt()     { command -v apt-get &>/dev/null; }
has_dnf()     { command -v dnf &>/dev/null; }
has_yum()     { command -v yum &>/dev/null; }
has_pacman()  { command -v pacman &>/dev/null; }
has_zypper()  { command -v zypper &>/dev/null; }
has_apk()     { command -v apk &>/dev/null; }

# ---------- sudo helper ----------
run_privileged() {
    if [[ $EUID -eq 0 ]]; then
        "$@"
    elif command -v sudo &>/dev/null; then
        sudo "$@"
    else
        echo -e "${RED}ERROR:${NC} Need root privileges but sudo is not available"
        exit 1
    fi
}

# ---------- Go ----------
check_go() {
    if ! command -v go &>/dev/null; then
        return 1
    fi
    local current
    current="$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/^go//')"
    if [[ -z "$current" ]]; then
        return 1
    fi
    if version_gte "$current" "$MIN_GO_VERSION"; then
        return 0
    fi
    return 1
}

install_go_direct() {
    # Find latest patch release for the minimum version
    local dl_version="go${MIN_GO_VERSION}.0"
    local tarball="${dl_version}.${OS}-${ARCH}.tar.gz"
    local url="https://go.dev/dl/${tarball}"

    echo -e "  Downloading ${BOLD}${dl_version}${NC} from go.dev..."
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' RETURN

    if command -v curl &>/dev/null; then
        curl -fsSL -o "$tmpdir/$tarball" "$url"
    elif command -v wget &>/dev/null; then
        wget -q -O "$tmpdir/$tarball" "$url"
    else
        echo -e "${RED}ERROR:${NC} Neither curl nor wget found"
        return 1
    fi

    run_privileged rm -rf /usr/local/go
    run_privileged tar -C /usr/local -xzf "$tmpdir/$tarball"

    # Ensure symlinks in /usr/local/bin
    run_privileged ln -sf /usr/local/go/bin/go /usr/local/bin/go
    run_privileged ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
}

install_go_pkg_manager() {
    case "$OS" in
        darwin)
            if has_brew; then
                echo -e "  Installing via ${BOLD}Homebrew${NC}..."
                brew install go
                return $?
            elif has_port; then
                echo -e "  Installing via ${BOLD}MacPorts${NC}..."
                run_privileged port install go
                return $?
            fi
            ;;
        linux)
            if has_apt; then
                echo -e "  Installing via ${BOLD}apt${NC}..."
                run_privileged apt-get update -qq
                run_privileged apt-get install -y -qq golang-go
                return $?
            elif has_dnf; then
                echo -e "  Installing via ${BOLD}dnf${NC}..."
                run_privileged dnf install -y golang
                return $?
            elif has_yum; then
                echo -e "  Installing via ${BOLD}yum${NC}..."
                run_privileged yum install -y golang
                return $?
            elif has_pacman; then
                echo -e "  Installing via ${BOLD}pacman${NC}..."
                run_privileged pacman -S --noconfirm go
                return $?
            elif has_zypper; then
                echo -e "  Installing via ${BOLD}zypper${NC}..."
                run_privileged zypper install -y go
                return $?
            elif has_apk; then
                echo -e "  Installing via ${BOLD}apk${NC}..."
                run_privileged apk add --no-cache go
                return $?
            fi
            ;;
    esac
    return 1
}

install_go() {
    echo -e "${BOLD}Go:${NC} installing (need >= ${MIN_GO_VERSION})..."

    # Try package manager first
    if install_go_pkg_manager 2>/dev/null; then
        # Verify version is sufficient
        hash -r 2>/dev/null || true
        if check_go; then
            echo -e "  ${GREEN}Go installed successfully${NC}"
            return 0
        fi
        echo -e "  ${YELLOW}Package manager version too old, falling back to direct download${NC}"
    fi

    # Direct download fallback
    install_go_direct
    hash -r 2>/dev/null || true
    if check_go; then
        echo -e "  ${GREEN}Go installed successfully${NC}"
        return 0
    fi

    echo -e "  ${RED}Failed to install Go${NC}"
    return 1
}

# ---------- CUE ----------
check_cue() {
    command -v cue &>/dev/null
}

install_cue_go() {
    if ! command -v go &>/dev/null; then
        return 1
    fi
    echo -e "  Installing via ${BOLD}go install${NC}..."
    go install cuelang.org/go/cmd/cue@latest

    # Ensure GOPATH/bin is accessible — symlink into /usr/local/bin if needed
    local gobin
    gobin="$(go env GOPATH)/bin"
    if [[ -x "$gobin/cue" ]] && ! command -v cue &>/dev/null; then
        run_privileged ln -sf "$gobin/cue" /usr/local/bin/cue
    fi
}

install_cue_brew() {
    if ! has_brew; then
        return 1
    fi
    echo -e "  Installing via ${BOLD}Homebrew${NC}..."
    brew install cue-lang/tap/cue
}

install_cue_direct() {
    echo -e "  Installing via ${BOLD}direct download${NC}..."
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' RETURN

    # Fetch latest release tag
    local latest_url="https://api.github.com/repos/cue-lang/cue/releases/latest"
    local tag
    if command -v curl &>/dev/null; then
        tag="$(curl -fsSL "$latest_url" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')"
    elif command -v wget &>/dev/null; then
        tag="$(wget -q -O - "$latest_url" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')"
    else
        echo -e "${RED}ERROR:${NC} Neither curl nor wget found"
        return 1
    fi

    if [[ -z "$tag" ]]; then
        echo -e "${RED}ERROR:${NC} Could not determine latest CUE version"
        return 1
    fi

    local cue_os="$OS"
    local cue_arch="$ARCH"
    local tarball="cue_${tag}_${cue_os}_${cue_arch}.tar.gz"
    local url="https://github.com/cue-lang/cue/releases/download/${tag}/${tarball}"

    if command -v curl &>/dev/null; then
        curl -fsSL -o "$tmpdir/$tarball" "$url"
    else
        wget -q -O "$tmpdir/$tarball" "$url"
    fi

    tar -C "$tmpdir" -xzf "$tmpdir/$tarball"
    run_privileged install -m 755 "$tmpdir/cue" /usr/local/bin/cue
}

install_cue() {
    echo -e "${BOLD}CUE:${NC} installing..."

    # Try go install first (primary method)
    if install_cue_go 2>/dev/null; then
        hash -r 2>/dev/null || true
        if check_cue; then
            echo -e "  ${GREEN}CUE installed successfully${NC}"
            return 0
        fi
    fi

    # Try Homebrew (macOS fallback)
    if install_cue_brew 2>/dev/null; then
        hash -r 2>/dev/null || true
        if check_cue; then
            echo -e "  ${GREEN}CUE installed successfully${NC}"
            return 0
        fi
    fi

    # Direct download (last resort)
    install_cue_direct
    hash -r 2>/dev/null || true
    if check_cue; then
        echo -e "  ${GREEN}CUE installed successfully${NC}"
        return 0
    fi

    echo -e "  ${RED}Failed to install CUE${NC}"
    return 1
}

# ---------- golangci-lint ----------
check_golangci_lint() {
    command -v golangci-lint &>/dev/null
}

install_golangci_lint_go() {
    if ! command -v go &>/dev/null; then
        return 1
    fi
    echo -e "  Installing via ${BOLD}go install${NC}..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

    # Ensure GOPATH/bin is accessible — symlink into /usr/local/bin if needed
    local gobin
    gobin="$(go env GOPATH)/bin"
    if [[ -x "$gobin/golangci-lint" ]] && ! command -v golangci-lint &>/dev/null; then
        run_privileged ln -sf "$gobin/golangci-lint" /usr/local/bin/golangci-lint
    fi
}

install_golangci_lint_brew() {
    if ! has_brew; then
        return 1
    fi
    echo -e "  Installing via ${BOLD}Homebrew${NC}..."
    brew install golangci-lint
}

install_golangci_lint() {
    echo -e "${BOLD}golangci-lint:${NC} installing..."

    # Try Homebrew first on macOS (recommended by golangci-lint docs)
    if install_golangci_lint_brew 2>/dev/null; then
        hash -r 2>/dev/null || true
        if check_golangci_lint; then
            echo -e "  ${GREEN}golangci-lint installed successfully${NC}"
            return 0
        fi
    fi

    # Try go install
    if install_golangci_lint_go 2>/dev/null; then
        hash -r 2>/dev/null || true
        if check_golangci_lint; then
            echo -e "  ${GREEN}golangci-lint installed successfully${NC}"
            return 0
        fi
    fi

    echo -e "  ${RED}Failed to install golangci-lint${NC}"
    return 1
}

# ---------- main ----------
if [[ "$INSTALL_GO" == true ]]; then
    if check_go; then
        current="$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)"
        echo -e "${GREEN}Go:${NC} ${current} (>= ${MIN_GO_VERSION}) ${GREEN}OK${NC}"
    elif [[ "$CHECK_ONLY" == true ]]; then
        echo -e "${RED}Go:${NC} missing or too old (need >= ${MIN_GO_VERSION})"
        exit 1
    else
        install_go
    fi
fi

if [[ "$INSTALL_CUE" == true ]]; then
    if check_cue; then
        current="$(cue version 2>/dev/null | head -1 | awk '{print $2}' || echo "unknown")"
        echo -e "${GREEN}CUE:${NC} ${current} ${GREEN}OK${NC}"
    elif [[ "$CHECK_ONLY" == true ]]; then
        echo -e "${RED}CUE:${NC} missing"
        exit 1
    else
        install_cue
    fi
fi

if [[ "$INSTALL_GOLANGCI_LINT" == true ]]; then
    if check_golangci_lint; then
        current="$(golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")"
        echo -e "${GREEN}golangci-lint:${NC} v${current} ${GREEN}OK${NC}"
    elif [[ "$CHECK_ONLY" == true ]]; then
        echo -e "${RED}golangci-lint:${NC} missing"
        exit 1
    else
        install_golangci_lint
    fi
fi
