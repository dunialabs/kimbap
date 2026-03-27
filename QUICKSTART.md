# Kimbap Quick Start

## Prerequisites

- Go 1.24+
- Docker (optional, for postgres)

## Install & Run

```bash
cd kimbap
make deps
make build
./bin/kimbap --help
```

## Embedded Mode (local, no server)

```bash
# Install a service
./bin/kimbap service install github

# Store credentials
printf 'ghp_xxx' | ./bin/kimbap vault set github.token --stdin

# Execute an action
./bin/kimbap call github.list-repos --json '{"owner": "octocat"}'
```

## Connected Mode (REST API server)

```bash
./bin/kimbap serve
# enable embedded operations shell at /console for this run
./bin/kimbap serve --console
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

See [docs/api/API.md](docs/api/API.md) for the full endpoint list.
