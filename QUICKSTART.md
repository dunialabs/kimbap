# Kimbap Quick Start

Get from zero to your first `kimbap call` in under 5 minutes.

---

## 1. Install

```bash
curl -fsSL https://kimbap.sh/install.sh | bash
```

Or with Homebrew:

```bash
brew install dunialabs/kimbap/kimbap
```

---

## 2. Initialize

```bash
kimbap init --mode dev --services all
```

Creates `~/.kimbap/` with a config file, encrypted vault, and default policy. Installs all catalog service manifests. Dev mode auto-generates a vault key stored locally, so no extra setup is needed.

---

## 3. Your first call

Run this:

```bash
kimbap call open-meteo-geocoding.search --name "San Francisco" --count 1
```

Then fetch weather for San Francisco:

```bash
kimbap call open-meteo.get-forecast --latitude 37.7749 --longitude -122.4194
```

Simple flow, no API key, no localhost setup.

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

Auto-detects installed agents and sets up kimbap discovery for them.

Need profile/rules customization? See the Installation Guide for advanced setup options.

Uninstall later if needed:

```bash
curl -fsSL https://kimbap.sh/install.sh | bash -s -- --uninstall
```

---

## Next steps

- [CLI Reference](docs/cli-reference.md) — all commands and flags
- [Service Development Guide](docs/service-development.md) — write your own YAML manifests
- [Installation Guide](docs/installation.md) — production setup, vault, credentials
- [Architecture](docs/architecture.md) — how the pipeline works
