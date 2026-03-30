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
kimbap init --services select
```

Creates local runtime files (default in `~/.kimbap/`) and opens an interactive service checklist. Recommended services are preselected, and `all` is available in one command.
Shortcut aliases are set up by default during init. Use `--no-shortcuts` to skip.

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

Auto-detects installed agents and installs global kimbap discovery hints. For project sync, run:

```bash
kimbap agents setup --sync --dir .
```

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
