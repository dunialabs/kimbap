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

### Docker
- `docker compose up -d` - Start services in background
- `docker compose down` - Stop and remove containers

## Architecture

kimbap is a CLI-first secure action runtime for AI agents. It runs actions locally using installed services and the local encrypted vault.

## Key CLI Commands

### Service management
- `kimbap service install <file|name>` — install a service manifest
- `kimbap service validate <file>` — validate a service manifest against the strict parser
- `kimbap service list` — list installed services
- `kimbap service export-agent-skill` — export SKILL.md for agent discovery

### Action execution & discovery
- `kimbap call <service>.<action> [--arg value]` — call an action directly
- `kimbap search <query>` — search installed actions by keyword or description
- `kimbap actions list` — list all installed actions

### Code generation
- `kimbap generate ts [--service <name>] [-o <file>]` — generate TypeScript input interfaces
- `kimbap generate py [--service <name>] [-o <file>]` — generate Python TypedDict inputs

### Credential and connector management
- `kimbap vault set <key>` — store a secret in the encrypted vault
- `kimbap vault list` — list vault key metadata
- `kimbap link <service>` — link a service to vault credentials or an OAuth connector
- `kimbap connector login <provider>` — start an OAuth connector flow
- `kimbap connector status` — show connector health and token state
- `kimbap auth connect <provider>` — authenticate with an OAuth provider
- `kimbap auth revoke <provider>` — revoke an OAuth session

### Policy and approvals
- `kimbap policy set --file policy.yaml` — load a policy document
- `kimbap policy get` — show the active policy
- `kimbap approve list` — list pending approvals
- `kimbap approve accept <id>` — approve a pending action

### Audit
- `kimbap audit tail` — stream recent audit entries
- `kimbap audit export` — export audit records

### Runtime modes
- `kimbap run -- <cmd>` — wrap an agent subprocess with credential injection
- `kimbap proxy [--port 10255]` — start HTTP proxy interceptor
- `kimbap daemon` — start a persistent background daemon (unix socket)

### Agents and setup
- `kimbap agents setup` — set up SKILL.md for agent discovery
- `kimbap agents sync` — sync installed actions to SKILL.md
- `kimbap agent-profile install <profile>` — install an agent operating profile
- `kimbap agent-profile list` — list installed agent profiles
- `kimbap agent-profile print <profile>` — print an agent profile
- `kimbap init` — initialise a new kimbap workspace
- `kimbap doctor` — run environment diagnostics

## Project Structure
- `cmd/kimbap/` - CLI entry point (main.go + subcommands)
- `internal/runtime/` - Action execution pipeline
- `internal/actions/` - Action types and interfaces
- `internal/approvals/` - Approval manager (email/slack/telegram/webhook notifiers)
- `internal/policy/` - Policy evaluator (YAML DSL)
- `internal/store/` - SQL store (SQLite default, Postgres supported)
- `internal/vault/` - Secret storage (encrypted SQLite)
- `internal/services/` - Service manifest loading and action discovery
- `internal/connectors/` - OAuth2 connector flows
- `internal/audit/` - Audit log writers (JSONL, multi-writer)
- `internal/config/` - Config loading (config.yaml)
- `internal/app/` - Runtime bootstrap and adapters
- `internal/crypto/` - Encryption utilities

## Key Patterns
- Actions addressed as `service.action` (e.g., `github.create-issue`)
- Policy YAML files control which agents can call which actions

## Database
- SQLite via `internal/store/` (default, embedded, zero setup)
- Postgres via `internal/store/` (optional, for multi-instance)
- Configure via `~/.kimbap/config.yaml` or `KIMBAP_DATA_DIR`

## Port & Environment
- Vault master key: `KIMBAP_MASTER_KEY_HEX` (or auto-generated in dev mode)
- Dev mode: `KIMBAP_DEV=true` (relaxes security, auto-generates vault key)
