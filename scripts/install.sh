#!/usr/bin/env bash
#
# Installs the latest version of veil-cli from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/thatsbass/veil-cli/main/scripts/install.sh | bash
#
# Supported platforms:
#   - macOS (amd64, arm64)
#   - Linux (amd64, arm64)
#
# Installation target:
#   /usr/local/bin/veil
#

set -euo pipefail

readonly REPO="thatsbass/veil-cli"
readonly BINARY_NAME="veil"
readonly INSTALL_DIR="/usr/local/bin"

# -----------------------------------------------------------------------------
# Logging
# -----------------------------------------------------------------------------

log() {
    echo "› $*"
}

fatal() {
    echo "error: $*" >&2
    exit 1
}

# -----------------------------------------------------------------------------
# Platform Detection
# -----------------------------------------------------------------------------

detect_os() {
    local os

    os="$(uname -s | tr '[:upper:]' '[:lower:]')"

    case "$os" in
        linux|darwin)
            echo "$os"
            ;;
        *)
            fatal "unsupported operating system: $os"
            ;;
    esac
}

detect_arch() {
    local arch

    arch="$(uname -m)"

    case "$arch" in
        x86_64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            fatal "unsupported architecture: $arch"
            ;;
    esac
}

# -----------------------------------------------------------------------------
# GitHub Release
# -----------------------------------------------------------------------------

get_latest_tag() {
    local response

    response="$(
        curl -fsSL \
            "https://api.github.com/repos/${REPO}/releases/latest"
    )"

    echo "$response" \
        | grep '"tag_name":' \
        | sed -E 's/.*"([^"]+)".*/\1/'
}

build_asset_name() {
    local version="$1"
    local os="$2"
    local arch="$3"

    echo "veil_${version}_${os}_${arch}.tar.gz"
}

build_download_url() {
    local tag="$1"
    local asset="$2"

    echo "https://github.com/${REPO}/releases/download/${tag}/${asset}"
}

# -----------------------------------------------------------------------------
# Installation
# -----------------------------------------------------------------------------

install_binary() {
    local source="$1"
    local target="${INSTALL_DIR}/${BINARY_NAME}"

    if [[ -w "$INSTALL_DIR" ]]; then
        cp "$source" "$target"
    else
        log "sudo required to install into ${INSTALL_DIR}"
        sudo cp "$source" "$target"
    fi

    chmod +x "$target"
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

main() {
    local os
    local arch
    local tag
    local version
    local asset
    local url
    local tmp_dir

    os="$(detect_os)"
    arch="$(detect_arch)"

    log "platform: ${os}/${arch}"

    tag="$(get_latest_tag)"

    if [[ -z "$tag" ]]; then
        fatal "no GitHub release found

Install manually:

  git clone https://github.com/${REPO}.git
  cd veil-cli
  make install"
    fi

    version="${tag#v}"

    asset="$(build_asset_name "$version" "$os" "$arch")"
    url="$(build_download_url "$tag" "$asset")"

    log "downloading veil ${version}"

    tmp_dir="$(mktemp -d)"

    # Cleanup temporary files on exit.
    trap "rm -rf '${tmp_dir}'" EXIT

    curl -fsSL "$url" -o "${tmp_dir}/${asset}"

    tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"

    install_binary "${tmp_dir}/${BINARY_NAME}"

    log "installed: ${INSTALL_DIR}/${BINARY_NAME}"
    log "version: $(${BINARY_NAME} version 2>/dev/null || echo unknown)"

    echo
    log "run 'veil auth login' to get started"
}

main "$@"