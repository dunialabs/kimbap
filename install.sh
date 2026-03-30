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

FORCE_INSTALL=0
CHECK_ONLY=0

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -v|--version)
        [[ $# -ge 2 ]] || error "Missing value for $1"
        VERSION="${2#v}"
        shift 2
        ;;
      -f|--force)
        FORCE_INSTALL=1
        shift
        ;;
      --check)
        CHECK_ONLY=1
        shift
        ;;
      -h|--help)
        cat <<'EOF'
Usage: install.sh [options]

Options:
  -v, --version <x.y.z>  Install a specific version (default: latest release)
  -f, --force            Reinstall even if installed version matches target
      --check            Show installed/latest versions and exit
  -h, --help             Show this help message
EOF
        exit 0
        ;;
      *)
        error "Unknown option: $1"
        ;;
    esac
  done
}

detect_installed_version() {
  local bin="${1:-}"
  if [[ -z "$bin" ]]; then
    return 0
  fi
  local raw
  raw="$($bin --version 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    return 0
  fi
  printf '%s\n' "$raw" | awk 'NR==1 {print $2}' | sed 's/^v//'
}

detect_installed_binary() {
  command -v kimbap 2>/dev/null || true
}

resolve_install_dir() {
  if [[ -n "${INSTALL_DIR:-}" ]]; then
    printf '%s\n' "$INSTALL_DIR"
    return
  fi
  if [[ -w "/usr/local/bin" ]]; then
    printf '/usr/local/bin\n'
    return
  fi
  if [ -t 1 ] && command -v sudo >/dev/null 2>&1; then
    printf '/usr/local/bin\n'
    return
  fi
  printf '%s/.local/bin\n' "$HOME"
}

print_installer_splash() {
  local os="$1"
  local arch="$2"
  local version="$3"
  local current="$4"
  local source_url="$5"

  printf "\n${GREEN}${BOLD}kimbap installer${RESET}\n"
  printf "  target:   %s/%s\n" "$os" "$arch"
  printf "  version:  v%s\n" "$version"
  if [[ -n "$current" ]]; then
    printf "  current:  v%s\n" "$current"
  fi
  printf "  source:   %s\n" "$source_url"
  printf "  update:   manual (rerun install.sh or use brew upgrade)\n\n"
}

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
  expected="$(awk -v f="$filename" '{gsub(/\r$/,"")} $2 == f || $2 == ("*" f) {print $1}' "$checksums_file")"
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
    sudo ln -sf "kimbap" "$install_dir/kb"
    INSTALL_PATH="$install_dir"
    return
  else
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
    warn "'$install_dir' selected. Make sure it is in your PATH:"
    warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi

  install -m 755 "$src_dir/kimbap" "$install_dir/kimbap"
  ln -sf "kimbap" "$install_dir/kb"
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
	parse_args "$@"
  info "Installing kimbap..."

  local OS ARCH
  OS="$(detect_os)"
  ARCH="$(detect_arch)"
  VERSION="$(get_latest_version)"
  local CURRENT_BIN
  CURRENT_BIN="$(detect_installed_binary)"
  local CURRENT_VERSION
  CURRENT_VERSION="$(detect_installed_version "$CURRENT_BIN")"
  local EXPECTED_INSTALL_DIR
  EXPECTED_INSTALL_DIR="$(resolve_install_dir)"
  local EXPECTED_BIN="${EXPECTED_INSTALL_DIR}/kimbap"

  if [[ "$CHECK_ONLY" -eq 1 ]]; then
    printf "latest:    v%s\n" "$VERSION"
    if [[ -n "$CURRENT_VERSION" ]]; then
      printf "installed: v%s\n" "$CURRENT_VERSION"
      if [[ "$CURRENT_VERSION" == "$VERSION" ]]; then
        printf "status:    up-to-date\n"
      else
        printf "status:    update available\n"
      fi
    else
      printf "installed: (none)\n"
      printf "status:    not installed\n"
    fi
    return 0
  fi

  info "Detected platform: ${OS}/${ARCH}"
  info "Version: ${VERSION}"

  TMPDIR_KIMBAP="$(mktemp -d)"
  trap cleanup EXIT

  local base_url="https://github.com/dunialabs/kimbap/releases/download/v${VERSION}"
  local archive_name="kimbap_${VERSION}_${OS}_${ARCH}.tar.gz"
  local archive="${TMPDIR_KIMBAP}/${archive_name}"
  local checksums="${TMPDIR_KIMBAP}/checksums.txt"
  local source_url="${base_url}/${archive_name}"

  print_installer_splash "$OS" "$ARCH" "$VERSION" "$CURRENT_VERSION" "$source_url"

  if [[ "$FORCE_INSTALL" -ne 1 && -n "$CURRENT_VERSION" && "$CURRENT_VERSION" == "$VERSION" ]]; then
    if [[ -n "$CURRENT_BIN" && "$CURRENT_BIN" == "$EXPECTED_BIN" ]]; then
      info "kimbap v${VERSION} is already installed at ${CURRENT_BIN}. Use --force to reinstall."
      return 0
    fi
    warn "kimbap v${VERSION} found at ${CURRENT_BIN:-unknown}, expected managed path ${EXPECTED_BIN}; reinstalling."
  fi

  info "Downloading ${archive_name}..."
  download "$source_url" "$archive"

  info "Downloading checksums..."
  download "${base_url}/checksums.txt" "$checksums"

  verify_checksum "$archive" "$checksums"

  info "Extracting archive..."
  tar -xzf "$archive" -C "$TMPDIR_KIMBAP"

  if [[ ! -f "$TMPDIR_KIMBAP/kimbap" ]]; then
    error "Archive is missing expected binary (kimbap). Release may be malformed."
  fi

  INSTALL_PATH=""
  install_binary "$TMPDIR_KIMBAP"

  printf "\n${GREEN}${BOLD}✓ kimbap v${VERSION} installed successfully!${RESET}\n"
  printf "  Binary: %s/kimbap\n" "$INSTALL_PATH"
  printf "  Alias:  %s/kb -> %s/kimbap\n\n" "$INSTALL_PATH" "$INSTALL_PATH"

  if [ -t 1 ] && [ -e /dev/tty ]; then
    printf "Run quickstart init? [Y/n] " >/dev/tty
    read -r answer </dev/tty || answer="n"
    answer="${answer:-Y}"
    case "$answer" in
      [Yy]*)
        # Quickstart: dev mode with all official services (macOS gets Apple Notes etc, Linux gets data APIs)
        info "Running: kimbap init --mode dev --services all"
        "${INSTALL_PATH}/kimbap" init --mode dev --services all </dev/tty
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
