# Deployment

## Quick Start

### Prerequisites

- Go **1.24+**
- Docker and Docker Compose (for PostgreSQL and optional Cloudflare DDNS)

### Local Development

Install dependencies:

```bash
make deps
```

Start the full development environment (gateway + local database):

```bash
make dev
```

Start only the backend (if you already have PostgreSQL running):

```bash
make dev
```

Database helper commands:

```bash
docker compose up -d              # Start PostgreSQL via Docker
docker compose logs postgres      # View PostgreSQL logs
docker compose down -v            # Reset database (destructive)
docker compose down               # Stop database services
```

Build for production:

```bash
make build
```

To skip Cloudflared in development, set:

```bash
SKIP_CLOUDFLARED=true make dev
```

### Production with Docker

Kimbap Core ships with a shell script that prepares a Docker-based deployment:

```bash
curl -O https://raw.githubusercontent.com/dunialabs/kimbap-core/main/docs/docker-deploy.sh
chmod +x docker-deploy.sh
./docker-deploy.sh
```

The script will:

1. Validate your Docker environment.
2. Generate random secrets (for example `JWT_SECRET` and a database password).
3. Create a `docker-compose.yml` and `.env` file.
4. Start all services (PostgreSQL, Kimbap Core, and optional Cloudflared DDNS).
5. Wait for basic health checks.
6. Print connection information and next steps.

You can also adapt the generated files to your own Docker or orchestration setup.

### Production with Go Binary

To run Kimbap Core directly with an existing PostgreSQL database:

```bash
# 1. Clone the repository
git clone https://github.com/dunialabs/kimbap-core.git
cd kimbap-core

# 2. Install dependencies
make deps

# 3. Configure environment
cp .env.example .env
# Edit .env and set required values such as DATABASE_URL and JWT_SECRET

# 4. Build
make build

# 5. Start the service
./bin/kimbap-core
```

For process management in production you can use systemd with a service file like the following:

```ini
[Unit]
Description=Kimbap Core MCP Gateway
After=network.target postgresql.service

[Service]
Type=simple
User=kimbap
WorkingDirectory=/opt/kimbap-core
EnvironmentFile=/opt/kimbap-core/.env
ExecStart=/opt/kimbap-core/bin/kimbap-core
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Then start Kimbap Core with:

```bash
sudo systemctl enable kimbap-core
sudo systemctl start kimbap-core
```

---

## Configuration

All configuration is set via environment variables (for example in a `.env` file).

### Key Environment Variables

#### Database

| Name           | Required | Default | Description                                                                                                      |
| -------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------------- |
| `DATABASE_URL` | ✓        | –       | PostgreSQL connection string, for example `postgresql://user:password@host:5432/kimbap_mcp_gateway?schema=public`. |

#### Server

| Name            | Required | Default | Description                                               |
| --------------- | -------- | ------- | --------------------------------------------------------- |
| `BACKEND_PORT`  |          | `3002`  | HTTP port that the gateway listens on.                    |
| `BACKEND_HTTPS_PORT` |      | Value of `BACKEND_PORT`  | HTTPS listener port when `ENABLE_HTTPS=true`. Defaults to `BACKEND_PORT` if not set. |
| `ENABLE_HTTPS`  |          | `false` | Enable built-in HTTPS listener in the Go server.          |
| `SSL_CERT_PATH` |          | –       | Path to the TLS certificate file used for HTTPS.          |
| `SSL_KEY_PATH`  |          | –       | Path to the TLS private key file used for HTTPS.          |

#### Authentication

| Name         | Required          | Default | Description                                         |
| ------------ | ----------------- | ------- | --------------------------------------------------- |
| `JWT_SECRET` | ✓ (in production) | –       | Secret used to sign and verify OAuth access tokens (JWT) issued by Kimbap Core. |

OAuth 2.0 and multi-tenant settings are also configured via environment variables; refer to `../.env.example` and the API docs for the full list.

> For production deployments, treat `JWT_SECRET` as a high-value key: provision it from your secret manager or KMS, never check it into source control, and rotate it according to your organization's security policies.

#### Kimbap Auth (optional)

Kimbap Core supports multiple OAuth-based integrations (for example Google, Notion, GitHub, and Figma). There are two ways to supply OAuth credentials:

1. **Kimbap-managed credentials** (Kimbap provides `clientId` and `clientSecret`) — requires the separate `kimbap-auth` service so Kimbap secrets are never exposed.
2. **Bring your own credentials** — no `kimbap-auth` service is required.

If you are certain you will not use Kimbap-managed credentials, set `KIMBAP_AUTH_AUTOSTART='false'` to skip installing and starting `kimbap-auth`.

#### Logging

| Name         | Required | Default                      | Description                                           |
| ------------ | -------- | ---------------------------- | ----------------------------------------------------- |
| `LOG_LEVEL`  |          | `trace` (dev) / `info` (prod)  | Log level: `trace`, `debug`, `info`, `warn`, `error`. Defaults to `trace` in development and `info` in production. |
| `LOG_PRETTY` |          | `true` (dev) / `false` (prod)  | Pretty-printed logs. Defaults to enabled in development, disabled in production.            |
| `LOG_RESPONSE_MAX_LENGTH` |  | `300`   | Maximum character length for response bodies in audit logs. Longer responses are truncated.  |

#### Database

| Name           | Required | Default | Description                                                              |
| -------------- | -------- | ------- | ------------------------------------------------------------------------ |
| `AUTO_MIGRATE` |          | `false` | Run GORM AutoMigrate on startup. Set to `true` or `1` to enable.        |

#### Environment Detection

| Name       | Required | Default | Description                                                                                           |
| ---------- | -------- | ------- | ----------------------------------------------------------------------------------------------------- |
| `NODE_ENV`  |         | `development` | Environment name (`production`, `development`). Affects log verbosity and GORM log level. Defaults to `development` if neither `NODE_ENV` nor `APP_ENV` is set. |
| `APP_ENV`   |         | `development` | Alternative to `NODE_ENV`. Either variable is accepted. If both are unset, defaults to `development`.  |

#### Public URL

| Name                  | Required | Default | Description                                                                                          |
| --------------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `KIMBAP_PUBLIC_BASE_URL` |         | –       | Public-facing base URL of this Kimbap Core instance. Used for OAuth redirect URIs and metadata endpoints. |

#### MCP Server Management

| Name                  | Required | Default | Description                                                                                      |
| --------------------- | -------- | ------- | ------------------------------------------------------------------------------------------------ |
| `LAZY_START_ENABLED`  |          | `true`  | Enable lazy loading for MCP servers. When true, servers load config but delay startup until first use; idle servers auto-shutdown. |

#### Skills

| Name              | Required | Default    | Description                                                                 |
| ----------------- | -------- | ---------- | --------------------------------------------------------------------------- |
| `SKILLS_DIR`      |          | `./skills` | Directory where MCP skill definitions are stored.                           |
| `HOST_SKILLS_DIR` |          | –          | Host-absolute skills directory used to rewrite Docker bind mounts when `KIMBAP_CORE_IN_DOCKER=true`. |

#### Cloudflared DDNS (optional)

| Name                        | Required | Default              | Description                                             |
| --------------------------- | -------- | -------------------- | ------------------------------------------------------- |
| `SKIP_CLOUDFLARED`          |          | `false`              | Skip Cloudflared setup in development environments.     |
| `CLOUDFLARED_CONTAINER_DIR` |          | `/etc/cloudflared`   | Config directory path for the Cloudflared container.    |
| `KIMBAP_CORE_IN_DOCKER`       |          | –                    | Set to `true` when running inside Docker. Adjusts Cloudflared config paths. |
| `KIMBAP_CLOUD_API_URL`        |          | –                    | Kimbap Cloud API base URL for Cloudflared tunnel management. |

For additional environment variables (for example OAuth clients, multi-tenant configuration, or external services), see `../.env.example` and the deployment documentation.

---

## Docker Configuration

The default Docker setup uses the following containers and settings.

### PostgreSQL

- Container name: `kimbap-core-postgres`
- Host port: `5433` (default via `KIMBAP_POSTGRES_PORT`, container listens on `5432`)
- Database name: `kimbap_mcp_gateway`
- User/password: `kimbap` / `kimbap123` (⚠️ change these in production)

### Cloudflared DDNS (optional)

- Container name: `kimbap-core-cloudflared`
- Configuration directory: `./cloudflared`

These values come from the default Docker compose files and can be adjusted to match your environment.

### Running Kimbap Core in Docker with skills MCP servers

If a skills server launch config uses Docker bind mounts like `-v ./skills/<serverId>:/app/skills:ro`, set `HOST_SKILLS_DIR` to the host path and mount that same host directory into the kimbap-core container.

```yaml
services:
  kimbap-core:
    environment:
      KIMBAP_CORE_IN_DOCKER: "true"
      SKILLS_DIR: /data/skills
      HOST_SKILLS_DIR: /absolute/path/to/skills
    volumes:
      - ./skills:/data/skills
```

---

## Available Commands

**Development**

```bash
make dev              # Watch and run gateway + dev stack
make build            # Compile Go binary to ./bin/kimbap-core
make run              # Build and run the binary
make clean            # Clean build artifacts
```

**Database**

```bash
docker compose up -d              # Start PostgreSQL in Docker
docker compose logs postgres      # View database container logs
docker compose restart postgres   # Restart database containers
docker compose down               # Stop database containers
docker compose down -v            # Reset database (destructive)
```

> **Note**: Kimbap Core uses GORM AutoMigrate for database schema management. Migrations run automatically on startup, so there is no separate migration command. To reset the database, use `docker compose down -v && docker compose up -d`.

See `../Makefile` for the full list of commands.

---
