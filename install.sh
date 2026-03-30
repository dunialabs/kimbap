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
UNINSTALL_ONLY=0
PURGE_DATA=0
WITH_AGENTS=0
WITH_PROFILES=0
AGENT_PROJECT_DIR=""
AGENT_KINDS="claude-code,opencode,codex"
QUICKSTART_SERVICES="starter"
BREW_TAP="dunialabs/kimbap"
BREW_FORMULA="kimbap"
BREW_FULL_FORMULA="${BREW_TAP}/kimbap"

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
      --uninstall)
        UNINSTALL_ONLY=1
        shift
        ;;
      --purge-data)
        PURGE_DATA=1
        shift
        ;;
      --with-agents)
        WITH_AGENTS=1
        shift
        ;;
      --with-profiles)
        WITH_PROFILES=1
        shift
        ;;
      --agent-project-dir)
        [[ $# -ge 2 ]] || error "Missing value for $1"
        AGENT_PROJECT_DIR="$2"
        shift 2
        ;;
      --agent-kinds)
        [[ $# -ge 2 ]] || error "Missing value for $1"
        AGENT_KINDS="$2"
        shift 2
        ;;
      --quickstart-services)
        [[ $# -ge 2 ]] || error "Missing value for $1"
        QUICKSTART_SERVICES="$2"
        shift 2
        ;;
      -h|--help)
        cat <<'EOF'
Usage: install.sh [options]

Options:
  -v, --version <x.y.z>  Install a specific version (default: latest release)
  -f, --force            Reinstall even if installed version matches target
      --check            Show installed/latest versions and exit
      --uninstall        Remove kimbap binaries (and alias) from common install paths
      --purge-data       Remove ~/.kimbap after --uninstall
      --with-agents      Run agent setup+sync after install
      --with-profiles    Include operating profile install (use with --with-agents)
      --agent-project-dir <path>
                          Project directory to sync agent skills into (default: current directory)
      --agent-kinds <csv>
                          Agents to configure (default: claude-code,opencode,codex)
      --quickstart-services <value>
                          starter|all|none|<comma-separated service names> (default: starter)
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
  printf "  update:   manual (rerun install.sh or use brew upgrade %s)\n\n" "$BREW_FULL_FORMULA"
}

resolve_agent_project_dir() {
  local dir
  if [[ -n "${AGENT_PROJECT_DIR:-}" ]]; then
    dir="$AGENT_PROJECT_DIR"
  else
    dir="$PWD"
  fi
  if [[ "$dir" == "~" ]]; then
    dir="$HOME"
  fi
  printf '%s\n' "$dir"
}

run_agent_setup() {
  local bin_path="$1"
  local project_dir="$2"

  if [[ ! -d "$project_dir" ]]; then
    warn "Agent sync skipped: project directory does not exist: $project_dir"
    return 0
  fi

  local cmd=("$bin_path" agents setup --agent "$AGENT_KINDS" --dir "$project_dir")
  if [[ "$WITH_PROFILES" -eq 1 ]]; then
    cmd+=(--with-profiles --profile-dir "$project_dir")
  fi

  info "Configuring AI agents (${AGENT_KINDS}) in ${project_dir}..."
  if ! "${cmd[@]}"; then
    warn "Agent setup failed. You can retry manually:"
    warn "  $bin_path agents setup --agent \"$AGENT_KINDS\" --dir \"$project_dir\""
    return 0
  fi

  info "Agent setup complete."
}

run_quickstart_init() {
  local bin_path="$1"
  local services_value="$2"

  local cmd=("$bin_path" init --mode dev)
  local normalized
  normalized="$(printf '%s' "$services_value" | tr '[:upper:]' '[:lower:]' | xargs)"

  case "$normalized" in
    none|no|skip)
      cmd+=(--no-services)
      ;;
    starter|all)
      cmd+=(--services "$normalized")
      ;;
    "")
      ;;
    *)
      cmd+=(--services "$services_value")
      ;;
  esac

  info "Running: ${cmd[*]}"
  if [ -t 1 ] && [ -e /dev/tty ]; then
    "${cmd[@]}" </dev/tty
  else
    "${cmd[@]}"
  fi
}

ensure_agent_sync_ready() {
  local bin_path="$1"
  local services_value="$2"

  if "$bin_path" service list --format json >/dev/null 2>&1; then
    return 0
  fi

  local normalized
  normalized="$(printf '%s' "$services_value" | tr '[:upper:]' '[:lower:]' | xargs)"
  if [[ "$normalized" == "none" || "$normalized" == "no" || "$normalized" == "skip" || -z "$normalized" ]]; then
    normalized="starter"
    warn "Agent sync requires installed services; using --quickstart-services starter for bootstrap."
  fi

  info "Preparing kimbap config/services for agent sync..."
  run_quickstart_init "$bin_path" "$normalized"
}

remove_path_if_exists() {
  local path="$1"
  local use_sudo="$2"
  if [[ ! -e "$path" && ! -L "$path" ]]; then
    return 0
  fi
  if [[ "$use_sudo" == "1" ]]; then
    sudo rm -f "$path"
  else
    rm -f "$path"
  fi
}

uninstall_kimbap() {
  local removed=0
  local path

  if command -v brew >/dev/null 2>&1; then
    local brew_formula
    local uninstall_candidates=("$BREW_FULL_FORMULA" "$BREW_FORMULA")

    for brew_formula in "${uninstall_candidates[@]}"; do
      if brew list --versions "$brew_formula" >/dev/null 2>&1; then
        info "Detected Homebrew-managed install (${brew_formula}); using brew uninstall..."
        if brew uninstall "$brew_formula"; then
          removed=$((removed + 1))
          break
        else
          warn "Homebrew uninstall failed. Retry manually: brew uninstall $brew_formula"
        fi
      fi
    done
  fi

  for path in "/usr/local/bin/kimbap" "/usr/local/bin/kb"; do
    if [[ -e "$path" || -L "$path" ]]; then
      if [[ -w "$path" || -w "$(dirname "$path")" ]]; then
        remove_path_if_exists "$path" 0
      elif command -v sudo >/dev/null 2>&1; then
        remove_path_if_exists "$path" 1
      else
        warn "Cannot remove $path (permission denied, sudo unavailable)"
        continue
      fi
      removed=$((removed + 1))
      info "Removed $path"
    fi
  done

  for path in "$HOME/.local/bin/kimbap" "$HOME/.local/bin/kb"; do
    if [[ -e "$path" || -L "$path" ]]; then
      remove_path_if_exists "$path" 0
      removed=$((removed + 1))
      info "Removed $path"
    fi
  done

  if [[ "$PURGE_DATA" -eq 1 ]]; then
    if [[ -d "$HOME/.kimbap" ]]; then
      rm -rf "$HOME/.kimbap"
      info "Removed $HOME/.kimbap"
    else
      info "No ~/.kimbap directory found"
    fi
  fi

  if [[ "$removed" -eq 0 && "$PURGE_DATA" -eq 0 ]]; then
    info "No managed kimbap binaries found to remove."
  fi

  printf "\n${GREEN}${BOLD}✓ uninstall complete${RESET}\n"
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

  if [[ "$PURGE_DATA" -eq 1 && "$UNINSTALL_ONLY" -ne 1 ]]; then
    error "--purge-data requires --uninstall"
  fi

  if [[ "$UNINSTALL_ONLY" -eq 1 ]]; then
    uninstall_kimbap
    return 0
  fi

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

  local QUICKSTART_SELECTED="$QUICKSTART_SERVICES"

  if [ -t 1 ] && [ -e /dev/tty ]; then
    printf "Run quickstart init? [Y/n] " >/dev/tty
    read -r answer </dev/tty || answer="n"
    answer="${answer:-Y}"
    case "$answer" in
      [Yy]*)
        printf "Select services to install [starter/all/none/custom] (default: %s): " "$QUICKSTART_SERVICES" >/dev/tty
        read -r service_choice </dev/tty || service_choice=""
        service_choice="${service_choice:-$QUICKSTART_SERVICES}"
        if [[ "$(printf '%s' "$service_choice" | tr '[:upper:]' '[:lower:]' | xargs)" == "custom" ]]; then
          printf "Enter comma-separated service names: " >/dev/tty
          read -r custom_services </dev/tty || custom_services=""
          if [[ -n "$(printf '%s' "$custom_services" | xargs)" ]]; then
            service_choice="$custom_services"
          else
            service_choice="none"
          fi
        fi
        QUICKSTART_SELECTED="$service_choice"
        run_quickstart_init "${INSTALL_PATH}/kimbap" "$service_choice"
        ;;
      [Nn]*)
        info "Skipped. Run when ready:"
        printf "  %s/kimbap init --mode dev --services starter\n" "$INSTALL_PATH"
        ;;
      *)
        info "Unrecognised input. Run when ready:"
        printf "  %s/kimbap init --mode dev --services starter\n" "$INSTALL_PATH"
        ;;
    esac
  else
    QUICKSTART_SELECTED="$QUICKSTART_SERVICES"
    printf "Next: %s/kimbap init --mode dev --services %s\n" "$INSTALL_PATH" "$QUICKSTART_SERVICES"
  fi

  local AGENT_DIR
  AGENT_DIR="$(resolve_agent_project_dir)"

  if [[ "$WITH_AGENTS" -eq 1 ]]; then
    ensure_agent_sync_ready "${INSTALL_PATH}/kimbap" "$QUICKSTART_SELECTED"
    run_agent_setup "${INSTALL_PATH}/kimbap" "$AGENT_DIR"
  elif [ -t 1 ] && [ -e /dev/tty ]; then
    printf "Set up Claude/OpenCode/Codex skills in %s? [y/N] " "$AGENT_DIR" >/dev/tty
    read -r setup_agents </dev/tty || setup_agents="n"
    case "${setup_agents:-N}" in
      [Yy]*)
        run_agent_setup "${INSTALL_PATH}/kimbap" "$AGENT_DIR"
        ;;
      *)
        info "Skipped agent setup. Run later with:"
        printf "  %s/kimbap agents setup --agent \"%s\" --dir \"%s\"\n" "${INSTALL_PATH}" "$AGENT_KINDS" "$AGENT_DIR"
        ;;
    esac
  else
    printf "Next: %s/kimbap agents setup --agent \"%s\" --dir \"%s\"\n" "${INSTALL_PATH}" "$AGENT_KINDS" "$AGENT_DIR"
  fi
}

main "$@"
