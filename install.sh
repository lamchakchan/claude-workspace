#!/usr/bin/env bash
set -euo pipefail

# claude-workspace installer
# Usage: curl -fsSL https://raw.githubusercontent.com/lamchakchan/claude-workspace/main/install.sh | bash

REPO="lamchakchan/claude-workspace"
BINARY="claude-workspace"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)
    echo "Error: Unsupported operating system: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected: ${OS}/${ARCH}"

# Get latest release tag
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: Could not determine latest release"
  exit 1
fi

VERSION="${LATEST#v}"
echo "Latest version: ${LATEST}"

# Download
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"

echo "Downloading ${URL}..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"

# Extract
echo "Extracting..."
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Install
echo "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
  cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"
else
  echo "Requires sudo for ${INSTALL_DIR}"
  sudo cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  sudo chmod +x "${INSTALL_DIR}/${BINARY}"
fi

# Verify
if "${INSTALL_DIR}/${BINARY}" --version; then
  echo ""
  echo "Successfully installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
  echo ""
  echo "Next steps:"
  echo "  claude-workspace setup                    # First-time setup"
  echo "  claude-workspace attach /path/to/project  # Attach to a project"
else
  echo "Error: Installation verification failed"
  exit 1
fi
