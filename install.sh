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
AGENT_KINDS=""
QUICKSTART_SERVICES="select"
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
      --purge-data       Remove resolved kimbap data/config paths under $HOME after --uninstall
      --with-agents      Run agent setup+sync after install
      --with-profiles    Include operating profile install (use with --with-agents)
      --agent-project-dir <path>
                          Project directory to sync agent skills into (default: current directory)
      --agent-kinds <csv>
                          Setup agent kinds (claude-code,opencode,codex,cursor,openclaw,nanoclaw)
                          default: auto-detect supported agents (generic is sync-only)
      --quickstart-services <value>
                           recommended|all|none|select|<comma-separated service names>
                           (default: select; legacy alias 'starter' is still accepted)
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

validate_agent_kinds_for_setup() {
  local trimmed
  trimmed="$(printf '%s' "$AGENT_KINDS" | xargs)"
  if [[ -z "$trimmed" ]]; then
    return 0
  fi

  local part kind
  local invalid=()
  IFS=',' read -r -a parts <<<"$trimmed"
  for part in "${parts[@]}"; do
    kind="$(printf '%s' "$part" | tr '[:upper:]' '[:lower:]' | xargs)"
    [[ -z "$kind" ]] && continue
    case "$kind" in
      claude-code|opencode|codex|cursor|openclaw|nanoclaw)
        ;;
      generic)
        invalid+=("generic (sync-only; use 'kimbap agents sync --agent generic --dir <project>')")
        ;;
      *)
        invalid+=("$kind")
        ;;
    esac
  done

  if [[ ${#invalid[@]} -gt 0 ]]; then
    error "Unsupported --agent-kinds for installer setup: ${invalid[*]}"
  fi
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

  local cmd=("$bin_path" agents setup --dir "$project_dir" --sync)
  if [[ -n "$(printf '%s' "$AGENT_KINDS" | xargs)" ]]; then
    cmd+=(--agent "$AGENT_KINDS")
  fi
  if [[ "$WITH_PROFILES" -eq 1 ]]; then
    cmd+=(--with-profiles --profile-dir "$project_dir")
  fi

  local agent_scope="auto-detect"
  if [[ -n "$(printf '%s' "$AGENT_KINDS" | xargs)" ]]; then
    agent_scope="$AGENT_KINDS"
  fi
  info "Configuring AI agents (${agent_scope}) in ${project_dir}..."
  if ! "${cmd[@]}"; then
    warn "Agent setup failed. You can retry manually:"
    if [[ -n "$(printf '%s' "$AGENT_KINDS" | xargs)" ]]; then
      warn "  $bin_path agents setup --agent \"$AGENT_KINDS\" --sync --dir \"$project_dir\""
    else
      warn "  $bin_path agents setup --sync --dir \"$project_dir\""
    fi
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
    recommended|starter|all)
      if [[ "$normalized" == "starter" ]]; then
        warn "'starter' is deprecated; using 'recommended'"
        normalized="recommended"
      fi
      cmd+=(--services "$normalized")
      ;;
    select|interactive|checkbox)
      if [ -t 1 ] && [ -e /dev/tty ]; then
        cmd+=(--services select)
      else
        return 1
      fi
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
  if [[ "$normalized" == "none" || "$normalized" == "no" || "$normalized" == "skip" || -z "$normalized" || "$normalized" == "select" || "$normalized" == "interactive" || "$normalized" == "checkbox" ]]; then
    normalized="recommended"
    warn "Agent sync requires installed services; using --quickstart-services recommended for bootstrap."
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
    local purged_any=0

    canonicalize_purge_path() {
      local input="$1"
      if [[ -z "$input" ]]; then
        return 1
      fi
      if [[ "$input" != /* ]]; then
        input="$PWD/$input"
      fi
      local parent
      local base
      parent="$(dirname "$input")"
      base="$(basename "$input")"

      if [[ -d "$input" ]]; then
        (cd "$input" 2>/dev/null && pwd -P)
        return
      fi

      if [[ ! -d "$parent" ]]; then
        return 1
      fi

      local parent_abs
      parent_abs="$(cd "$parent" 2>/dev/null && pwd -P)" || return 1
      printf '%s/%s\n' "$parent_abs" "$base"
    }

    is_safe_purge_target() {
      local target="$1"
      local target_abs
      local home_abs

      target_abs="$(canonicalize_purge_path "$target")" || return 1
      home_abs="$(canonicalize_purge_path "$HOME")" || return 1

      if [[ -z "$target_abs" || -z "$home_abs" ]]; then
        return 1
      fi
      case "$target" in
        "/"|"."|".."|"$HOME"|"$HOME/")
          return 1
          ;;
      esac
      case "$target_abs" in
        "$home_abs"/*)
          return 0
          ;;
      esac
      return 1
    }

    purge_path_if_exists() {
      local target="$1"
      if [[ -z "$target" ]]; then
        return
      fi
      if [[ ! -d "$target" && ! -e "$target" && ! -L "$target" ]]; then
        return
      fi
      if ! is_safe_purge_target "$target"; then
        warn "Skipping unsafe purge target outside HOME: $target"
        return
      fi
      if [[ -d "$target" ]]; then
        rm -rf "$target"
        info "Removed $target"
        purged_any=1
        return
      fi
      if [[ -e "$target" || -L "$target" ]]; then
        rm -f "$target"
        info "Removed $target"
        purged_any=1
      fi
    }

    local resolved_data_dir="${KIMBAP_DATA_DIR:-$HOME/.kimbap}"
    purge_path_if_exists "$resolved_data_dir"

    if [[ -n "${KIMBAP_CONFIG:-}" ]]; then
      purge_path_if_exists "$KIMBAP_CONFIG"
    fi
    if [[ -n "${XDG_CONFIG_HOME:-}" ]]; then
      purge_path_if_exists "$XDG_CONFIG_HOME/kimbap/config.yaml"
    fi
    purge_path_if_exists "$HOME/.kimbap/config.yaml"

    if [[ "$purged_any" -eq 0 ]]; then
      info "No resolved kimbap data/config paths found to purge"
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
  if [[ "$UNINSTALL_ONLY" -ne 1 && "$CHECK_ONLY" -ne 1 ]]; then
    validate_agent_kinds_for_setup
  fi

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
  printf "  Alias:  %s/kb -> %s/kimbap\n" "$INSTALL_PATH" "$INSTALL_PATH"
  printf "  Tip:    if this shell still resolves an old kimbap path, run: hash -r\n\n"

  local QUICKSTART_SELECTED="$QUICKSTART_SERVICES"

  if [ -t 1 ] && [ -e /dev/tty ]; then
    printf "Run quickstart init? [Y/n] " >/dev/tty
    read -r answer </dev/tty || answer="n"
    answer="${answer:-Y}"
    case "$answer" in
      [Yy]*)
        local recommended_preview
        if [[ "$OS" == "darwin" ]]; then
          recommended_preview="apple-notes, apple-calendar, apple-reminders, finder, safari, contacts, wikipedia, open-meteo, open-meteo-geocoding, hacker-news"
        else
          recommended_preview="wikipedia, open-meteo, open-meteo-geocoding, hacker-news, rest-countries, exchange-rate, public-holidays, nominatim"
        fi
        printf "Service presets:\n" >/dev/tty
        printf "  recommended: curated defaults (%s)\n" "$recommended_preview" >/dev/tty
        printf "  all:     every catalog service\n" >/dev/tty
        printf "  none:    skip service installation for now\n" >/dev/tty
        printf "  select:  interactive checkbox-style picker\n" >/dev/tty
        printf "  custom:  enter comma-separated service names\n" >/dev/tty
        printf "Select services to install [recommended/all/none/select/custom] (default: %s): " "$QUICKSTART_SERVICES" >/dev/tty
        read -r service_choice </dev/tty || service_choice=""
        service_choice="${service_choice:-$QUICKSTART_SERVICES}"
        local normalized_choice
        normalized_choice="$(printf '%s' "$service_choice" | tr '[:upper:]' '[:lower:]' | xargs)"
        if [[ "$normalized_choice" == "custom" ]]; then
          printf "Enter comma-separated service names: " >/dev/tty
          read -r custom_services </dev/tty || custom_services=""
          if [[ -n "$(printf '%s' "$custom_services" | xargs)" ]]; then
            service_choice="$custom_services"
          else
            service_choice="none"
          fi
        elif [[ "$normalized_choice" == "select" || "$normalized_choice" == "interactive" || "$normalized_choice" == "checkbox" ]]; then
          service_choice="select"
        elif [[ "$normalized_choice" == "starter" ]]; then
          service_choice="recommended"
        fi
        QUICKSTART_SELECTED="$service_choice"
        if ! run_quickstart_init "${INSTALL_PATH}/kimbap" "$service_choice"; then
          warn "Selected quickstart mode '$service_choice' is not available in this shell. Run manually in an interactive terminal."
        fi
        ;;
      [Nn]*)
        info "Skipped. Run when ready:"
        printf "  %s/kimbap init --mode dev --services select\n" "$INSTALL_PATH"
        ;;
      *)
        info "Unrecognised input. Run when ready:"
        printf "  %s/kimbap init --mode dev --services select\n" "$INSTALL_PATH"
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
    printf "Set up detected agent skills in %s? [Y/n] " "$AGENT_DIR" >/dev/tty
    read -r setup_agents </dev/tty || setup_agents="y"
    case "${setup_agents:-Y}" in
      [Yy]*)
        run_agent_setup "${INSTALL_PATH}/kimbap" "$AGENT_DIR"
        ;;
      *)
        info "Skipped agent setup. Run later with:"
        if [[ -n "$(printf '%s' "$AGENT_KINDS" | xargs)" ]]; then
          printf "  %s/kimbap agents setup --agent \"%s\" --sync --dir \"%s\"\n" "${INSTALL_PATH}" "$AGENT_KINDS" "$AGENT_DIR"
        else
          printf "  %s/kimbap agents setup --sync --dir \"%s\"\n" "${INSTALL_PATH}" "$AGENT_DIR"
        fi
        ;;
    esac
  else
    if [[ -n "$(printf '%s' "$AGENT_KINDS" | xargs)" ]]; then
      printf "Next: %s/kimbap agents setup --agent \"%s\" --sync --dir \"%s\"\n" "${INSTALL_PATH}" "$AGENT_KINDS" "$AGENT_DIR"
    else
      printf "Next: %s/kimbap agents setup --sync --dir \"%s\"\n" "${INSTALL_PATH}" "$AGENT_DIR"
    fi
  fi
}

main "$@"
