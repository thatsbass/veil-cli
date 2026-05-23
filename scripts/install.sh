#!/usr/bin/env bash
#
# install.sh -- download and install veil-cli
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/thatsbass/veil-cli/main/scripts/install.sh | bash
#
# The script detects OS and architecture, downloads the latest binary from
# GitHub Releases, and installs it to /usr/local/bin.
#
set -euo pipefail

REPO="thatsbass/veil-cli"
BINARY="veil"
INSTALL_DIR="/usr/local/bin"

# ── helpers ──

err() { echo "error: $*" >&2; exit 1; }
info() { echo "  $*"; }

# ── detect platform ──

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) err "unsupported architecture: $ARCH" ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) err "unsupported OS: $OS" ;;
esac

# ── fetch latest release ──

info "detected: ${OS}/${ARCH}"

LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)
TAG=$(echo "$LATEST" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
    err "could not determine latest release tag"
fi

VERSION="${TAG#v}"
ASSET="veil_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

info "downloading veil ${VERSION} (${OS}/${ARCH})"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "$TMPDIR/$ASSET" || err "download failed: $URL"
tar -xzf "$TMPDIR/$ASSET" -C "$TMPDIR"

# ── install ──

if [ -w "$INSTALL_DIR" ]; then
    cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
else
    info "need sudo to install to $INSTALL_DIR"
    sudo cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY"

info "installed: $INSTALL_DIR/$BINARY"
info "version:  $($BINARY version 2>/dev/null || echo "unknown")"
echo ""
info "run 'veil auth login' to get started"
