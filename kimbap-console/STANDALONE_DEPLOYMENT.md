# KIMBAP Console Standalone Deployment Guide

This guide explains how to deploy KIMBAP Console by itself while connecting to an external KIMBAP Core service.

## When to Use
- ✅ You already have a running KIMBAP Core service
- ✅ You need the web management UI
- ✅ Multiple Console instances share one Core
- ✅ Frontend/backend separation

## Quick Start

### 1) Prepare environment variables
Create `.env` under `kimbap-console`:
```bash
cp .env.example .env
nano .env
```

**Required:**
```bash
DATABASE_URL="postgresql://kimbap:kimbap123@postgres-console:5432/kimbap_console?schema=public"
JWT_SECRET="your-console-jwt-secret-change-in-production-min-32-chars"
MCP_GATEWAY_URL="http://localhost:3002"          # Core address
PROXY_ADMIN_URL="http://localhost:3002/admin"
```

**Optional:**
```bash
CONSOLE_PORT=3000
LOG_SYNC_ENABLED=true
LOG_SYNC_INTERVAL_MINUTES=2
MAX_LOGS_PER_REQUEST=5000
LOG_BATCH_SIZE=500
LOG_SYNC_TIMEOUT=180000
LOG_SYNC_RETRY_ATTEMPTS=2
```

### 2) Start services
```bash
docker-compose up -d
docker-compose logs -f
docker-compose ps
```

### 3) Access
Open http://localhost:3000

### 4) Stop
```bash
docker-compose down
# Remove volumes (clears data)
docker-compose down -v
```

## Connecting to External KIMBAP Core
- **Same host:** `MCP_GATEWAY_URL=http://localhost:3002`, `PROXY_ADMIN_URL=http://localhost:3002/admin`
- **Remote host (e.g., 192.168.1.100):** set URLs to that host and test with `curl http://192.168.1.100:3002/health`
- **Domain:** `MCP_GATEWAY_URL=https://core.example.com`, `PROXY_ADMIN_URL=https://core.example.com/admin`

## Log Sync
Console can sync logs from KIMBAP Core into its local DB. Control via env vars:
```bash
LOG_SYNC_ENABLED=true
LOG_SYNC_INTERVAL_MINUTES=2
MAX_LOGS_PER_REQUEST=5000
LOG_BATCH_SIZE=500
LOG_SYNC_TIMEOUT=180000
LOG_SYNC_RETRY_ATTEMPTS=2
```
Disable with `LOG_SYNC_ENABLED=false`. View sync logs with `docker-compose logs -f kimbap-console | grep LogSync`.

## Config Reference
### Database
Console needs its own database (separate from Core):
```yaml
postgres-console:
  image: postgres:16-alpine
  environment:
    POSTGRES_USER: kimbap
    POSTGRES_PASSWORD: kimbap123
    POSTGRES_DB: kimbap_console
```

### Env variables
| Variable | Required | Default | Description |
|------|------|--------|------|
| `DATABASE_URL` | ✅ | - | PostgreSQL connection string |
| `JWT_SECRET` | ✅ | - | JWT secret (32+ chars) |
| `MCP_GATEWAY_URL` | ✅ | http://localhost:3002 | Core gateway URL |
| `PROXY_ADMIN_URL` | ✅ | http://localhost:3002/admin | Core admin endpoint |
| `CONSOLE_PORT` | ❌ | 3000 | Service port |
| `LOG_SYNC_ENABLED` | ❌ | true | Enable log sync |

## Data Management
- **Backup:** `docker exec kimbap-console-postgres pg_dump -U kimbap -d kimbap_console > console-backup-$(date +%Y%m%d).sql`
- **Restore:** `docker exec -i kimbap-console-postgres psql -U kimbap -d kimbap_console < console-backup.sql`

## Troubleshooting
- Console won’t start: `docker-compose logs kimbap-console`, `docker-compose config`, `lsof -i :3000`
- Cannot reach Core: `curl http://your-core-host:3002/health`, check env with `docker exec kimbap-console env | grep MCP`, and view logs.
- Log sync failures: `docker-compose logs kimbap-console | grep LogSync`, check `LOG_SYNC_*` envs, and verify Core connectivity.

## Performance Tips
- Reduce sync load: `LOG_SYNC_INTERVAL_MINUTES=5`, `MAX_LOGS_PER_REQUEST=1000`
- Clean old logs: connect to Postgres and run `DELETE FROM "Log" WHERE "addtime" < EXTRACT(EPOCH FROM NOW() - INTERVAL '30 days');`

## Security Checklist
- Change all default passwords/secrets
- Use HTTPS via reverse proxy
- Set firewall rules
- Enable audit logging
- Back up regularly
- Restrict DB external access
- Configure CORS
- Keep secrets in environment variables

Example Nginx TLS proxy:
```nginx
server {
    listen 443 ssl;
    server_name console.example.com;
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Upgrades
- Pull latest image and restart: `docker pull kimbapio/kimbap-console:latest && docker-compose down && docker-compose up -d`
- Check version: `docker exec kimbap-console cat package.json | grep version`
- Migrations run automatically on startup.

## Local Development
```bash
npm install
cp .env.example .env
nano .env
npm run dev
open http://localhost:3000
```

## Related Docs
- [Full deployment guide](../DEPLOYMENT_GUIDE.md) for Console + Core
- [Core-only deployment](../CORE_ONLY_DEPLOYMENT.md)
- [Docker build guide](./DOCKER_BUILD.md)
- [Env variable reference](./.env.example)

## Support
- GitHub Issues: https://github.com/kimbapio/kimbap/issues
- Docs: https://docs.kimbap.ai
