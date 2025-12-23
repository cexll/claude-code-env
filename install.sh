#!/bin/sh
set -euo pipefail

# Claude Code Environment Switcher (CCE) Installation Script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cexll/claude-code-env/master/install.sh | bash
#   VERSION=v2.1.0 bash -c "$(curl -fsSL https://raw.githubusercontent.com/cexll/claude-code-env/master/install.sh)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Configuration
REPO_OWNER="cexll"
REPO_NAME="claude-code-env"
BINARY_NAME="cce"
INSTALL_DIR="/usr/local/bin"

# Print colored messages
info() {
    printf "${BLUE}→${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}✓${NC} %s\n" "$1"
}

error() {
    printf "${RED}✗${NC} %s\n" "$1" >&2
}

warn() {
    printf "${YELLOW}⚠${NC} %s\n" "$1"
}

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Linux*)
            echo "linux"
            ;;
        Darwin*)
            echo "darwin"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            error "Windows is not supported by this installer"
            error "Please download the Windows binary manually from:"
            error "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
            exit 1
            ;;
        FreeBSD*)
            error "FreeBSD is not officially supported"
            error "You may try building from source"
            exit 1
            ;;
        *)
            error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
}

# Detect system architecture
detect_arch() {
    local arch
    arch="$(uname -m)"

    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            error "Supported architectures: amd64, arm64"
            exit 1
            ;;
    esac
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check required dependencies
check_dependencies() {
    local missing_deps=0

    # Check for download tool
    if ! command_exists curl && ! command_exists wget; then
        error "Neither curl nor wget found. Please install one of them."
        missing_deps=1
    fi

    # Check for tar
    if ! command_exists tar; then
        error "tar command not found. Please install tar."
        missing_deps=1
    fi

    # Check for checksum tool
    if ! command_exists shasum && ! command_exists sha256sum; then
        warn "Neither shasum nor sha256sum found. Checksum verification will be skipped."
    fi

    if [ "$missing_deps" -eq 1 ]; then
        exit 1
    fi
}

# Download file with progress
download_file() {
    local url="$1"
    local output="$2"

    if command_exists curl; then
        curl -fsSL --progress-bar "$url" -o "$output"
    elif command_exists wget; then
        wget -q --show-progress "$url" -O "$output"
    else
        error "No download tool available"
        exit 1
    fi
}

# Get latest release version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"

    info "Fetching latest version from GitHub..."

    if command_exists curl; then
        local version
        version=$(curl -fsSL "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

        if [ -z "$version" ]; then
            error "Failed to fetch latest version from GitHub API"
            error "You can manually specify a version: VERSION=v2.1.0 bash install.sh"
            exit 1
        fi

        echo "$version"
    elif command_exists wget; then
        local version
        version=$(wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

        if [ -z "$version" ]; then
            error "Failed to fetch latest version from GitHub API"
            error "You can manually specify a version: VERSION=v2.1.0 bash install.sh"
            exit 1
        fi

        echo "$version"
    else
        error "No download tool available to fetch version"
        exit 1
    fi
}

# Verify checksum
verify_checksum() {
    local file="$1"
    local checksum_file="$2"

    if [ ! -f "$checksum_file" ]; then
        warn "Checksum file not found, skipping verification"
        return 0
    fi

    info "Verifying checksum..."

    if command_exists sha256sum; then
        if sha256sum -c "$checksum_file" >/dev/null 2>&1; then
            success "Checksum verification passed"
            return 0
        else
            error "Checksum verification failed!"
            error "The downloaded file may be corrupted or tampered with."
            return 1
        fi
    elif command_exists shasum; then
        if shasum -a 256 -c "$checksum_file" >/dev/null 2>&1; then
            success "Checksum verification passed"
            return 0
        else
            error "Checksum verification failed!"
            error "The downloaded file may be corrupted or tampered with."
            return 1
        fi
    else
        warn "No checksum tool available, skipping verification"
        return 0
    fi
}

# Main installation function
main() {
    echo ""
    info "Claude Code Environment Switcher (CCE) Installer"
    echo ""

    # Check dependencies first
    check_dependencies

    # Detect platform
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)

    info "Detected platform: ${os}/${arch}"

    # Determine version to install
    local version="${VERSION:-}"
    if [ -z "$version" ]; then
        version=$(get_latest_version)
    fi

    info "Installing version: ${version}"

    # Construct download URLs
    local package_name="cce-${version}-${os}-${arch}.tar.gz"
    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${package_name}"
    local checksum_url="${download_url}.sha256"

    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    cd "$tmp_dir"

    # Download package
    info "Downloading ${package_name}..."
    if ! download_file "$download_url" "$package_name"; then
        error "Failed to download package from:"
        error "$download_url"
        error ""
        error "Please check:"
        error "  1. Your internet connection"
        error "  2. The version exists: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
        exit 1
    fi
    success "Downloaded successfully"

    # Download checksum
    info "Downloading checksum..."
    if download_file "$checksum_url" "${package_name}.sha256" 2>/dev/null; then
        # Verify checksum
        if ! verify_checksum "$package_name" "${package_name}.sha256"; then
            error "Installation aborted due to checksum mismatch"
            exit 1
        fi
    else
        warn "Checksum file not available, skipping verification"
    fi

    # Extract package
    info "Extracting package..."
    if ! tar -xzf "$package_name"; then
        error "Failed to extract package"
        exit 1
    fi
    success "Extracted successfully"

    # Find the extracted directory
    local extract_dir="cce-${version}-${os}-${arch}"
    if [ ! -d "$extract_dir" ]; then
        error "Extracted directory not found: $extract_dir"
        exit 1
    fi

    # Check if binary exists
    local binary_path="${extract_dir}/${BINARY_NAME}"
    if [ ! -f "$binary_path" ]; then
        error "Binary not found in package: $binary_path"
        exit 1
    fi

    # Install binary
    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        if command_exists sudo; then
            info "Administrator privileges required"
            if ! sudo cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"; then
                error "Failed to install binary (permission denied)"
                error "Please run with sudo or install manually"
                exit 1
            fi
            sudo chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
        else
            error "Cannot write to ${INSTALL_DIR} and sudo is not available"
            error "Please run this script with appropriate permissions"
            exit 1
        fi
    else
        if ! cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"; then
            error "Failed to install binary"
            exit 1
        fi
        chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    success "Installation complete!"
    echo ""

    # Verify installation
    if command_exists "${BINARY_NAME}"; then
        local installed_version
        installed_version=$("${BINARY_NAME}" --version 2>/dev/null || echo "unknown")
        success "Installed version: ${installed_version}"
    else
        warn "${INSTALL_DIR} may not be in your PATH"
        info "Add it to your PATH or use the full path: ${INSTALL_DIR}/${BINARY_NAME}"
    fi

    echo ""
    info "Quick start:"
    echo "  1. Add your first environment:"
    echo "     ${BINARY_NAME} add"
    echo ""
    echo "  2. Launch Claude Code:"
    echo "     ${BINARY_NAME}"
    echo ""
    echo "  3. Get help:"
    echo "     ${BINARY_NAME} --help"
    echo ""
    info "Documentation: https://github.com/${REPO_OWNER}/${REPO_NAME}#readme"
    echo ""
    info "To uninstall, run:"
    echo "  sudo rm ${INSTALL_DIR}/${BINARY_NAME}"
    echo "  rm -rf ~/.claude-code-env"
    echo ""
}

# Run main function
main
