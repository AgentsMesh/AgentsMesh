#!/bin/sh
# AgentsMesh Runner Installation Script
# Usage: curl -fsSL https://agentsmesh.ai/install.sh | sh
#
# This script installs the AgentsMesh Runner CLI on macOS and Linux.
# For Windows, use: irm https://agentsmesh.ai/install.ps1 | iex

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub release repository
GITHUB_REPO="AgentsMesh/AgentsMesh"
BINARY_NAME="agentsmesh-runner"
INSTALL_DIR=""

# Print colored message
info() {
    printf "${BLUE}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}==>${NC} %s\n" "$1"
}

error() {
    printf "${RED}==>${NC} %s\n" "$1" >&2
    exit 1
}

# Check if stdin is a TTY (interactive terminal)
is_tty() {
    [ -t 0 ]
}

# Detect install directory:
#   1. $INSTALL_DIR env var (user-specified)
#   2. ~/.local/bin (primary, user-space, no sudo required)
detect_install_dir() {
    # Priority 1: user-specified INSTALL_DIR
    if [ -n "$INSTALL_DIR" ]; then
        if [ ! -d "$INSTALL_DIR" ]; then
            mkdir -p "$INSTALL_DIR" 2>/dev/null || {
                error "Cannot create directory: $INSTALL_DIR"
            }
        fi
        if [ -w "$INSTALL_DIR" ]; then
            info "Install directory (user-specified): $INSTALL_DIR"
            return
        fi
        error "Cannot write to specified INSTALL_DIR: $INSTALL_DIR"
    fi

    # Priority 2: ~/.local/bin (primary)
    INSTALL_DIR="$HOME/.local/bin"
    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR"
    fi
    info "Install directory: $INSTALL_DIR"
}

# Check if ~/.local/bin is in PATH and print configuration hints
ensure_path() {
    case ":$PATH:" in
        *":$INSTALL_DIR:"*)
            # Already in PATH, nothing to do
            return
            ;;
    esac

    # Only show hints for non-standard directories
    case "$INSTALL_DIR" in
        /usr/local/bin|/usr/bin|/bin)
            return
            ;;
    esac

    echo ""
    warn "$INSTALL_DIR is not in your PATH."
    echo ""

    # Detect user's shell and provide specific instructions
    CURRENT_SHELL=$(basename "${SHELL:-/bin/sh}")
    case "$CURRENT_SHELL" in
        zsh)
            echo "  Add to your ${BLUE}~/.zshrc${NC}:"
            echo "    ${BLUE}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
            echo ""
            echo "  Then reload:"
            echo "    ${BLUE}source ~/.zshrc${NC}"
            ;;
        bash)
            echo "  Add to your ${BLUE}~/.bashrc${NC} (or ${BLUE}~/.bash_profile${NC} on macOS):"
            echo "    ${BLUE}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
            echo ""
            echo "  Then reload:"
            echo "    ${BLUE}source ~/.bashrc${NC}"
            ;;
        fish)
            echo "  Add to your ${BLUE}~/.config/fish/config.fish${NC}:"
            echo "    ${BLUE}fish_add_path $INSTALL_DIR${NC}"
            echo ""
            echo "  Then reload:"
            echo "    ${BLUE}source ~/.config/fish/config.fish${NC}"
            ;;
        *)
            echo "  Add to your shell profile:"
            echo "    ${BLUE}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
            ;;
    esac
    echo ""
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin)
            OS="darwin"
            # Use universal binary for macOS
            ARCH="all"
            ;;
        linux)
            OS="linux"
            case "$ARCH" in
                x86_64|amd64)
                    ARCH="amd64"
                    ;;
                aarch64|arm64)
                    ARCH="arm64"
                    ;;
                *)
                    error "Unsupported architecture: $ARCH"
                    ;;
            esac
            ;;
        *)
            error "Unsupported operating system: $OS. For Windows, use: irm https://agentsmesh.ai/install.ps1 | iex"
            ;;
    esac

    PLATFORM="${OS}_${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest version from GitHub API
get_latest_version() {
    info "Fetching latest version..."

    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -sL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version. Please check your internet connection."
    fi

    info "Latest version: v$VERSION"
}

# Download and install
install() {
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/agentsmesh-runner_${VERSION}_${PLATFORM}.tar.gz"

    info "Downloading from: $DOWNLOAD_URL"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/runner.tar.gz" || error "Download failed"
    else
        wget -q "$DOWNLOAD_URL" -O "$TMP_DIR/runner.tar.gz" || error "Download failed"
    fi

    # Extract
    info "Extracting..."
    tar -xzf "$TMP_DIR/runner.tar.gz" -C "$TMP_DIR"

    # Find the binary
    if [ -f "$TMP_DIR/$BINARY_NAME" ]; then
        BINARY_PATH="$TMP_DIR/$BINARY_NAME"
    else
        error "Binary not found in archive"
    fi

    # Install binary
    info "Installing to $INSTALL_DIR..."
    mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    success "AgentsMesh Runner v$VERSION installed successfully!"
}

# Verify installation
verify() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        echo ""
        success "Installation verified:"
        "$BINARY_NAME" version
    else
        warn "$BINARY_NAME not found in PATH. You may need to add $INSTALL_DIR to your PATH."
    fi
}

# Print next steps
print_next_steps() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    success "Next steps:"
    echo ""
    echo "  1. Register your runner:"
    echo "     ${BLUE}agentsmesh-runner register --server https://agentsmesh.ai --token <YOUR_TOKEN>${NC}"
    echo ""
    echo "  2. Start the runner:"
    echo "     ${BLUE}agentsmesh-runner run${NC}"
    echo ""
    echo "  Get your registration token from: Settings → Runners → Create Token"
    echo ""
    echo "  For more options, run: ${BLUE}agentsmesh-runner --help${NC}"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Main
main() {
    echo ""
    echo "  █████╗  ██████╗ ███████╗███╗   ██╗████████╗███████╗███╗   ███╗███████╗███████╗██╗  ██╗"
    echo " ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝██╔════╝████╗ ████║██╔════╝██╔════╝██║  ██║"
    echo " ███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║   ███████╗██╔████╔██║█████╗  ███████╗███████║"
    echo " ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║   ╚════██║██║╚██╔╝██║██╔══╝  ╚════██║██╔══██║"
    echo " ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║   ███████║██║ ╚═╝ ██║███████╗███████║██║  ██║"
    echo " ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝╚═╝     ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝"
    echo ""
    echo "                           Runner Installation Script"
    echo ""

    detect_platform
    detect_install_dir
    get_latest_version
    install
    verify
    ensure_path
    print_next_steps
}

main
