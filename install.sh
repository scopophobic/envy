#!/bin/sh
set -e

# Envo CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh
#
# Set ENVO_INSTALL_NO_PATH=1 to skip editing shell config (you add PATH yourself).

REPO="scopophobic/envy"
BINARY="envo"
MARKER="# Added by Envo CLI installer"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { printf "${CYAN}${BOLD}▸${RESET} %s\n" "$1"; }
ok()      { printf "${GREEN}${BOLD}✓${RESET} %s\n" "$1"; }
warn()    { printf "${YELLOW}${BOLD}!${RESET} %s\n" "$1"; }
fail()    { printf "${RED}${BOLD}✗${RESET} %s\n" "$1" >&2; exit 1; }

# Pick a directory we can write to without sudo.
pick_install_dir() {
    USE_USER_BIN=""
    if [ "$OS" = "windows" ]; then
        INSTALL_DIR="$HOME/bin"
        mkdir -p "$INSTALL_DIR"
        return
    fi

    if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
        return
    fi

    if mkdir -p "/usr/local/bin" 2>/dev/null && [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
        return
    fi

    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    USE_USER_BIN=1
}

# Shell line to put INSTALL_DIR on PATH (uses $HOME when possible).
path_export_line() {
    if [ "$INSTALL_DIR" = "${HOME}/.local/bin" ]; then
        printf '%s\n' 'export PATH="$HOME/.local/bin:$PATH"'
    elif [ "$INSTALL_DIR" = "${HOME}/bin" ]; then
        printf '%s\n' 'export PATH="$HOME/bin:$PATH"'
    else
        printf 'export PATH="%s:$PATH"\n' "$INSTALL_DIR"
    fi
}

install_dir_on_path() {
    case ":$PATH:" in *":$INSTALL_DIR:"*) return 0 ;; *) return 1 ;; esac
}

# Pick rc file: zshrc on mac / when present, else bashrc / .profile
detect_rc_file() {
    if [ -f "${HOME}/.zshrc" ] || [ "$OS" = "darwin" ]; then
        printf '%s\n' "${HOME}/.zshrc"
        return
    fi
    if [ -f "${HOME}/.bashrc" ]; then
        printf '%s\n' "${HOME}/.bashrc"
        return
    fi
    if [ -f "${HOME}/.bash_profile" ]; then
        printf '%s\n' "${HOME}/.bash_profile"
        return
    fi
    printf '%s\n' "${HOME}/.profile"
}

# Append PATH once (marked block). No-op if already configured.
auto_configure_path() {
    if install_dir_on_path; then
        return 0
    fi

    if [ -n "${ENVO_INSTALL_NO_PATH}" ]; then
        warn "'${INSTALL_DIR}' is not in your PATH."
        info "Add this to ~/.zshrc (or run: ${CYAN}${INSTALL_DIR}/${BINARY} login${RESET}):"
        printf "    ${CYAN}%s${RESET}\n" "$(path_export_line | head -n 1)"
        return 0
    fi

    RC_FILE=$(detect_rc_file)

    if [ -f "$RC_FILE" ] && grep -qF "$MARKER" "$RC_FILE" 2>/dev/null; then
        ok "PATH already configured for Envo in ${RC_FILE}"
        info "Open a new terminal, or run: ${CYAN}source ${RC_FILE}${RESET}"
        return 0
    fi

    info "Adding ${INSTALL_DIR} to your PATH in ${RC_FILE} …"
    {
        printf '\n%s\n' "$MARKER"
        path_export_line
    } >> "$RC_FILE"

    ok "Updated ${RC_FILE} — open a new terminal, or run: ${CYAN}source ${RC_FILE}${RESET}"
}

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

    VERSION_NUM="${VERSION#v}"
}

download_and_install() {
    TMPDIR=$(mktemp -d)

    if [ "$OS" = "windows" ]; then
        ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.zip"
        EXE_NAME="${BINARY}.exe"
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
    mv "${TMPDIR}/${EXE_NAME}" "${INSTALL_DIR}/${EXE_NAME}"
    chmod +x "${INSTALL_DIR}/${EXE_NAME}"

    rm -rf "$TMPDIR"
}

main() {
    printf "\n"
    printf "${BOLD}  Envo CLI Installer${RESET}\n"
    printf "  Secure secret management for developers\n"
    printf "\n"

    detect_platform
    pick_install_dir
    get_latest_version
    download_and_install

    ok "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}"
    printf "\n"

    if [ "$OS" = "windows" ]; then
        case ":$PATH:" in *":$INSTALL_DIR:"*) ;; *)
            auto_configure_path
            ;;
        esac
    elif [ -n "$USE_USER_BIN" ]; then
        auto_configure_path
    fi

    printf "  Get started:\n"
    printf "    ${CYAN}envo login${RESET}                                    Sign in via browser\n"
    printf "    ${CYAN}envo pull --org my-team --project api --env dev${RESET}   Pull secrets to .env\n"
    printf "    ${CYAN}envo run  --org my-team --project api --env dev -- npm start${RESET}\n"
    printf "                                                   Run with secrets injected\n"
    printf "\n"
}

main
