#!/bin/bash
set -e

# MUSE // One-Line Universal Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/KARTHIKKJ369/Tmusic/main/install.sh | bash

REPO="KARTHIKKJ369/Tmusic"
VERSION="latest"

echo "==> Installing muse..."

# Detect OS
UNAME_OS="$(uname -s)"
case "$UNAME_OS" in
    Darwin)
        OS="Darwin"
        ;;
    Linux)
        OS="Linux"
        ;;
    *)
        echo "Error: Unsupported operating system: $UNAME_OS"
        exit 1
        ;;
esac

# Detect Architecture
UNAME_ARCH="$(uname -m)"
case "$UNAME_ARCH" in
    x86_64|amd64)
        ARCH="x86_64"
        ARCH_ALT="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ARCH_ALT="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $UNAME_ARCH"
        exit 1
        ;;
esac

# Determine installation directory
if [ -w "/usr/local/bin" ]; then
    BIN_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ]; then
    BIN_DIR="$HOME/.local/bin"
else
    mkdir -p "$HOME/.local/bin"
    BIN_DIR="$HOME/.local/bin"
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

INSTALLED=0

OS_LOWER="$(echo "$OS" | tr '[:upper:]' '[:lower:]')"
ARCH_LOWER="$(echo "$ARCH_ALT" | tr '[:upper:]' '[:lower:]')"

# Candidate URLs for release tarball
URLS=(
    "https://github.com/$REPO/releases/latest/download/muse_${OS_LOWER}_${ARCH_LOWER}.tar.gz"
    "https://github.com/$REPO/releases/latest/download/muse_${OS}_${ARCH_ALT}.tar.gz"
    "https://github.com/$REPO/releases/latest/download/muse_${OS_LOWER}_${ARCH}.tar.gz"
    "https://github.com/$REPO/releases/latest/download/muse_${OS}_${ARCH}.tar.gz"
    "https://github.com/$REPO/releases/latest/download/Tmusic_${OS_LOWER}_${ARCH_LOWER}.tar.gz"
)

echo "==> Checking for pre-built release binary (${OS} ${ARCH})..."

for URL in "${URLS[@]}"; do
    if curl -fsSL "$URL" -o "$TMP_DIR/muse.tar.gz" 2>/dev/null; then
        echo "==> Downloaded release from: $URL"
        tar -xzf "$TMP_DIR/muse.tar.gz" -C "$TMP_DIR"
        if [ -f "$TMP_DIR/muse" ]; then
            mv "$TMP_DIR/muse" "$BIN_DIR/muse"
            chmod +x "$BIN_DIR/muse"
            INSTALLED=1
            break
        fi
    fi
done

# Fallback: go install if Go is installed
if [ "$INSTALLED" -eq 0 ]; then
    if command -v go >/dev/null 2>&1; then
        echo "==> Pre-built release not found, building from source via 'go install'..."
        GOBIN="$BIN_DIR" go install "github.com/$REPO/cmd/muse@latest"
        INSTALLED=1
    else
        echo "Error: Could not download pre-built release binary."
        echo "Please install Go (https://go.dev) and run: go install github.com/$REPO/cmd/muse@latest"
        exit 1
    fi
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✓ MUSE has been installed successfully to:"
echo "    $BIN_DIR/muse"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Ensure BIN_DIR is in PATH
case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
        echo "Notice: $BIN_DIR is not in your PATH."
        echo "Add it to your shell configuration (e.g. ~/.bashrc or ~/.zshrc):"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo ""
        ;;
esac

echo "Quick Start:"
echo "  1. Set your music folder (only once):"
echo "     muse dir ~/Music"
echo ""
echo "  2. Start listening:"
echo "     muse"
echo ""
