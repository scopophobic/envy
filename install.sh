#!/bin/sh
set -e

# Envo CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh

REPO="scopophobic/envy"
BINARY="envo"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()  { printf "${CYAN}${BOLD}▸${RESET} %s\n" "$1"; }
ok()    { printf "${GREEN}${BOLD}✓${RESET} %s\n" "$1"; }
fail()  { printf "${RED}${BOLD}✗${RESET} %s\n" "$1" >&2; exit 1; }

detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)          OS="linux" ;;
        darwin)         OS="darwin" ;;
        mingw*|msys*)   OS="windows" ;;
        *)              fail "Unsupported OS: $OS (try the PowerShell installer on Windows)" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)   ARCH="amd64" ;;
        arm64|aarch64)  ARCH="arm64" ;;
        *)              fail "Unsupported architecture: $ARCH" ;;
    esac
}

get_latest_version() {
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        fail "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi

    # Strip leading 'v' for the archive name
    VERSION_NUM="${VERSION#v}"
}

download_and_install() {
    TMPDIR=$(mktemp -d)

    if [ "$OS" = "windows" ]; then
        ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.zip"
        EXE_NAME="${BINARY}.exe"
        INSTALL_DIR="$HOME/bin"
    else
        ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
        EXE_NAME="${BINARY}"
    fi

    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

    info "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
    curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}" || fail "Download failed. Is ${VERSION} released for ${OS}/${ARCH}?"

    info "Extracting..."
    if [ "$OS" = "windows" ]; then
        unzip -qo "${TMPDIR}/${ARCHIVE}" -d "${TMPDIR}"
    else
        tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"
    fi

    info "Installing to ${INSTALL_DIR}/${EXE_NAME}..."
    mkdir -p "${INSTALL_DIR}"
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMPDIR}/${EXE_NAME}" "${INSTALL_DIR}/${EXE_NAME}"
    else
        sudo mv "${TMPDIR}/${EXE_NAME}" "${INSTALL_DIR}/${EXE_NAME}"
    fi
    chmod +x "${INSTALL_DIR}/${EXE_NAME}"

    rm -rf "$TMPDIR"

    # On Windows (Git Bash), check if ~/bin is in PATH
    if [ "$OS" = "windows" ]; then
        case ":$PATH:" in
            *":$INSTALL_DIR:"*) ;;
            *)
                info "Add this to your ~/.bashrc to make 'envo' available:"
                printf "    ${CYAN}export PATH=\"\$HOME/bin:\$PATH\"${RESET}\n"
                ;;
        esac
    fi
}

main() {
    printf "\n"
    printf "${BOLD}  Envo CLI Installer${RESET}\n"
    printf "  Secure secret management for developers\n"
    printf "\n"

    detect_platform
    get_latest_version
    download_and_install

    ok "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
    printf "\n"
    printf "  Get started:\n"
    printf "    ${CYAN}envo login${RESET}                                    Sign in via browser\n"
    printf "    ${CYAN}envo pull --org my-team --project api --env dev${RESET}   Pull secrets to .env\n"
    printf "    ${CYAN}envo run  --org my-team --project api --env dev -- npm start${RESET}\n"
    printf "                                                   Run with secrets injected\n"
    printf "\n"
}

main
