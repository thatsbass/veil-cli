#!/usr/bin/env bash
#
# install.sh -- download and install veil-cli from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/thatsbass/veil-cli/main/scripts/install.sh | bash
#
# The script auto-detects the running OS and architecture, fetches the latest
# release asset, and installs the binary to /usr/local/bin.  When the install
# directory is not writable, sudo is invoked automatically.
#
# Environment variables (optional):
#   VEIL_INSTALL_DIR   Override the default install path (/usr/local/bin).
#   VEIL_REPO          Override the GitHub repository (owner/name).
#
set -eu

readonly DEFAULT_INSTALL_DIR="/usr/local/bin"
readonly DEFAULT_REPO="thatsbass/veil-cli"
readonly BINARY_NAME="veil"

# ──────────────────────────────────────────────────────────────────────────────
# Logging helpers
# ──────────────────────────────────────────────────────────────────────────────

err() {
  echo "error: $*" >&2
  exit 1
}

info() {
  echo "  $*"
}

# ──────────────────────────────────────────────────────────────────────────────
# Platform detection
# ──────────────────────────────────────────────────────────────────────────────

detect_platform() {
  local os arch

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "${arch}" in
    x86_64)        arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)             err "unsupported architecture: ${arch}" ;;
  esac

  case "${os}" in
    linux|darwin) ;;
    *)            err "unsupported OS: ${os}" ;;
  esac

  echo "${os}/${arch}"
}

# ──────────────────────────────────────────────────────────────────────────────
# GitHub release helpers
# ──────────────────────────────────────────────────────────────────────────────

fetch_latest_tag() {
  local repo="$1"
  local api_url json tag

  api_url="https://api.github.com/repos/${repo}/releases/latest"
  json="$(curl -fsSL "${api_url}" 2>/dev/null)" || true
  tag="$(echo "${json}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')" || true

  if [[ -z "${tag}" ]]; then
    err "no release found. Build from source instead:

  git clone https://github.com/${repo}.git
  cd veil-cli
  make install"
  fi

  echo "${tag}"
}

build_asset_name() {
  local tag="$1"
  local platform="$2"
  local version="${tag#v}"
  echo "veil_${version}_${platform}.tar.gz"
}

download_release() {
  local repo="$1"
  local tag="$2"
  local asset="$3"
  local dest="$4"
  local url

  url="https://github.com/${repo}/releases/download/${tag}/${asset}"
  curl -fsSL "${url}" -o "${dest}" || err "download failed: ${url}"
}

# ──────────────────────────────────────────────────────────────────────────────
# Installation
# ──────────────────────────────────────────────────────────────────────────────

install_binary() {
  local src="$1"
  local install_dir="${VEIL_INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"
  local dest="${install_dir}/${BINARY_NAME}"

  if [[ -w "${install_dir}" ]]; then
    cp "${src}" "${dest}"
  else
    info "sudo required to write to ${install_dir}"
    sudo cp "${src}" "${dest}"
  fi

  chmod +x "${dest}"
  echo "${dest}"
}

# ──────────────────────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────────────────────

main() {
  local repo="${VEIL_REPO:-${DEFAULT_REPO}}"
  local platform tag asset tmpdir src dest

  platform="$(detect_platform)"
  info "detected: ${platform}"

  tag="$(fetch_latest_tag "${repo}")"
  asset="$(build_asset_name "${tag}" "${platform}")"

  info "downloading veil ${tag#v} (${platform})"

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "${tmpdir}"' EXIT

  download_release "${repo}" "${tag}" "${asset}" "${tmpdir}/${asset}"
  tar -xzf "${tmpdir}/${asset}" -C "${tmpdir}"

  src="${tmpdir}/${BINARY_NAME}"
  dest="$(install_binary "${src}")"

  info "installed: ${dest}"
  info "run 'veil auth login' to get started"
}

main "$@"

