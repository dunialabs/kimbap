#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_PATH="$ROOT_DIR/bin/kimbap"

resolve_config_path() {
  if [ -n "${KIMBAP_CONFIG:-}" ]; then
    printf '%s\n' "$KIMBAP_CONFIG"
    return
  fi

  if [ -n "${KIMBAP_DATA_DIR:-}" ]; then
    printf '%s\n' "${KIMBAP_DATA_DIR%/}/config.yaml"
    return
  fi

  local xdg_path=""
  if [ -n "${XDG_CONFIG_HOME:-}" ]; then
    xdg_path="$XDG_CONFIG_HOME/kimbap/config.yaml"
  fi

  local legacy_path="$HOME/.kimbap/config.yaml"
  if [ -n "$xdg_path" ]; then
    if [ -f "$legacy_path" ]; then
      printf '%s\n' "$legacy_path"
      return
    fi
    printf '%s\n' "$xdg_path"
    return
  fi

  printf '%s\n' "$legacy_path"
}

if ! command -v go >/dev/null 2>&1; then
  printf 'Error: Go is required (>=1.24).\n' >&2
  exit 1
fi

go_version="$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')"
go_major="${go_version%%.*}"
go_minor="${go_version#*.}"
if [ "${go_major:-0}" -lt 1 ] || { [ "${go_major:-0}" -eq 1 ] && [ "${go_minor:-0}" -lt 24 ]; }; then
  printf 'Error: Go >= 1.24 is required (found %s).\n' "$go_version" >&2
  exit 1
fi

printf 'Building kimbap from source...\n'
make -C "$ROOT_DIR" deps
make -C "$ROOT_DIR" build

if [ -w /usr/local/bin ]; then
  make -C "$ROOT_DIR" install
  BIN_PATH="/usr/local/bin/kimbap"
elif command -v sudo >/dev/null 2>&1; then
  if [ -t 0 ]; then
    sudo make -C "$ROOT_DIR" install
    BIN_PATH="/usr/local/bin/kimbap"
  elif sudo -n true >/dev/null 2>&1; then
    sudo -n make -C "$ROOT_DIR" install
    BIN_PATH="/usr/local/bin/kimbap"
  else
    mkdir -p "$HOME/.local/bin"
    install -m 755 "$BIN_PATH" "$HOME/.local/bin/kimbap"
    ln -sf "$HOME/.local/bin/kimbap" "$HOME/.local/bin/kb"
    BIN_PATH="$HOME/.local/bin/kimbap"
    printf 'Installed to %s (add to PATH if needed).\n' "$HOME/.local/bin"
  fi
else
  mkdir -p "$HOME/.local/bin"
  install -m 755 "$BIN_PATH" "$HOME/.local/bin/kimbap"
  ln -sf "$HOME/.local/bin/kimbap" "$HOME/.local/bin/kb"
  BIN_PATH="$HOME/.local/bin/kimbap"
  printf 'Installed to %s (add to PATH if needed).\n' "$HOME/.local/bin"
fi

enable_console="false"
enable_agents="false"

if [ -t 0 ]; then
  read -r -p 'Enable console mode in config now? [y/N]: ' answer_console || true
  case "${answer_console:-}" in
    y|Y|yes|YES) enable_console="true" ;;
  esac

  read -r -p 'Set up agent integration now? [y/N]: ' answer_agents || true
  case "${answer_agents:-}" in
    y|Y|yes|YES) enable_agents="true" ;;
  esac

  init_args=()
  if [ -n "${KIMBAP_CONFIG:-}" ]; then
    init_args+=("--config" "$KIMBAP_CONFIG")
  fi
  apply_console_flag="false"
  if [ "$enable_console" = "true" ]; then
    apply_console_flag="true"
  fi
  if [ "$enable_agents" = "true" ]; then
    init_args+=("--with-agents")
  fi

  config_path="$(resolve_config_path)"
  if [ "$enable_console" = "true" ] && [ -f "$config_path" ]; then
    read -r -p "Existing config found at $config_path. Overwrite config to persist --with-console now? [y/N]: " overwrite_console || true
    case "${overwrite_console:-}" in
      y|Y|yes|YES) init_args+=("--force") ;;
      *)
        apply_console_flag="false"
        printf 'Console setting left unchanged (existing config not overwritten). Run later: %s init --force --with-console\n' "$BIN_PATH"
        ;;
    esac
  fi
  if [ "$apply_console_flag" = "true" ]; then
    init_args+=("--with-console")
  fi

  printf 'Running %s init %s\n' "$BIN_PATH" "${init_args[*]:-}"
  "$BIN_PATH" init "${init_args[@]}"
else
  printf 'Non-interactive shell detected. Run `%s init` manually when ready.\n' "$BIN_PATH"
fi

printf '\nDone.\n'
