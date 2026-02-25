#!/usr/bin/env bash
set -euo pipefail

# claude-workspace installer
# Usage: curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-workspace/main/install.sh | bash

REPO="lamchakchan/claude-workspace"
BINARY="claude-workspace"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)
    echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
    exit 1
    ;;
esac

echo -e "${BOLD}Detected:${NC} ${OS}/${ARCH}"

# Get latest release tag
echo -e "${BOLD}Fetching latest release...${NC}"
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo -e "${RED}Error: Could not determine latest release${NC}"
  exit 1
fi

VERSION="${LATEST#v}"
echo -e "${BOLD}Latest version:${NC} ${LATEST}"

# Download
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"

echo -e "${BOLD}Downloading${NC} ${URL}..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"

# Extract
echo -e "${BOLD}Extracting...${NC}"
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Install
echo -e "${BOLD}Installing${NC} to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
  cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"
else
  echo -e "${YELLOW}Requires sudo for ${INSTALL_DIR}${NC}"
  sudo cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  sudo chmod +x "${INSTALL_DIR}/${BINARY}"
fi

# Verify
if "${INSTALL_DIR}/${BINARY}" --version; then
  echo ""
  echo -e "${GREEN}Successfully installed ${BINARY} to ${INSTALL_DIR}/${BINARY}${NC}"
  echo ""
  echo -e "${BOLD}Next steps:${NC}"
  echo -e "  ${YELLOW}claude-workspace setup${NC}                    # First-time setup"
  echo -e "  ${YELLOW}claude-workspace attach /path/to/project${NC}  # Attach to a project"
  echo ""
else
  echo -e "${RED}Error: Installation verification failed${NC}"
  exit 1
fi
