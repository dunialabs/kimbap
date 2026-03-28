# Kimbap Quick Start

Get from zero to your first `kimbap call` in under 5 minutes.

---

## 1. Install

```bash
curl -fsSL https://raw.githubusercontent.com/dunialabs/kimbap/main/install.sh | bash
```

Or with Homebrew:

```bash
brew install kimbap
```

---

## 2. Initialize

```bash
kimbap init --mode dev --services all
```

Creates `~/.kimbap/` with a config file, encrypted vault, and default policy. Installs all built-in service manifests. Dev mode auto-generates a vault key stored locally, so no extra setup is needed.

---

## 3. Your first call

macOS native services need no credentials. Try this now:

```bash
kimbap call apple-notes.list-notes
```

Returns your Apple Notes list. macOS may prompt for Automation access on first use.

---

## 4. Optional: Connect a SaaS service

```bash
kimbap link github
```

Walks you through OAuth or credential setup for GitHub. For non-interactive setups:

```bash
printf '%s' "$GITHUB_TOKEN" | kimbap link github --stdin
```

Once linked, call it:

```bash
kimbap call github.list-repos
```

The same pattern works for any service: `kimbap link <service>`.

---

## 5. Optional: Connect your AI agent

```bash
kimbap agents setup
```

Auto-detects installed agents (Claude Code, OpenCode, Cursor, Codex) and writes operating rules into the current project directory so the agent discovers kimbap automatically.

---

## Next steps

- [CLI Reference](docs/cli-reference.md) — all commands and flags
- [Service Development Guide](docs/service-development.md) — write your own YAML manifests
- [Installation Guide](docs/installation.md) — production setup, vault, credentials
- [Architecture](docs/architecture.md) — how the pipeline works
