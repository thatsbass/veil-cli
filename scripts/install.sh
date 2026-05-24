#!/usr/bin/env bash
#
# Copyright 2026 Veil CLI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://apache.org
#
# Unless required by applicable law or agreed to in writing, Version 2.0.
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script downloads and installs the latest version of veil-cli.
# It automatically detects the host operating system and architecture,
# fetches the appropriate binary from GitHub, and places it in the
# installation directory.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/thatsbass/veil-cli/main/scripts/install.sh | bash

set -euo pipefail

# Global constants
readonly REPO="thatsbass/veil-cli"
readonly BINARY="veil"
readonly INSTALL_DIR="/usr/local/bin"

#######################################
# Log an error message to stderr and exit.
# Arguments:
#   Message strings to log.
# Outputs:
#   Writes the error message prefixed with 'error:' to stderr.
#######################################
err() {
  echo "error: $*" >&2
  exit 1
}

#######################################
# Log an information message to stdout.
# Arguments:
#   Message strings to log.
# Outputs:
#   Writes the message prefixed with spaces to stdout.
#######################################
info() {
  echo "  $*"
}

#######################################
# Check if required system tools are installed.
# Arguments:
#   None
# Returns:
#   0 if all tools exist, exits with 1 otherwise.
#######################################
verify_dependencies() {
  local -r required_tools=("curl" "tar" "uname" "mktemp" "sed")
  for tool in "${required_tools[@]}"; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
      err "Required tool '${tool}' is missing. Please install it first."
    fi
  done
}

#######################################
# Detect and normalize system architecture.
# Arguments:
#   None
# Outputs:
#   Writes the normalized architecture name to stdout.
#######################################
detect_arch() {
  local -r arch=$(uname -m)
  case "${arch}" in
    x86_64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) err "Unsupported architecture: ${arch}" ;;
  case
}

#######################################
# Detect and normalize operating system.
# Arguments:
#   None
# Outputs:
#   Writes the normalized OS name to stdout.
#######################################
detect_os() {
  local -r os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "${os}" in
    linux|darwin) echo "${os}" ;;
    *) err "Unsupported OS: ${os}" ;;
  esac
}

#######################################
# Fetch the latest release tag from GitHub API.
# Arguments:
#   None
# Outputs:
#   Writes the release tag (e.g., v1.0.0) to stdout.
#######################################
get_latest_tag() {
  local json
  json=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null) || true
  
  local tag
  tag=$(echo "${json}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
  
  if [[ -z "${tag}" ]]; then
    err "No release found on GitHub. Build from source instead:

  git clone https://github.com/${REPO}.git
  cd veil-cli
  make install"
  fi
  echo "${tag}"
}

#######################################
# Main script execution flow.
#######################################
main() {
  verify_dependencies

  local -r os=$(detect_os)
  local -r arch=$(detect_arch)
  info "Detected: ${os}/${arch}"

  local -r tag=$(get_latest_tag)
  local -r version="${tag#v}"
  local -r asset="veil_${version}_${os}_${arch}.tar.gz"
  local -r url="https://github.com/${REPO}/releases/download/${tag}/${asset}"

  info "Downloading veil ${version} (${os}/${arch})"

  local -r tmpdir=$(mktemp -d)
  trap 'rm -rf "${tmpdir}"' EXIT

  curl -fsSL "${url}" -o "${tmpdir}/${asset}" || err "Download failed: ${url}"
  tar -xzf "${tmpdir}/${asset}" -C "${tmpdir}"

  # Safe installation with privilege elevation if required
  if [[ -w "${INSTALL_DIR}" ]]; then
    cp "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    info "Need sudo privileges to install to ${INSTALL_DIR}"
    sudo cp "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi

  chmod +x "${INSTALL_DIR}/${BINARY}"

  info "Installed: ${INSTALL_DIR}/${BINARY}"
  info "Version:   $("${INSTALL_DIR}/${BINARY}" version 2>/dev/null || echo "unknown")"
  echo ""
  info "Run 'veil auth login' to get started"
}

main "$@"
