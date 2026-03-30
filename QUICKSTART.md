# Kimbap Quick Start

Get from zero to your first action in under 5 minutes.

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
kimbap init --services all
```

Creates `~/.kimbap/` with a config file, encrypted vault, and default policy. Installs all catalog service manifests. Dev mode auto-generates a vault key stored locally, so no extra setup is needed.
Eligible shortcut aliases are set up by default during init. In interactive flows, you'll be asked first; use `--no-shortcuts` to skip.

---

## 3. Your first action

Run this:

```bash
geosearch --name "San Francisco"
```

Then fetch weather for San Francisco:

```bash
weather --latitude 37.7749 --longitude -122.4194
```

Simple flow, no API key, no localhost setup.

Canonical form is still available:

```bash
kimbap call open-meteo-geocoding.search --name "San Francisco" --count 1
kimbap call open-meteo.get-forecast --latitude 37.7749 --longitude -122.4194
```

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

If you installed via Homebrew instead, uninstall with:

```bash
brew uninstall dunialabs/kimbap/kimbap
```

---

## Next steps

- [CLI Reference](docs/cli-reference.md) — all commands and flags
- [Service Development Guide](docs/service-development.md) — write your own YAML manifests
- [Installation Guide](docs/installation.md) — production setup, vault, credentials
- [Architecture](docs/architecture.md) — how the pipeline works
