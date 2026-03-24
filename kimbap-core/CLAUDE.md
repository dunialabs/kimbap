# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository, specifically the Kimbap Core core.

## Commands

### Core Commands
- `make dev` - Start development environment
- `make build` - Build for production
- `make run` - Run the built server
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make deps` - Install dependencies
- `make lint` - Lint the codebase

### Docker
- `docker compose up -d` - Start services in background
- `docker compose down` - Stop and remove containers

## Architecture

Go binary with a chi router, serving a single HTTP server on port 3002.
Endpoints:
- `/mcp` - MCP JSON-RPC protocol endpoint
- `/admin` - legacy action-code admin (frozen)
- `/user` - legacy action-code user (frozen)
- `/api/v1` - canonical REST management API (tokens, policies, approvals, audit, actions)
- `/health`, `/ready` - liveness/readiness
- OAuth2 endpoints
- Socket.IO for real-time communication

## Project Structure
- `cmd/server/` - Server entry point
- `cmd/kimbap/` - CLI tooling
- `internal/admin/` - Legacy admin functionality
- `internal/api/` - REST v1 API
- `internal/config/`, `internal/database/`, `internal/mcp/` (MCP core), `internal/middleware/`,
  `internal/oauth/`, `internal/repository/`, `internal/security/`, `internal/service/`,
  `internal/skills/`, `internal/socket/`, `internal/store/`, `internal/types/`, `internal/user/`

## Key Patterns
- Action-code routing on `/admin` and `/user` (legacy, frozen)
- RESTful resource routes on `/api/v1` (canonical)
- MCP JSON-RPC on `/mcp`
- OAuth2 standard endpoints

## Database
- PostgreSQL via GORM (internal/database/)
- Auto-migrate controlled by `AUTO_MIGRATE=true`

## Port & Environment
- Back-end port default: `BACKEND_PORT=3002` (configurable)
- `BACKEND_HTTPS_PORT` for HTTPS if used
- CORS allowed origins via `CORS_ALLOWED_ORIGINS`
- JWT secret must match across Core and Console (`JWT_SECRET`)
