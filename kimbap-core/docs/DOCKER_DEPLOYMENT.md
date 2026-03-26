# Kimbap Docker Deployment Guide

This guide covers deploying Kimbap via Docker.

## Table of Contents

- [System Overview](#system-overview)
- [Quick Start](#quick-start)
- [Requirements](#requirements)
- [Deployment Steps](#deployment-steps)
- [Configuration](#configuration)
- [Usage Guide](#usage-guide)
- [FAQ](#faq)
- [Troubleshooting](#troubleshooting)

## System Overview

Kimbap is a **secure action runtime for AI agents**. It sits between agents and external systems, handling identity, policy, credential injection, approvals, and audit.

### Key Features

- Secure action runtime pipeline (auth, policy, approval, credentials, execution, audit)
- Bearer token authentication with scope-based authorization
- Policy evaluation with human approval gates
- Encrypted vault for secret storage
- Webhook-based approval notifications (Slack, Telegram, email, generic)
- Embedded SQLite database (zero setup)
- One-click Docker deployment

## Quick Start

### One-Click Deployment

Use the automated deployment script:

```bash
curl -O https://raw.githubusercontent.com/dunialabs/kimbap-core/main/docs/docker-deploy.sh
chmod +x docker-deploy.sh
./docker-deploy.sh
```

The script will:
1. Check Docker environment
2. Generate a random vault master key
3. Create `docker-compose.yml` and `.env` files
4. Start the kimbap service
5. Wait for health checks to pass
6. Display access information

### Manual Deployment

See [Deployment Steps](#deployment-steps) below.

## Requirements

### System Requirements

- **OS**: Linux / macOS / Windows (with Docker support)
- **CPU**: 2 cores or more
- **Memory**: 2GB RAM or more
- **Disk**: 5GB available space

### Software Requirements

- **Docker**: 20.10 or higher
- **Docker Compose**: 2.0 or higher

### Port Requirements

Ensure the following port is available:

- `8080` - Kimbap API service (default, configurable)

## Deployment Steps

### Option 1: Using Automated Script (Recommended)

```bash
# 1. Download the deployment script
curl -O https://raw.githubusercontent.com/dunialabs/kimbap-core/main/docs/docker-deploy.sh
chmod +x docker-deploy.sh

# 2. Run the deployment script
./docker-deploy.sh
```

### Option 2: Manual Deployment

#### 1. Create Deployment Directory

```bash
mkdir kimbap-deployment
cd kimbap-deployment
```

#### 2. Create docker-compose.yml File

```yaml
services:
  kimbap:
    image: dunialabs/kimbap:latest
    container_name: kimbap
    restart: unless-stopped
    environment:
      KIMBAP_PORT: ${KIMBAP_PORT:-8080}
      KIMBAP_MASTER_KEY_HEX: ${KIMBAP_MASTER_KEY_HEX}
      LOG_LEVEL: ${LOG_LEVEL:-info}
    ports:
      - '${KIMBAP_PORT:-8080}:${KIMBAP_PORT:-8080}'
    volumes:
      - kimbap_data:/data/kimbap

volumes:
  kimbap_data:
    driver: local
```

#### 3. Create .env File

```bash
# Kimbap Docker Deployment Environment

# API port
KIMBAP_PORT=8080

# Vault master key (generate with: openssl rand -hex 32)
# REQUIRED for production
KIMBAP_MASTER_KEY_HEX=your-hex-master-key-change-in-production

# Logging
LOG_LEVEL=info
```

#### 4. Start Services

```bash
docker compose up -d
docker compose ps
docker compose logs -f
```

#### 5. Access the Service

- **API**: http://localhost:8080
- **Health Check**: http://localhost:8080/v1/health

## Configuration

### Required Configuration (Production)

```bash
# Generate a secure vault master key
KIMBAP_MASTER_KEY_HEX=$(openssl rand -hex 32)
```

### Port Changes

```bash
KIMBAP_PORT=9090  # Change API port from default 8080
```

## Usage Guide

### Health Check

```bash
curl http://localhost:8080/v1/health
# Returns: { "status": "ok" }
```

### List Actions

```bash
curl http://localhost:8080/v1/actions \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Execute an Action

```bash
curl -X POST "http://localhost:8080/v1/actions/github/list_pull_requests:execute" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "args": { "repo": "owner/repo" } }'
```

## FAQ

### Q1: Service fails to start?

Check if the port is in use:

```bash
lsof -i :8080
kill -9 <PID>
```

### Q2: How to update to the latest version?

```bash
docker compose pull
docker compose up -d
docker compose ps
```

### Q3: How to reset all data?

```bash
# Warning: this deletes all data including vault, audit, and tokens
docker compose down -v
docker compose up -d
```

## Troubleshooting

### View Logs

```bash
docker compose logs -f
docker compose logs -f kimbap
docker compose logs --tail 100 kimbap
```

### Restart Services

```bash
docker compose restart
docker compose restart kimbap
```

### Complete Reset

```bash
# Warning: deletes all data
docker compose down -v
docker compose up -d
```

### Health Check

```bash
curl http://localhost:8080/v1/health
```

## Monitoring and Maintenance

### Resource Monitoring

```bash
docker stats
docker system df
docker system prune -a
```

### Regular Maintenance

```bash
# Check logs for anomalies
docker compose logs --since 7d | grep -i error

# Update images
docker compose pull
docker compose up -d
```

### Data Backup

```bash
# Backup the kimbap data volume
docker run --rm -v kimbap_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/kimbap-backup-$(date +%Y%m%d).tar.gz /data
```

## Security Recommendations

1. **Set a strong vault master key**: generate with `openssl rand -hex 32`
2. **Use HTTPS**: put a TLS-terminating reverse proxy (nginx, Caddy) in front of kimbap
3. **Firewall**: restrict port 8080 to trusted sources
4. **Regular backups**: set up automated volume backup tasks
5. **Log auditing**: review audit logs regularly
6. **Keep updated**: pull the latest image regularly

## Support

- **Documentation**: check the project docs directory
- **Issue Reporting**: submit issues to the project repository
- **API Reference**: see [docs/reference.md](./reference.md)

## License

MIT License. Copyright © 2026 [Dunia Labs, Inc.](https://dunialabs.io)
