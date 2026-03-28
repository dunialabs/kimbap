#!/usr/bin/env bash
set -euo pipefail

# ── Colors ────────────────────────────────────────────────────────────────────
BOLD='\033[1m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
RESET='\033[0m'

info()  { printf "${CYAN}${BOLD}==>${RESET} %s\n" "$*"; }
warn()  { printf "${YELLOW}${BOLD}WARN:${RESET} %s\n" "$*" >&2; }
error() { printf "${RED}${BOLD}ERROR:${RESET} %s\n" "$*" >&2; exit 1; }

# ── OS / Arch detection ───────────────────────────────────────────────────────
detect_os() {
  local raw
  raw="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$raw" in
    darwin) echo "darwin" ;;
    linux)  echo "linux"  ;;
    *)      error "Unsupported OS: $raw" ;;
  esac
}

detect_arch() {
  local raw
  raw="$(uname -m)"
  case "$raw" in
    x86_64)          echo "amd64" ;;
    aarch64 | arm64) echo "arm64" ;;
    *)               error "Unsupported architecture: $raw" ;;
  esac
}

# ── Version resolution ────────────────────────────────────────────────────────
get_latest_version() {
  if [[ -n "${VERSION:-}" ]]; then
    # Strip leading 'v' if user provided it
    echo "${VERSION#v}"
    return
  fi

  command -v curl >/dev/null 2>&1 || error "'curl' is required but not installed."

  local api_url="https://api.github.com/repos/dunialabs/kimbap/releases/latest"
  local response http_code
  response="$(curl -sSL -w $'\n%{http_code}' "$api_url" 2>/dev/null)" || \
    error "Failed to reach GitHub API. Check your network connection."

  http_code="${response##*$'\n'}"
  response="${response%$'\n'*}"

  if [[ "$http_code" == "404" ]]; then
    error "No releases found yet. Check https://github.com/dunialabs/kimbap/releases"
  elif [[ "$http_code" != "200" ]]; then
    error "GitHub API returned HTTP $http_code. Check your network connection."
  fi

  local tag=""
  if command -v jq >/dev/null 2>&1; then
    tag="$(printf '%s' "$response" | jq -r '.tag_name // empty')"
  else
    tag="$(printf '%s' "$response" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  fi

  if [[ -z "$tag" || "$tag" == "null" ]]; then
    error "No releases found. Check https://github.com/dunialabs/kimbap/releases"
  fi

  # Strip leading 'v'
  echo "${tag#v}"
}

# ── Download helper ───────────────────────────────────────────────────────────
download() {
  local url="$1"
  local dest="$2"
  curl -fsSL -o "$dest" "$url" || error "Download failed: $url"
}

# ── Checksum verification ─────────────────────────────────────────────────────
verify_checksum() {
  local archive="$1"
  local checksums_file="$2"
  local filename
  filename="$(basename "$archive")"

  local expected
  expected="$(awk -v f="$filename" '$2 == f || $2 == ("*" f) {print $1}' "$checksums_file")" || true
  if [[ -z "$expected" ]]; then
    error "No checksum entry found for '$filename' in checksums file."
  fi

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
  else
    error "Neither 'sha256sum' nor 'shasum' found. Cannot verify checksum."
  fi

  if [[ "$actual" != "$expected" ]]; then
    error "Checksum mismatch for '$filename':\n  expected: $expected\n  actual:   $actual"
  fi

  info "Checksum verified"
}

# ── Binary installation ───────────────────────────────────────────────────────
install_binary() {
  local src_dir="$1"
  local install_dir=""

  if [[ -n "${INSTALL_DIR:-}" ]]; then
    install_dir="$INSTALL_DIR"
    mkdir -p "$install_dir"
  elif [[ -w "/usr/local/bin" ]]; then
    install_dir="/usr/local/bin"
  elif [ -t 1 ] && command -v sudo >/dev/null 2>&1; then
    install_dir="/usr/local/bin"
    info "Installing to /usr/local/bin (requires sudo)..."
    sudo mkdir -p "$install_dir"
    sudo install -m 755 "$src_dir/kimbap" "$install_dir/kimbap"
    sudo install -m 755 "$src_dir/kb"     "$install_dir/kb"
    INSTALL_PATH="$install_dir"
    return
  else
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
    warn "'$install_dir' selected. Make sure it is in your PATH:"
    warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi

  install -m 755 "$src_dir/kimbap" "$install_dir/kimbap"
  install -m 755 "$src_dir/kb"     "$install_dir/kb"
  INSTALL_PATH="$install_dir"
}

# ── Cleanup ───────────────────────────────────────────────────────────────────
TMPDIR_KIMBAP=""

cleanup() {
  if [[ -n "$TMPDIR_KIMBAP" && -d "$TMPDIR_KIMBAP" ]]; then
    rm -rf "$TMPDIR_KIMBAP"
  fi
}

# ── Main ──────────────────────────────────────────────────────────────────────
main() {
  info "Installing kimbap..."

  local OS ARCH
  OS="$(detect_os)"
  ARCH="$(detect_arch)"
  VERSION="$(get_latest_version)"

  info "Detected platform: ${OS}/${ARCH}"
  info "Version: ${VERSION}"

  TMPDIR_KIMBAP="$(mktemp -d)"
  trap cleanup EXIT

  local base_url="https://github.com/dunialabs/kimbap/releases/download/v${VERSION}"
  local archive_name="kimbap_${VERSION}_${OS}_${ARCH}.tar.gz"
  local archive="${TMPDIR_KIMBAP}/${archive_name}"
  local checksums="${TMPDIR_KIMBAP}/checksums.txt"

  info "Downloading ${archive_name}..."
  download "${base_url}/${archive_name}" "$archive"

  info "Downloading checksums..."
  download "${base_url}/checksums.txt" "$checksums"

  verify_checksum "$archive" "$checksums"

  info "Extracting archive..."
  tar -xzf "$archive" -C "$TMPDIR_KIMBAP"

  if [[ ! -f "$TMPDIR_KIMBAP/kimbap" ]] || [[ ! -f "$TMPDIR_KIMBAP/kb" ]]; then
    error "Archive is missing expected binaries (kimbap, kb). Release may be malformed."
  fi

  INSTALL_PATH=""
  install_binary "$TMPDIR_KIMBAP"

  printf "\n${GREEN}${BOLD}✓ kimbap v${VERSION} installed successfully!${RESET}\n"
  printf "  Binaries: %s/kimbap  %s/kb\n\n" "$INSTALL_PATH" "$INSTALL_PATH"

  if [ -t 1 ] && [ -e /dev/tty ]; then
    printf "Run quickstart init? [Y/n] " >/dev/tty
    read -r answer </dev/tty || answer="n"
    answer="${answer:-Y}"
    case "$answer" in
      [Yy]*)
        info "Running: kimbap init --mode dev"
        "${INSTALL_PATH}/kimbap" init --mode dev </dev/tty
        ;;
      [Nn]*)
        info "Skipped. Run when ready:"
        printf "  %s/kimbap init --mode dev --services all\n" "$INSTALL_PATH"
        ;;
      *)
        info "Unrecognised input. Run when ready:"
        printf "  %s/kimbap init --mode dev --services all\n" "$INSTALL_PATH"
        ;;
    esac
  else
    printf "Next: %s/kimbap init --mode dev --services all\n" "$INSTALL_PATH"
  fi
}

main "$@"
