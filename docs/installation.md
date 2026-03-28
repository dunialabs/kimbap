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
curl -fsSL https://raw.githubusercontent.com/dunialabs/kimbap/main/install.sh | bash
```

Downloads the latest release binary, verifies SHA256 checksum, and installs `kimbap` (with `kb` created as a local alias symlink). Install path: `/usr/local/bin` if writable, or via `sudo` in interactive shells, falling back to `~/.local/bin` with PATH instructions. May prompt to run quickstart init in interactive shells.

Pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/dunialabs/kimbap/main/install.sh | VERSION=1.1.0 bash
```

### Homebrew (macOS / Linux)

```bash
brew install kimbap
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
kimbap init --mode dev
```

Dev mode auto-generates a vault master key and stores it in `~/.kimbap/.dev-master-key`. In interactive mode, pressing Enter at the service prompt installs all built-in services by default.

### Production

```bash
export KIMBAP_MASTER_KEY_HEX="$(openssl rand -hex 32)"
kimbap init --services all
```

Store the key securely. You need it to unlock the vault on every run. Use `--services all` explicitly in scripts and non-interactive environments.

### What init does

- Creates `~/.kimbap/` as the data directory
- Generates `config.yaml` with default settings
- Initializes the encrypted vault
- Creates a default policy file
- Installs all built-in service manifests (when `--services all`)

**Init flags:**

| Flag | Description |
|---|---|
| `--mode <mode>` | Runtime mode: `dev`, `embedded`, or `connected` |
| `--services <list>` | Comma-separated service names, or `"all"` to install everything |
| `--no-services` | Skip service installation |
| `--with-console` | Enable the embedded console route |
| `--with-agents` | Run agent setup immediately after init |
| `--agents-project-dir <path>` | Project directory to use during agent sync |
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

kimbap ships with profiles for common AI coding agents. Profiles write an operating rules file into the current project directory so the agent discovers kimbap when working in that project.

```bash
# Auto-detect and configure all installed agents
kimbap agents setup

# Also install agent operating profiles into the project directory
kimbap agents setup --with-profiles

# Sync service discovery (generates SKILL.md per service)
kimbap agents sync
```

**Profile install locations:**

| Agent | File path |
|---|---|
| Claude Code | `.claude/KIMBAP_OPERATING_RULES.md` |
| OpenCode | `.opencode/KIMBAP_OPERATING_RULES.md` |
| Cursor | `.cursor/KIMBAP_OPERATING_RULES.md` |
| Codex | `.codex/KIMBAP_OPERATING_RULES.md` |
| Generic | `.agents/KIMBAP_OPERATING_RULES.md` |

Run `kimbap agents sync` any time you install new services to regenerate the discovery files.

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

---

## Configuration reference

These environment variables control kimbap's runtime behavior. All can also be set in `~/.kimbap/config.yaml`.

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
