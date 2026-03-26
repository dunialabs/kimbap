# CLAUDE.md

This file provides guidance to Claude Code when working with kimbap-core.

## Commands

### Core Commands
- `make build` - Build the kimbap binary (output: bin/kimbap)
- `make dev` - Run in development mode (kimbap serve)
- `make run` - Build and run (kimbap serve)
- `make test` - Run tests
- `make vet` - Run go vet
- `make clean` - Clean build artifacts
- `make deps` - Tidy and download dependencies
- `make lint` - Lint the codebase

### Docker
- `docker compose up -d` - Start services in background
- `docker compose down` - Stop and remove containers

## Architecture

kimbap is a CLI-first secure action runtime for AI agents.

Two modes:
- **Embedded mode** (`kimbap call <service.action>`): runs actions locally using installed services and local vault
- **Connected mode** (`kimbap serve`): starts a REST API server that agents call remotely

REST API server (`kimbap serve`) serves:
- `/v1/health` - health check
- `/v1/actions` - list/describe installed actions
- `/v1/actions/{service}/{action}:execute` - execute an action
- `/v1/tokens` - token management (CRUD)
- `/v1/policies` - policy get/set/evaluate
- `/v1/approvals` - list/approve/deny pending approvals
- `/v1/audit` - query audit logs
- `/v1/vault` - list vault keys
- `/console` - embedded lightweight console (optional SPA)

## Project Structure
- `cmd/kimbap/` - CLI entry point (main.go + subcommands)
- `internal/api/` - REST v1 API server (chi router)
- `internal/runtime/` - Action execution pipeline
- `internal/actions/` - Action types and interfaces
- `internal/approvals/` - Approval manager (email/slack/telegram/webhook notifiers)
- `internal/policy/` - Policy evaluator (YAML DSL)
- `internal/store/` - SQL store (SQLite default, Postgres supported)
- `internal/vault/` - Secret storage (encrypted SQLite)
- `internal/services/` - Service manifest loading and action discovery
- `internal/connectors/` - OAuth2 connector flows
- `internal/auth/` - Token service and principal types
- `internal/audit/` - Audit log writers (JSONL, multi-writer)
- `internal/console/` - Embedded SPA (static files via go:embed)
- `internal/config/` - Config loading (kimbap.yaml)
- `internal/app/` - Runtime bootstrap and adapters
- `internal/crypto/` - Encryption utilities
- `internal/webhooks/` - Webhook dispatcher

## Key Patterns
- RESTful resource routes on `/v1` (canonical)
- Bearer token auth with scope-based authorization
- Actions addressed as `service.action` (e.g., `github.create_issue`)
- Policy YAML files control which agents can call which actions

## Database
- SQLite via `internal/store/` (default, embedded, zero setup)
- Postgres via `internal/store/` (optional, for connected/multi-instance mode)
- Configure via `~/.kimbap/config.yaml` or `KIMBAP_DATA_DIR`

## Port & Environment
- Default API port: `8080` (configurable via config.yaml or `--port` flag)
- Vault master key: `KIMBAP_MASTER_KEY_HEX` (or auto-generated in dev mode)
- Dev mode: `KIMBAP_DEV=true` (relaxes security, auto-generates vault key)
