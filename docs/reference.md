# Reference

## CLI Quick Reference

The complete CLI reference is in [cli-reference.md](./cli-reference.md).

## Core Concepts

Actions are addressed as `service.action` (e.g., `github.create-issue`).

The execution pipeline: **identify → resolve → policy → credential → execute → audit**

All CLI modes (call, proxy, run, serve, daemon) pass through the same pipeline.

## Configuration

All configuration via environment variables or `~/.kimbap/config.yaml`.

| Variable | Default | Description |
|---|---|---|
| `KIMBAP_DATA_DIR` | `~/.kimbap` | Data directory (DB, vault, audit logs) |
| `KIMBAP_MASTER_KEY_HEX` | auto (dev) | Hex-encoded vault master key |
| `KIMBAP_DEV` | `false` | Dev mode: relaxed security, verbose logging |
| `KIMBAP_DATABASE_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `KIMBAP_DATABASE_DSN` | `$DATA_DIR/kimbap.db` | Database connection string |
| `KIMBAP_LOG_LEVEL` | `info` | `trace`, `debug`, `info`, `warn`, `error` |

## Policy DSL

```yaml
version: "1.0.0"
rules:
  - id: allow-github-read
    priority: 10
    match:
      agents:
        - repo-bot
      actions:
        - github.list-repos
    decision: allow
  - id: approve-stripe-refunds
    priority: 20
    match:
      agents:
        - billing-bot
      actions:
        - stripe.create-refund
    decision: require_approval
```

## Risk Levels

| Level | Meaning | Auto-approve |
|---|---|---|
| `low` | Read-only, reversible | ✅ |
| `medium` | Write, limited blast radius | ✅ |
| `high` | Destructive or expensive | Policy-dependent |
| `critical` | Irreversible, high-blast | Requires approval |
