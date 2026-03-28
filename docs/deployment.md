# Deployment

## Quick Start

### Prerequisites

- Go **1.24+**
- Docker and Docker Compose (optional, for running services in containers)

### Local Development

Install dependencies:

```bash
make deps
```

Start in development mode:

```bash
make dev
```

Build for production:

```bash
make build
```

### Production with Go Binary

```bash
# 1. Clone the repository
git clone https://github.com/dunialabs/kimbap.git
cd kimbap

# 2. Install dependencies
make deps

# 3. Configure (optional)
cp .env.example .env
# Edit .env to set KIMBAP_MASTER_KEY_HEX and other values

# 4. Build
make build

# 5. Start the service
./bin/kimbap serve
```

For process management in production you can use systemd:

```ini
[Unit]
Description=Kimbap - Secure action runtime for AI agents
After=network.target

[Service]
Type=simple
User=kimbap
WorkingDirectory=/opt/kimbap
EnvironmentFile=/opt/kimbap/.env
ExecStart=/opt/kimbap/bin/kimbap serve
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Then start with:

```bash
sudo systemctl enable kimbap
sudo systemctl start kimbap
```

---

## Configuration

All configuration is set via environment variables or `~/.kimbap/config.yaml`.

### Key Environment Variables

#### Server

| Name            | Required | Default | Description                                               |
| --------------- | -------- | ------- | --------------------------------------------------------- |
| `KIMBAP_DATA_DIR` |        | `~/.kimbap` | Directory for SQLite database, vault, and audit logs. |

#### Security

| Name                   | Required          | Default | Description                                                              |
| ---------------------- | ----------------- | ------- | ------------------------------------------------------------------------ |
| `KIMBAP_MASTER_KEY_HEX` |                 | auto (dev mode) | Hex-encoded master key for vault encryption. Auto-generated in dev mode. |
| `KIMBAP_DEV`           |                  | `false` | Dev mode: relaxes security, auto-generates vault key, verbose logging.   |

#### Database

| Name              | Required | Default     | Description                                                             |
| ----------------- | -------- | ----------- | ----------------------------------------------------------------------- |
| `KIMBAP_DATABASE_DRIVER` |  | `sqlite`    | Database driver: `sqlite` or `postgres`.                  |
| `KIMBAP_DATABASE_DSN`    |  | `$KIMBAP_DATA_DIR/kimbap.db` | Database DSN. For postgres: `postgres://user:pass@host:5432/kimbap?sslmode=disable` |

#### Logging

| Name         | Required | Default                        | Description                                           |
| ------------ | -------- | ------------------------------ | ----------------------------------------------------- |
| `KIMBAP_LOG_LEVEL`  |          | `info`                         | Log level: `trace`, `debug`, `info`, `warn`, `error`. |
| `KIMBAP_LOG_FORMAT` |          | `text` (default) / `json`      | Log output format.                                    |

---

## Docker Configuration

The default Docker setup runs kimbap with SQLite (embedded, zero setup).

### Basic Docker Compose

```yaml
services:
  kimbap:
    image: dunialabs/kimbap:latest
    container_name: kimbap
    restart: unless-stopped
    environment:
      KIMBAP_DATA_DIR: /data/kimbap
      KIMBAP_MASTER_KEY_HEX: ${KIMBAP_MASTER_KEY_HEX}
      KIMBAP_LOG_LEVEL: info
    volumes:
      - kimbap_data:/data/kimbap
    healthcheck:
      test: ['CMD-SHELL', 'wget --spider -q http://localhost:8080/v1/health || exit 1']
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

volumes:
  kimbap_data:
    driver: local
```

### With PostgreSQL (optional)

For multi-instance or high-availability deployments, configure an external PostgreSQL database:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: kimbap-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: kimbap
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: kimbap
    ports:
      - '5433:5432'
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U kimbap']
      interval: 10s
      timeout: 5s
      retries: 5

  kimbap:
    image: dunialabs/kimbap:latest
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      KIMBAP_DATA_DIR: /data/kimbap
      KIMBAP_MASTER_KEY_HEX: ${KIMBAP_MASTER_KEY_HEX}
      KIMBAP_DATABASE_DRIVER: postgres
      KIMBAP_DATABASE_DSN: postgres://kimbap:${DB_PASSWORD}@postgres:5432/kimbap?sslmode=disable
    volumes:
      - kimbap_data:/data/kimbap

volumes:
  postgres_data:
```

---

## Available Commands

**Development**

```bash
make dev              # Run in development mode
make build            # Compile Go binary to ./bin/kimbap
make run              # Build and run
make clean            # Clean build artifacts
```

**Docker**

```bash
docker compose up -d              # Start services
docker compose logs -f kimbap     # View logs
docker compose restart kimbap     # Restart service
docker compose down               # Stop services
docker compose down -v            # Stop and remove volumes (destructive)
```

See `../Makefile` for the full list of commands.

---
