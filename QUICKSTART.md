# Kimbap Quick Start

## Prerequisites

- Go 1.24+
- Docker (optional, for postgres)

## Install & Run

```bash
cd kimbap-core
make deps
make build
./bin/kimbap --help
```

## Embedded Mode (local, no server)

```bash
# Install a service
./bin/kimbap service install github

# Store credentials
./bin/kimbap vault set github.token ghp_xxx

# Execute an action
./bin/kimbap call github.list-repos --input '{"owner": "octocat"}'
```

## Connected Mode (REST API server)

```bash
./bin/kimbap serve
```

API runs on http://localhost:8080.

```bash
curl http://localhost:8080/v1/health
curl http://localhost:8080/v1/actions
```

## Create a Token

```bash
./bin/kimbap token create --agent my-agent --scopes actions:execute
```

Use the returned token as `Authorization: Bearer <token>` for API calls.

## API Reference

See [docs/api/API.md](kimbap-core/docs/api/API.md) for the full endpoint list.
