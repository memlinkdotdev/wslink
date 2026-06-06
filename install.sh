#!/usr/bin/env bash
# wslink installer (Linux/macOS)
# curl -fsSL https://raw.githubusercontent.com/memlinkdotdev/wslink/main/install.sh | bash

set -e

REPO="memlinkdotdev/wslink"
INSTALL_DIR="${WSLINK_INSTALL_DIR:-$HOME/.local/bin}"
BIN_NAME="wslink"

# Detect OS + arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect latest release
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not detect latest release. Check your internet connection." >&2
  exit 1
fi

ASSET="wslink-${OS}-${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/v${VERSION}/$ASSET"

echo "Installing wslink v${VERSION} to $INSTALL_DIR"

# Create install dir
mkdir -p "$INSTALL_DIR"

# Download + extract
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$URL" -o "$TMP/$ASSET"
tar -xzf "$TMP/$ASSET" -C "$TMP"

# Find the binary inside the extracted dir
EXTRACTED_DIR=$(find "$TMP" -maxdepth 1 -type d -name "wslink-${OS}-${ARCH}" | head -1)
if [ -z "$EXTRACTED_DIR" ]; then
  echo "Extracted archive missing wslink-${OS}-${ARCH}/" >&2
  exit 1
fi

mv "$EXTRACTED_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
chmod +x "$INSTALL_DIR/$BIN_NAME"

# Verify
if [ ! -x "$INSTALL_DIR/$BIN_NAME" ]; then
  echo "Install failed: $INSTALL_DIR/$BIN_NAME not executable." >&2
  exit 1
fi

# PATH hint
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Add to your PATH:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
    echo "Or run:"
    echo "  echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc"
    ;;
esac

echo ""
echo "wslink v${VERSION} installed to $INSTALL_DIR/$BIN_NAME"
echo ""
echo "Try it:"
echo "  wslink forward 4444              # bridge Windows:4444 <-> WSL:4444"
echo "  wslink forward 4444 --windows-host 172.20.0.1"
echo "  wslink --version"
echo ""
