#!/bin/bash
set -e

# MUSE // One-Line Universal Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/KARTHIKKJ369/Tmusic/main/install.sh | bash

REPO="KARTHIKKJ369/Tmusic"
VERSION="latest"

echo "==> Installing muse..."

# Detect OS & Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Target install directory
if [ -w "/usr/local/bin" ]; then
    BIN_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ]; then
    BIN_DIR="$HOME/.local/bin"
else
    mkdir -p "$HOME/.local/bin"
    BIN_DIR="$HOME/.local/bin"
fi

# If Go is installed, build from source or install
if command -v go >/dev/null 2>&1; then
    echo "==> Go detected. Installing via 'go install'..."
    GOBIN="$BIN_DIR" go install "github.com/$REPO/cmd/muse@$VERSION" || {
        echo "==> Downloading pre-built release binary..."
        DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/muse_${OS}_${ARCH}.tar.gz"
        TMP_DIR="$(mktemp -d)"
        curl -fsSL "$DOWNLOAD_URL" | tar -xz -C "$TMP_DIR"
        mv "$TMP_DIR/muse" "$BIN_DIR/muse"
        rm -rf "$TMP_DIR"
    }
else
    echo "==> Downloading pre-built release binary for ${OS}_${ARCH}..."
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/muse_${OS}_${ARCH}.tar.gz"
    TMP_DIR="$(mktemp -d)"
    if curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/muse.tar.gz" 2>/dev/null; then
        tar -xzf "$TMP_DIR/muse.tar.gz" -C "$TMP_DIR"
        mv "$TMP_DIR/muse" "$BIN_DIR/muse"
        rm -rf "$TMP_DIR"
    else
        echo "==> Release binary not found yet. Please install Go (https://go.dev) and run: go install github.com/$REPO/cmd/muse@latest"
        rm -rf "$TMP_DIR"
        exit 1
    fi
fi

chmod +x "$BIN_DIR/muse"

echo ""
echo "✓ Success! 'muse' has been installed to $BIN_DIR/muse"
echo ""
echo "To get started:"
echo "  1. Set your music folder:  muse dir ~/Music"
echo "  2. Start listening:        muse"
echo ""
