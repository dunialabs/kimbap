# API

Kimbap exposes a REST surface when running `kimbap serve`.

CLI remains the primary interface, and each core API capability has a direct CLI equivalent.

## CLI Equivalents

| API (`/v1/...`) | CLI (`kimbap ...`) |
|---|---|
| `/v1/actions/{svc}/{action}:execute` | `kimbap call <svc>.<action>` |
| `/v1/approvals` | `kimbap approve list` |
| `/v1/approvals/{id}:approve` | `kimbap approve accept <id>` |
| `/v1/audit` | `kimbap audit tail` / `kimbap audit export` |
| `/v1/policies` | `kimbap policy set` / `kimbap policy get` |
| `/v1/vault` | `kimbap vault list` / `kimbap vault set` |

See [commands.md](../commands.md) for full CLI reference.
