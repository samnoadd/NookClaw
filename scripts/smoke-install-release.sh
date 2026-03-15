#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${1:-${ROOT_DIR}/dist}"
VERSION="${2:-}"

if [ -z "$VERSION" ]; then
  printf 'Usage: %s [dist-dir] <version>\n' "${0##*/}" >&2
  exit 1
fi

if [ ! -d "$DIST_DIR" ]; then
  printf 'Dist directory not found: %s\n' "$DIST_DIR" >&2
  exit 1
fi

DOWNLOAD_BASE="file://${DIST_DIR}"
declare -a CLEANUP_DIRS=()

cleanup() {
  local dir
  for dir in "${CLEANUP_DIRS[@]}"; do
    rm -rf "$dir"
  done
}

trap cleanup EXIT

smoke_install() {
  local os="$1"
  local arch="$2"
  local run_binary="$3"
  local install_dir

  install_dir="$(mktemp -d)"
  CLEANUP_DIRS+=("$install_dir")

  printf 'Smoke testing installer for %s/%s\n' "$os" "$arch"

  NOOKCLAW_VERSION="$VERSION" \
  NOOKCLAW_DOWNLOAD_BASE="$DOWNLOAD_BASE" \
  NOOKCLAW_INSTALL_DIR="$install_dir" \
  NOOKCLAW_OS="$os" \
  NOOKCLAW_ARCH="$arch" \
  "${ROOT_DIR}/install.sh"

  if [ ! -x "${install_dir}/nookclaw" ]; then
    printf 'Installer did not produce an executable binary for %s/%s\n' "$os" "$arch" >&2
    exit 1
  fi

  if [ "$run_binary" = "true" ]; then
    "${install_dir}/nookclaw" --help >/dev/null
  fi
}

smoke_install "Linux" "x86_64" "true"
smoke_install "Linux" "arm64" "false"

printf 'Installer smoke test passed for %s\n' "$VERSION"
