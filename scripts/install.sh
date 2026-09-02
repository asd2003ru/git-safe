#!/usr/bin/env bash
set -euo pipefail

REPO="${GIT_SAFE_REPO:-asd2003ru/git-safe}"
INSTALL_DIR="${GIT_SAFE_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="git-safe"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'git-safe install error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    *) fail "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) printf 'amd64' ;;
    arm64 | aarch64) printf 'arm64' ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

download() {
  url="$1"
  output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$output" "$url"
  else
    fail "curl or wget is required"
  fi
}

main() {
  need_cmd uname
  need_cmd mktemp
  need_cmd tar

  os="$(detect_os)"
  arch="$(detect_arch)"

  archive_name="${BINARY_NAME}-${os}-${arch}.tar.gz"
  download_url="https://github.com/${REPO}/releases/latest/download/${archive_name}"
  tmp_dir="$(mktemp -d)"

  cleanup() {
    rm -rf "$tmp_dir"
  }
  trap cleanup EXIT

  log "Downloading ${download_url}"
  download "$download_url" "${tmp_dir}/${archive_name}"

  tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"
  chmod +x "${tmp_dir}/${BINARY_NAME}-${os}-${arch}"

  mkdir -p "$INSTALL_DIR" 2>/dev/null || {
    fail "could not create ${INSTALL_DIR}; try GIT_SAFE_INSTALL_DIR=\$HOME/.local/bin"
  }

  # If the system directory is unavailable, install will show a clear error.
  install -m 0755 "${tmp_dir}/${BINARY_NAME}-${os}-${arch}" "${INSTALL_DIR}/${BINARY_NAME}"

  log "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
  if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
    log "Make sure ${INSTALL_DIR} is in your PATH."
  fi
}

main "$@"
