#!/usr/bin/env sh
# One-line install for Envo CLI (macOS and Linux).
# Usage: curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/install.sh | sh
#
# Set ENVO_REPO (e.g. yourorg/Envo) to override the default GitHub repo.

set -e

REPO="${ENVO_REPO:-envo/cli}"
VERSION="${ENVO_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  BASE_URL="https://github.com/${REPO}/releases/latest/download"
else
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
fi

# Detect OS and arch
OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
  Darwin)  OS_NAME="darwin" ;;
  Linux)   OS_NAME="linux" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH_NAME="amd64" ;;
  aarch64|arm64) ARCH_NAME="arm64" ;;
  *)
    echo "Unsupported arch: $ARCH"
    exit 1
    ;;
esac

# Asset name (no .exe on Unix)
ASSET="envo-${OS_NAME}-${ARCH_NAME}"
URL="${BASE_URL}/${ASSET}"

# Install directory: prefer $HOME/.local/bin, fallback to /usr/local/bin (needs sudo)
INSTALL_DIR="${ENVO_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$INSTALL_DIR"
BIN_PATH="$INSTALL_DIR/envo"

echo "Installing envo to $BIN_PATH"
echo "Downloading $URL ..."

if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "$BIN_PATH" "$URL"
elif command -v wget >/dev/null 2>&1; then
  wget -q -O "$BIN_PATH" "$URL"
else
  echo "Need curl or wget to download."
  exit 1
fi

chmod +x "$BIN_PATH"
echo "Installed: $BIN_PATH"

# Ensure install dir is on PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Add Envo to your PATH by running one of the following (then restart your terminal):"
    echo ""
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc   # bash"
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc   # zsh"
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.profile # sh"
    echo ""
    echo "Or run: export PATH=\"\$HOME/.local/bin:\$PATH\""
    ;;
esac

echo ""
echo "Run: envo whoami"
