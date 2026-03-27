# API

The kimbap REST API (`kimbap serve`) has been removed. All functionality is available through the CLI.

## CLI Equivalents

| Was (`POST /v1/...`) | Now (`kimbap ...`) |
|---|---|
| `/v1/actions/{svc}/{action}:execute` | `kimbap call <svc>.<action>` |
| `/v1/approvals` | `kimbap approve list` |
| `/v1/approvals/{id}:approve` | `kimbap approve accept <id>` |
| `/v1/audit` | `kimbap audit tail` / `kimbap audit export` |
| `/v1/policies` | `kimbap policy set` / `kimbap policy get` |
| `/v1/vault` | `kimbap vault list` / `kimbap vault set` |

See [commands.md](../commands.md) for full CLI reference.
