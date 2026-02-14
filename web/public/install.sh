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
GITHUB_REPO="AgentsMesh/AgentsMeshRunner"
BINARY_NAME="agentsmesh-runner"
INSTALL_DIR="/usr/local/bin"

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

    # Install
    info "Installing to $INSTALL_DIR..."

    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        warn "Need sudo permission to install to $INSTALL_DIR"
        sudo mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

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
    echo "     ${BLUE}agentsmesh-runner register --server https://api.agentsmesh.ai --token <YOUR_TOKEN>${NC}"
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

# Check for Homebrew on macOS (suggest but don't require)
check_homebrew() {
    if [ "$OS" = "darwin" ] && command -v brew >/dev/null 2>&1; then
        echo ""
        warn "Homebrew detected! You can also install via:"
        echo "     ${BLUE}brew tap agentsmesh/tap https://github.com/AgentsMesh/BrewCask${NC}"
        echo "     ${BLUE}brew install agentsmesh/tap/agentsmesh-runner${NC}"
        echo ""
        printf "Continue with direct installation? [Y/n] "
        read -r response
        case "$response" in
            [nN][oO]|[nN])
                info "Installation cancelled. Use Homebrew to install."
                exit 0
                ;;
        esac
    fi
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
    check_homebrew
    get_latest_version
    install
    verify
    print_next_steps
}

main
