#!/usr/bin/env bash
set -euo pipefail

# claude-workspace installer
# Usage: curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-workspace/main/install.sh | bash

REPO="lamchakchan/claude-workspace"
BINARY="claude-workspace"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERIFY_CHECKSUM=true

# Parse arguments
for arg in "$@"; do
  case "$arg" in
    --no-verify)
      VERIFY_CHECKSUM=false
      ;;
  esac
done

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

# Get latest release tag (uses redirect instead of API to avoid rate limits)
echo -e "${BOLD}Fetching latest release...${NC}"
LATEST=$(curl -fsSI "https://github.com/${REPO}/releases/latest" | grep -i '^location:' | sed -E 's|.*/tag/([^ \r\n]+).*|\1|')

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

# Verify checksum
if [ "$VERIFY_CHECKSUM" = true ]; then
  CHECKSUM_URL="https://github.com/${REPO}/releases/download/${LATEST}/checksums.txt"
  echo -e "${BOLD}Verifying checksum...${NC}"
  if curl -fsSL "$CHECKSUM_URL" -o "${TMPDIR}/checksums.txt" 2>/dev/null; then
    EXPECTED=$(grep -F "${ARCHIVE}" "${TMPDIR}/checksums.txt" | head -1 | awk '{print $1}')
    if [ -z "$EXPECTED" ]; then
      echo -e "${RED}Error: Archive not found in checksums.txt${NC}"
      exit 1
    fi
    # Cross-platform SHA256: shasum on macOS, sha256sum on Linux
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
    else
      echo -e "${YELLOW}Warning: No sha256sum or shasum found. Skipping verification.${NC}"
      ACTUAL="$EXPECTED"
    fi
    if [ "$EXPECTED" != "$ACTUAL" ]; then
      echo -e "${RED}Error: Checksum verification failed!${NC}"
      echo -e "${RED}Expected: ${EXPECTED}${NC}"
      echo -e "${RED}Actual:   ${ACTUAL}${NC}"
      echo -e "${RED}The downloaded file may have been tampered with.${NC}"
      exit 1
    fi
    echo -e "${GREEN}Checksum verified.${NC}"
  else
    echo -e "${YELLOW}Warning: Could not download checksums.txt. Skipping verification.${NC}"
    echo -e "${YELLOW}Use --no-verify to suppress this warning.${NC}"
  fi
else
  echo -e "${YELLOW}Warning: Checksum verification skipped (--no-verify).${NC}"
fi

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
