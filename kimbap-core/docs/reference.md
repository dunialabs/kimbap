# Reference

## CLI Quick Reference

The complete CLI reference is in [commands.md](./commands.md).

## Core Concepts

Actions are addressed as `service.action` (e.g., `github.create-issue`).

The execution pipeline: **identify → resolve → policy → credential → execute → audit**

All CLI modes (call, proxy, run, daemon) pass through the same pipeline.

## Configuration

All configuration via environment variables or `~/.kimbap/config.yaml`.

| Variable | Default | Description |
|---|---|---|
| `KIMBAP_DATA_DIR` | `~/.kimbap` | Data directory (DB, vault, audit logs) |
| `KIMBAP_MASTER_KEY_HEX` | auto (dev) | Hex-encoded vault master key |
| `KIMBAP_DEV` | `false` | Dev mode: relaxed security, verbose logging |
| `KIMBAP_DATABASE_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `KIMBAP_DATABASE_DSN` | `$DATA_DIR/kimbap.db` | Database connection string |
| `LOG_LEVEL` | `info` | `trace`, `debug`, `info`, `warn`, `error` |

## Policy DSL

```yaml
version: 1
defaults:
  mode: deny
rules:
  - id: allow-github-read
    match:
      agent: repo-bot
      actions:
        - github.list-repos
    effect: allow
  - id: approve-stripe-refunds
    match:
      agent: billing-bot
      actions:
        - stripe.create-refund
    effect: require_approval
```

## Risk Levels

| Level | Meaning | Auto-approve |
|---|---|---|
| `low` | Read-only, reversible | ✅ |
| `medium` | Write, limited blast radius | ✅ |
| `high` | Destructive or expensive | Policy-dependent |
| `critical` | Irreversible, high-blast | Requires approval |
