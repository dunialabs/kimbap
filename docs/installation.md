# Kimbap Installation Guide

> This guide is designed to be read by both humans and AI agents.
> An agent can fetch this file and follow the steps to install and configure kimbap.

---

## Prerequisites

- **Go 1.24+** — required only for source builds
- **macOS or Linux**
- **Docker** — optional, only needed if you want to run kimbap with Postgres instead of the default SQLite

---

## Install the binary

Choose one method based on your environment.

### Quick install (binary)

```bash
curl -fsSL https://kimbap.sh/install.sh | bash
```

Downloads the latest release binary, verifies SHA256 checksum, and installs `kimbap` (with `kb` created as a local alias symlink). Install path: `/usr/local/bin` if writable, or via `sudo` in interactive shells, falling back to `~/.local/bin` with PATH instructions. May prompt to run quickstart init in interactive shells.

Pin a specific version:

```bash
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --version 0.1.0
```

Check current installed version against latest release:

```bash
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --check
```

Install and immediately configure skills for detected supported agents in the current project (override with `--agent-kinds` if needed):

```bash
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --with-agents
```

If no kimbap config/services exist yet, installer offers interactive service selection first (default quickstart mode: `select`, with recommended services preselected).
Use `--quickstart-services all` when you want the full catalog on first setup.

Control quickstart service selection during install:

```bash
# default is interactive select
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --quickstart-services select

# install recommended curated defaults directly
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --quickstart-services recommended

# install all catalog services
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --quickstart-services all

# skip service install during quickstart init
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --quickstart-services none

```

### Homebrew (macOS / Linux)

```bash
brew install dunialabs/kimbap/kimbap
```

Upgrade later:

```bash
brew update && brew upgrade dunialabs/kimbap/kimbap
```

For script installs, manual update is rerunning the installer:

```bash
curl -fsSL https://kimbap.sh/install.sh | bash
```

### Uninstall

If you installed using `install.sh`, remove script-managed binaries (`kimbap`, `kb`):

```bash
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --uninstall
```

Also remove resolved local kimbap data/config paths (default `~/.kimbap`; respects `KIMBAP_DATA_DIR` / `KIMBAP_CONFIG` when set):

```bash
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --uninstall --purge-data
```

If you installed with Homebrew, uninstall with:

```bash
brew uninstall dunialabs/kimbap/kimbap
```

### From source

Requires Go 1.24+. See [`scripts/install-from-source.sh`](../scripts/install-from-source.sh) for the full script, or:

```bash
git clone https://github.com/dunialabs/kimbap.git
cd kimbap
make deps && make build
# binary at ./bin/kimbap
```

---

## Initialize workspace

### Local / dev evaluation

```bash
kimbap init --services select
```

Dev mode auto-generates a vault master key and stores it in `~/.kimbap/.dev-master-key`. In interactive mode, `--services select` opens a checklist with recommended services preselected (you can switch to `all` from the checklist).
If services are installed during init, eligible shortcut aliases are set up by default. In interactive flows, you'll be asked first; use `--no-shortcuts` to skip.

After init, you can run shortcuts directly (no `kimbap call` prefix):

```bash
geosearch --name "San Francisco"
weather --latitude 37.7749 --longitude -122.4194
```

### Production

```bash
export KIMBAP_MASTER_KEY_HEX="$(openssl rand -hex 32)"
kimbap init --mode embedded --services all
```

Store the key securely. You need it to unlock the vault on every run. Use `--services all` explicitly in scripts and non-interactive environments.

### What init does

- Creates `~/.kimbap/` as the data directory
- Generates `config.yaml` with default settings
- Initializes the encrypted vault
- Creates a default policy file
- Installs selected catalog service manifests (all/recommended/custom via `--services`)

**Init flags:**

| Flag | Description |
|---|---|
| `--mode <mode>` | Runtime mode: `dev`, `embedded`, or `connected` |
| `--services <list>` | Comma-separated service names, or `"all"`, `"recommended"` (legacy alias: `starter`), `"select"` (interactive checklist) |
| `--no-services` | Skip service installation |
| `--no-shortcuts` | Skip automatic shortcut alias setup during service installation |
| `--with-console` | Enable the embedded console route |
| `--with-agents` | Run agent setup immediately after init (syncs into current directory by default) |
| `--agents-project-dir <path>` | Override project directory used during agent sync |
| `--force` | Overwrite existing config if present |

---

## Store credentials

Secrets are never accepted as inline CLI arguments. The easiest way to store and link a credential in one step:

```bash
# From environment variable (stores + links in one step)
printf '%s' "$GITHUB_TOKEN" | kimbap link github --stdin

# From file
kimbap link stripe --file ./key.txt
```

Or use `kimbap vault set` for direct vault access:

```bash
printf '%s' "$GITHUB_TOKEN" | kimbap vault set github.token --stdin
kimbap vault set stripe.api_key --file ./key.txt
```

The vault is encrypted with a master key. In dev mode (`--mode dev` or `KIMBAP_DEV=true`), the key is auto-generated and stored locally. In production, set `KIMBAP_MASTER_KEY_HEX` explicitly.

---

## Link services to credentials

For interactive setup, `kimbap link` guides you through credential or OAuth configuration:

```bash
kimbap link github
kimbap link stripe
```

Linking tells kimbap which credential to inject when that service's actions are called.

---

## OAuth setup

For services backed by OAuth rather than static API keys:

```bash
kimbap auth connect slack
kimbap auth connect notion
kimbap auth connect zoom
```

Bundled OAuth providers: canva, canvas, figma, notion, slack, stripe, zendesk, zoom.

Each command starts the OAuth flow for that provider and stores the resulting tokens in the connector auth store (encrypted separately from the vault).

---

## Configure your AI agent

kimbap ships with profiles for common AI coding agents. Use `--with-profiles` to write project-level operating rules files so the agent discovers kimbap when working in that project.

```bash
# Auto-detect and install global discovery hints for installed agents
kimbap agents setup

# Also sync service discovery into current project
kimbap agents setup --sync --dir "$PWD"

# Also install agent operating profiles into the project directory
kimbap agents setup --sync --with-profiles --dir "$PWD"

# Sync service discovery (generates SKILL.md per service)
kimbap agents sync

# Force project SKILL sync for OpenCode in current project
kimbap agents sync --agent opencode --dir "$PWD" --force

# OpenClaw official workspace sync
kimbap agents sync --agent openclaw --dir "$HOME/.openclaw/workspace"

# NanoClaw repository sync
kimbap agents sync --agent nanoclaw --dir /path/to/nanoclaw
```

**Profile install locations:**

| Agent | File path |
|---|---|
| Claude Code | `.claude/KIMBAP_OPERATING_RULES.md` |
| OpenCode | `.opencode/KIMBAP_OPERATING_RULES.md` |
| Cursor | `.cursor/KIMBAP_OPERATING_RULES.md` |
| OpenClaw | `KIMBAP_OPERATING_RULES.md` (OpenClaw workspace root) |
| NanoClaw | `.claude/KIMBAP_OPERATING_RULES.md` |
| Codex | `.codex/KIMBAP_OPERATING_RULES.md` |
| Generic | `.agents/KIMBAP_OPERATING_RULES.md` |

Service lifecycle commands (`service install`, `enable`, `disable`, `remove`, `update`) attempt automatic agent sync for the current project when agent configs are detected in text output mode. Run `kimbap agents sync` when you want explicit control (`--dir`, `--agent`, `--services`, `--force`) or when using JSON output.

---

## Verify installation

```bash
kimbap doctor
```

`doctor` checks:

- Configuration file validity
- Data directory accessibility
- Vault status and key availability
- Services directory and installed manifests
- Policy file presence

To confirm actions are available and callable:

```bash
kimbap actions list
kimbap call github.list-repos --sort updated
```

To verify shortcut commands:

```bash
kimbap alias set geosearch open-meteo-geocoding.search
geosearch --name "San Francisco"
```

---

## Configuration reference

These environment variables control kimbap's runtime behavior. Config is resolved from `KIMBAP_CONFIG` or default discovery (`$XDG_CONFIG_HOME/kimbap/config.yaml` when set, otherwise `~/.kimbap/config.yaml`).

| Variable | Default | Description |
|---|---|---|
| `KIMBAP_DATA_DIR` | `~/.kimbap` | Data directory |
| `KIMBAP_MASTER_KEY_HEX` | auto (dev) | Vault master key |
| `KIMBAP_DEV` | `false` | Dev mode |
| `KIMBAP_DATABASE_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `KIMBAP_DATABASE_DSN` | `$DATA_DIR/kimbap.db` | DB connection string |
| `KIMBAP_LOG_LEVEL` | `info` | `trace` / `debug` / `info` / `warn` / `error` |

---

## Next steps

- [CLI Reference](./cli-reference.md)
- [Service Development Guide](./service-development.md)
- [Architecture](./architecture.md)
