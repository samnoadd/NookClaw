#!/usr/bin/env bash

set -euo pipefail

REPO="${NOOKCLAW_REPO:-samnoadd/NookClaw}"
INSTALL_DIR="${NOOKCLAW_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${NOOKCLAW_VERSION:-}"

usage() {
  cat <<'EOF'
Install NookClaw from GitHub releases.

Usage:
  install.sh [--version vX.Y.Z] [--dir /path/to/bin]

Environment:
  NOOKCLAW_VERSION      Release tag to install. Defaults to the latest release.
  NOOKCLAW_INSTALL_DIR  Destination directory for the nookclaw binary.
  NOOKCLAW_REPO         GitHub repository in owner/name form.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'Unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

fetch_url() {
  local url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO- "$url"
    return
  fi
  printf 'Missing required command: curl or wget\n' >&2
  exit 1
}

download_file() {
  local url="$1"
  local target="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$target"
    return
  fi
  wget -qO "$target" "$url"
}

detect_os() {
  case "$(uname -s)" in
    Linux)
      printf 'Linux'
      ;;
    Darwin)
      printf 'Darwin'
      ;;
    FreeBSD)
      printf 'Freebsd'
      ;;
    NetBSD)
      printf 'Netbsd'
      ;;
    *)
      printf 'Unsupported operating system: %s\n' "$(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'x86_64'
      ;;
    aarch64|arm64)
      printf 'arm64'
      ;;
    armv7l|armv7)
      printf 'armv7'
      ;;
    armv6l|armv6)
      printf 'armv6'
      ;;
    riscv64)
      printf 'riscv64'
      ;;
    loongarch64)
      printf 'loong64'
      ;;
    s390x)
      printf 's390x'
      ;;
    mipsel|mipsle)
      printf 'mipsle'
      ;;
    *)
      printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    printf '%s' "$VERSION"
    return
  fi

  local api json version
  api="https://api.github.com/repos/${REPO}/releases/latest"
  if ! json="$(fetch_url "$api" 2>/dev/null | tr -d '\n')"; then
    printf 'Unable to resolve the latest release from %s\n' "$api" >&2
    printf 'Publish a GitHub release first, or set NOOKCLAW_VERSION to an existing tag.\n' >&2
    exit 1
  fi
  version="$(printf '%s' "$json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p')"
  if [ -z "$version" ]; then
    printf 'Unable to resolve the latest release from %s\n' "$api" >&2
    printf 'Publish a GitHub release first, or set NOOKCLAW_VERSION to an existing tag.\n' >&2
    exit 1
  fi
  printf '%s' "$version"
}

resolve_asset_name() {
  local os="$1"
  local arch="$2"
  printf 'nookclaw_%s_%s.tar.gz' "$os" "$arch"
}

resolve_legacy_asset_name() {
  local os="$1"
  local arch="$2"
  printf 'NookClaw_%s_%s.tar.gz' "$os" "$arch"
}

checksum_file_for() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$path" | awk '{print $1}'
    return
  fi
  printf 'Missing required command: sha256sum or shasum\n' >&2
  exit 1
}

print_path_hint() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*)
      ;;
    *)
      printf '\nAdd %s to your PATH if it is not already available in your shell.\n' "$INSTALL_DIR"
      ;;
  esac
}

require_cmd tar
require_cmd mktemp

OS_NAME="$(detect_os)"
ARCH_NAME="$(detect_arch)"
VERSION="$(resolve_version)"
ASSET_NAME="$(resolve_asset_name "$OS_NAME" "$ARCH_NAME")"
LEGACY_ASSET_NAME="$(resolve_legacy_asset_name "$OS_NAME" "$ARCH_NAME")"
DOWNLOAD_BASE="https://github.com/${REPO}/releases/download/${VERSION}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE_PATH="${TMP_DIR}/${ASSET_NAME}"
CHECKSUMS_PATH="${TMP_DIR}/checksums.txt"

printf 'Installing NookClaw %s for %s/%s\n' "$VERSION" "$OS_NAME" "$ARCH_NAME"

if ! download_file "${DOWNLOAD_BASE}/${ASSET_NAME}" "$ARCHIVE_PATH"; then
  if ! download_file "${DOWNLOAD_BASE}/${LEGACY_ASSET_NAME}" "${TMP_DIR}/${LEGACY_ASSET_NAME}"; then
    printf 'No release archive found for %s/%s in %s\n' "$OS_NAME" "$ARCH_NAME" "$VERSION" >&2
    exit 1
  fi
  ARCHIVE_PATH="${TMP_DIR}/${LEGACY_ASSET_NAME}"
  ASSET_NAME="${LEGACY_ASSET_NAME}"
fi

download_file "${DOWNLOAD_BASE}/checksums.txt" "$CHECKSUMS_PATH"

EXPECTED_SHA="$(awk -v file="$ASSET_NAME" '$2 == file { print $1 }' "$CHECKSUMS_PATH" | head -n 1)"
if [ -z "$EXPECTED_SHA" ]; then
  printf 'No checksum entry found for %s\n' "$ASSET_NAME" >&2
  exit 1
fi

ACTUAL_SHA="$(checksum_file_for "$ARCHIVE_PATH")"
if [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
  printf 'Checksum mismatch for %s\n' "$ASSET_NAME" >&2
  exit 1
fi

tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

if [ ! -f "${TMP_DIR}/nookclaw" ]; then
  printf 'Archive did not contain a nookclaw binary\n' >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "${TMP_DIR}/nookclaw" "${INSTALL_DIR}/nookclaw"

printf 'Installed %s\n' "${INSTALL_DIR}/nookclaw"
print_path_hint
printf 'Run: nookclaw onboard\n'
