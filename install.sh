#!/usr/bin/env bash
#
# tsurf installer for macOS
# Builds from source and installs to /usr/local/bin
#
# Usage:
#   ./install.sh              # build + install
#   ./install.sh --uninstall  # remove tsurf
#

set -euo pipefail

VERSION="0.1.0"
BINARY_NAME="tsurf"
INSTALL_DIR="/usr/local/bin"
BUILD_DIR="$(cd "$(dirname "$0")" && pwd)"
DATA_DIR="$HOME/Library/Application Support/tsurf"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()  { printf "${CYAN}==> ${RESET}%s\n" "$1"; }
ok()    { printf "${GREEN}==> ${RESET}%s\n" "$1"; }
warn()  { printf "${YELLOW}==> ${RESET}%s\n" "$1"; }
fail()  { printf "${RED}==> ${RESET}%s\n" "$1" >&2; exit 1; }

banner() {
    printf "${CYAN}${BOLD}"
    cat << 'EOF'
    __                  ____
   / /______  ______  / __/
  / __/ ___/ / / / ___/ /_
 / /_(__  ) /_/ / /  / __/
 \__/____/\__,_/_/  /_/

EOF
    printf "${RESET}"
    printf "  ${BOLD}Terminal Web Browser for Developers${RESET}\n"
    printf "  Version %s\n\n" "$VERSION"
}

check_deps() {
    info "Checking dependencies..."

    if [[ "$(uname -s)" != "Darwin" ]]; then
        fail "This installer is for macOS only."
    fi

    if ! command -v go &>/dev/null; then
        fail "Go is not installed. Install it from https://go.dev/dl/ or: brew install go"
    fi

    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
    GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)
    if (( GO_MINOR < 21 )); then
        fail "Go 1.21+ is required. Found: $GO_VERSION"
    fi

    ok "Go $(go version | awk '{print $3}') found"
}

build() {
    info "Building tsurf..."

    cd "$BUILD_DIR"

    if [[ ! -f "go.mod" ]]; then
        fail "go.mod not found. Run this script from the tsurf project root."
    fi

    # Fetch dependencies
    go mod download

    # Build with version info and optimizations
    GOARCH=$(go env GOARCH)
    GOOS=$(go env GOOS)

    go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -trimpath \
        -o "${BINARY_NAME}" \
        ./cmd/tsurf/

    if [[ ! -f "${BINARY_NAME}" ]]; then
        fail "Build failed - binary not found."
    fi

    BINARY_SIZE=$(du -h "${BINARY_NAME}" | awk '{print $1}')
    ok "Built ${BINARY_NAME} (${BINARY_SIZE}, ${GOOS}/${GOARCH})"
}

install_binary() {
    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

    # Create install dir if needed
    if [[ ! -d "$INSTALL_DIR" ]]; then
        sudo mkdir -p "$INSTALL_DIR"
    fi

    # Copy binary
    if [[ -w "$INSTALL_DIR" ]]; then
        cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        warn "Requires sudo to install to ${INSTALL_DIR}"
        sudo cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    # Verify it's on PATH
    if command -v "$BINARY_NAME" &>/dev/null; then
        ok "Installed: $(command -v "$BINARY_NAME")"
    else
        warn "${INSTALL_DIR} is not in your PATH."
        warn "Add this to your shell profile:"
        warn "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

setup_data_dirs() {
    info "Setting up data directories..."

    mkdir -p "$DATA_DIR"
    ok "Data directory: ${DATA_DIR}"
}

add_shell_completion() {
    # Basic shell alias suggestions
    local shell_name
    shell_name=$(basename "$SHELL")

    local rc_file=""
    case "$shell_name" in
        zsh)  rc_file="$HOME/.zshrc" ;;
        bash) rc_file="$HOME/.bashrc" ;;
    esac

    if [[ -n "$rc_file" ]]; then
        info "Shell: ${shell_name} (${rc_file})"
        printf "\n  Optional aliases you can add to %s:\n" "$rc_file"
        printf "    ${CYAN}alias ts='tsurf'${RESET}\n"
        printf "    ${CYAN}alias hn='tsurf --theme gruvbox && echo \":hn\"'${RESET}\n\n"
    fi
}

uninstall() {
    banner
    info "Uninstalling tsurf..."

    # Remove binary
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        if [[ -w "$INSTALL_DIR" ]]; then
            rm "${INSTALL_DIR}/${BINARY_NAME}"
        else
            sudo rm "${INSTALL_DIR}/${BINARY_NAME}"
        fi
        ok "Removed ${INSTALL_DIR}/${BINARY_NAME}"
    else
        warn "Binary not found at ${INSTALL_DIR}/${BINARY_NAME}"
    fi

    # Ask about data
    if [[ -d "$DATA_DIR" ]]; then
        printf "\n"
        read -rp "Remove user data (bookmarks, config) at ${DATA_DIR}? [y/N] " answer
        if [[ "$answer" =~ ^[Yy]$ ]]; then
            rm -rf "$DATA_DIR"
            ok "Removed ${DATA_DIR}"
        else
            info "Kept user data at ${DATA_DIR}"
        fi
    fi

    printf "\n"
    ok "tsurf has been uninstalled."
    exit 0
}

verify() {
    info "Verifying installation..."

    INSTALLED_VERSION=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>&1 || true)
    if [[ "$INSTALLED_VERSION" == *"$VERSION"* ]]; then
        ok "Verified: ${INSTALLED_VERSION}"
    else
        warn "Version check returned: ${INSTALLED_VERSION}"
    fi
}

main() {
    banner

    # Handle --uninstall flag
    if [[ "${1:-}" == "--uninstall" ]] || [[ "${1:-}" == "uninstall" ]]; then
        uninstall
    fi

    check_deps
    printf "\n"
    build
    printf "\n"
    install_binary
    printf "\n"
    setup_data_dirs
    printf "\n"
    verify
    printf "\n"
    add_shell_completion

    printf "${GREEN}${BOLD}"
    cat << 'EOF'
  Installation complete!
EOF
    printf "${RESET}\n"

    printf "  Get started:\n"
    printf "    ${BOLD}tsurf${RESET}                        # launch with welcome screen\n"
    printf "    ${BOLD}tsurf https://news.ycombinator.com${RESET}  # open a URL\n"
    printf "    ${BOLD}tsurf --theme catppuccin${RESET}     # try a theme\n"
    printf "\n"
    printf "  Inside tsurf, press ${BOLD}?${RESET} for help or ${BOLD}:${RESET} for commands.\n"
    printf "  Try ${BOLD}:hn${RESET} for Hacker News, ${BOLD}:reddit golang${RESET} for Reddit.\n\n"
}

main "$@"
