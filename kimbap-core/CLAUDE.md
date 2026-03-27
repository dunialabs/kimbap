# CLAUDE.md

This file provides guidance to Claude Code when working with kimbap-core.

## Commands

### Core Commands
- `make build` - Build the kimbap binary (output: bin/kimbap)
- `make dev` - Run in development mode (kimbap daemon)
- `make run` - Build and run (kimbap daemon)
- `make test` - Run tests
- `make vet` - Run go vet
- `make clean` - Clean build artifacts
- `make deps` - Tidy and download dependencies
- `make lint` - Lint the codebase

## Architecture

kimbap is a CLI-first secure action runtime for AI agents. It runs actions locally using installed services and the local encrypted vault.

## Key CLI Commands

### Service management
- `kimbap service install <file|name>` — install a service manifest
- `kimbap service list` — list installed services
- `kimbap service export-agent-skill` — export SKILL.md for agent discovery

### Action execution
- `kimbap call <service>.<action> [--arg value]` — call an action directly
- `kimbap search <query>` — search installed actions

### Credential and connector management
- `kimbap vault set <key>` — store a secret in the encrypted vault
- `kimbap vault list` — list vault key metadata
- `kimbap link <service>` — link a service to vault credentials or OAuth connector
- `kimbap connector login <provider>` — start an OAuth connector flow
- `kimbap auth connect <provider>` — authenticate with an OAuth provider

### Policy and approvals
- `kimbap policy set --file policy.yaml` — load a policy document
- `kimbap policy get` — show the active policy
- `kimbap approve list` — list pending approvals
- `kimbap approve accept <id>` — approve a pending action

### Runtime modes
- `kimbap run -- <cmd>` — wrap an agent subprocess with credential injection
- `kimbap proxy [--port 10255]` — start HTTP proxy interceptor
- `kimbap daemon` — start a persistent background daemon (unix socket)

### Agents and setup
- `kimbap agents setup` / `kimbap agents sync` — SKILL.md management
- `kimbap init` — initialise workspace
- `kimbap doctor` — run environment diagnostics

## Project Structure
- `cmd/kimbap/` - CLI entry point
- `internal/runtime/` - Action execution pipeline
- `internal/approvals/` - Approval manager
- `internal/policy/` - Policy evaluator
- `internal/store/` - SQL store (SQLite default, Postgres optional)
- `internal/vault/` - Secret storage
- `internal/services/` - Service manifest loading
- `internal/connectors/` - OAuth2 connector flows

## Database
- SQLite via `internal/store/` (default, zero setup)
- Configure via `~/.kimbap/config.yaml` or `KIMBAP_DATA_DIR`

## Environment
- Vault master key: `KIMBAP_MASTER_KEY_HEX` (auto-generated in dev mode)
- Dev mode: `KIMBAP_DEV=true`
