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

Downloads the latest release binary, verifies SHA256 checksum, and installs `kimbap` + `kb` into `/usr/local/bin` when writable, otherwise `~/.local/bin` (with PATH instructions). Optionally runs quickstart init afterward.

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

Dev mode auto-generates a vault master key and stores it in `~/.kimbap/.dev-master-key`. In interactive mode, pressing Enter at the service prompt installs all 54 services by default.

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
- Installs all 54 built-in service manifests (when `--services all`)

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

Secrets are never accepted as inline CLI arguments. Always pipe them in via `--stdin` or point to a file with `--file`.

```bash
# From environment variable
printf '%s' "$GITHUB_TOKEN" | kimbap vault set github.token --stdin

# From file
kimbap vault set stripe.api_key --file ./key.txt
```

The vault is encrypted with a master key. In dev mode (`--mode dev` or `KIMBAP_DEV=true`), the key is auto-generated and stored locally. In production, set `KIMBAP_MASTER_KEY_HEX` explicitly.

---

## Link services to credentials

After storing credentials, bind each service to its vault key or OAuth connector:

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

kimbap ships with profiles for common AI coding agents. Profiles write an operating rules file into the agent's config directory so the agent knows kimbap is available and how to call it.

```bash
# Auto-detect and configure all installed agents
kimbap agents setup

# Or install a specific profile
kimbap profile install claude-code
kimbap profile install opencode
kimbap profile install cursor
kimbap profile install codex
kimbap profile install generic

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
