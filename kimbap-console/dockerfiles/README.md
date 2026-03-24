# Kimbap Console & Core Docker Compose Deployment Guide

This guide explains how to deploy Kimbap Console and Kimbap Core services using Docker Compose.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Service Overview](#service-overview)
- [Quick Start](#quick-start)
- [Using Profiles](#using-profiles)
- [Environment Variables](#environment-variables)
- [Common Commands](#common-commands)
- [Database Management](#database-management)
- [Troubleshooting](#troubleshooting)
- [Important Notes](#important-notes)

## 🔧 Prerequisites

- Docker Engine 20.10+ or Docker Desktop
- Docker Compose V2 (recommended) or Docker Compose V1
- At least 2GB available memory
- At least 5GB available disk space

### Check Docker Version

```bash
docker --version
docker compose version
```

> **Note**: Newer versions of Docker Desktop use `docker compose` (with space) instead of `docker-compose` (with hyphen).

## 🏗️ Service Overview

### Console Services

- **postgres-console**: PostgreSQL database for Console service
  - Port: 5432 (default, configurable via environment variables)
  - Database name: `kimbap_console`
  - Username: `kimbap`
  - Password: `kimbap123`

- **kimbap-console**: Kimbap Console web interface service
  - Port: 3000 (default, configurable via environment variables)
  - Access URL: http://localhost:3000

### Core Services

- **postgres-core**: PostgreSQL database for Core service
  - Port: 5432 (default, configurable via environment variables)
  - Database name: `kimbap_mcp_gateway`
  - Username: `kimbap`
  - Password: `kimbap123`

- **kimbap-core**: Kimbap Core MCP Gateway service
  - Port: 3002 (default, configurable via environment variables)
  - Access URL: http://localhost:3002
  - Health check: http://localhost:3002/health

## 🚀 Quick Start

### Start All Services

```bash
docker compose up -d
```

### Check Service Status

```bash
docker compose ps
```

### View Service Logs

```bash
# View all service logs
docker compose logs -f

# View specific service logs
docker compose logs -f kimbap-console
docker compose logs -f kimbap-core
```

### Stop All Services

```bash
docker compose down
```

### Start Services with Custom Ports

You can specify custom ports for Console and Core services when starting them.

#### Start Console with Custom Port

```bash
# Start Console on port 3001
CONSOLE_PORT=3001 docker compose --profile console up -d

# Or using .env file
echo "CONSOLE_PORT=3001" > .env
docker compose --profile console up -d
```

#### Start Core with Custom Port

```bash
# Start Core on port 3003
CORE_PORT=3003 docker compose --profile core up -d

# Or using .env file
echo "CORE_PORT=3003" > .env
docker compose --profile core up -d
```

#### Start Both Services with Custom Ports

```bash
# Start both Console and Core with custom ports
CONSOLE_PORT=3001 CORE_PORT=3003 docker compose --profile console --profile core up -d
```

**Note**: When using custom ports, make sure to:
- Update `MCP_GATEWAY_URL` and `PROXY_ADMIN_URL` if Core port is changed: `MCP_GATEWAY_URL=http://localhost:3003`

**Example with all custom configurations**:

```bash
CONSOLE_PORT=3001 \
CORE_PORT=3003 \
MCP_GATEWAY_URL=http://localhost:3003 \
PROXY_ADMIN_URL=http://localhost:3003/admin \
docker compose --profile console --profile core up -d
```

## 🎯 Using Profiles

The Docker Compose configuration uses **Profiles** feature, allowing you to selectively start different service groups.

### Profile Overview

- **`console`**: Console-related services (postgres-console + kimbap-console)
- **`core`**: Core-related services (postgres-core + kimbap-core)

### Usage Scenarios

#### 1. Start Console Services Only

Use this when you only need the Console web interface and Core service is running elsewhere.

```bash
docker compose --profile console up -d
```

**With custom port**:

```bash
# Start Console on port 3001
CONSOLE_PORT=3001 docker compose --profile console up -d
```

**Important**: If Core service is not running in the same Docker Compose, you need to set environment variables pointing to the external Core service:

```bash
MCP_GATEWAY_URL=http://localhost:3002 \
PROXY_ADMIN_URL=http://localhost:3002/admin \
docker compose --profile console up -d
```

**With custom Console port and external Core**:

```bash
CONSOLE_PORT=3001 \
MCP_GATEWAY_URL=http://localhost:3002 \
PROXY_ADMIN_URL=http://localhost:3002/admin \
docker compose --profile console up -d
```

#### 2. Start Core Services Only

Use this when you only need to run the Core MCP Gateway service.

```bash
docker compose --profile core up -d
```

**With custom port**:

```bash
# Start Core on port 3003
CORE_PORT=3003 docker compose --profile core up -d
```

#### 3. Start Both Console and Core Services

```bash
docker compose --profile console --profile core up -d
```

**With custom ports**:

```bash
# Start Console on port 3001 and Core on port 3003
CONSOLE_PORT=3001 \
CORE_PORT=3003 \
MCP_GATEWAY_URL=http://kimbap-core:3002 \
PROXY_ADMIN_URL=http://kimbap-core:3002/admin \
docker compose --profile console --profile core up -d
```

> **Note**: When both services are in the same Docker Compose, use the service name `kimbap-core` instead of `localhost` for internal communication.

#### 4. Start All Services (Default)

If no profile is specified, no services will start by default (since all services have profiles configured). To start all services, you need to specify both profiles:

```bash
docker compose --profile console --profile core up -d
```

### Stop Services by Profile

```bash
# Stop Console-related services
docker compose --profile console down

# Stop Core-related services
docker compose --profile core down

# Stop all services
docker compose down
```

## ⚙️ Environment Variables

You can configure services via environment variables or a `.env` file. Create a `.env` file (optional):

```bash
# Console configuration
CONSOLE_PORT=3000
CONSOLE_JWT_SECRET=your-console-jwt-secret-change-in-production-min-32-chars

# Core configuration
CORE_PORT=3002
CORE_JWT_SECRET=your-core-jwt-secret-change-in-production-min-32-chars

# Database configuration
POSTGRES_USER=kimbap
POSTGRES_PASSWORD=kimbap123
POSTGRES_DB=kimbap_console  # Console database name
POSTGRES_PORT=5432

# MCP Gateway connection configuration (when Console and Core are deployed separately)
MCP_GATEWAY_URL=http://localhost:3002
PROXY_ADMIN_URL=http://localhost:3002/admin

# Log sync configuration (optional)
LOG_SYNC_ENABLED=true
LOG_SYNC_INTERVAL_MINUTES=2
MAX_LOGS_PER_REQUEST=5000
LOG_BATCH_SIZE=500
LOG_SYNC_TIMEOUT=180000
LOG_SYNC_RETRY_ATTEMPTS=2
```

### Start with Environment Variables

You can set environment variables inline or use a `.env` file:

**Inline environment variables**:

```bash
CONSOLE_PORT=3001 CORE_PORT=3003 docker compose --profile console --profile core up -d
```

**Using .env file**:

1. Create a `.env` file in the same directory as `docker-compose.yml`:

```bash
CONSOLE_PORT=3001
CORE_PORT=3003
MCP_GATEWAY_URL=http://kimbap-core:3002
PROXY_ADMIN_URL=http://kimbap-core:3002/admin
```

2. Start services:

```bash
docker compose --profile console --profile core up -d
```

### Port Configuration Summary

| Service | Environment Variable | Default Port | Description |
|---------|---------------------|--------------|-------------|
| Console | `CONSOLE_PORT` | 3000 | Web interface port |
| Core | `CORE_PORT` | 3002 | MCP Gateway API port |

**Important**: When changing ports, remember to update related URLs:
- `MCP_GATEWAY_URL` should match `CORE_PORT` (when connecting externally)
- `PROXY_ADMIN_URL` should match `CORE_PORT` (when connecting externally)

## 📝 Common Commands

### Check Service Status

```bash
docker compose ps
```

### View Service Logs

```bash
# View all logs in real-time
docker compose logs -f

# View last 100 lines of logs
docker compose logs --tail=100

# View specific service logs
docker compose logs kimbap-console
docker compose logs kimbap-core
docker compose logs postgres-console
```

### Restart Services

```bash
# Restart all services
docker compose restart

# Restart specific service
docker compose restart kimbap-console
docker compose restart kimbap-core
```

### Access Containers

```bash
# Access Console container
docker exec -it kimbap-console sh

# Access Core container
docker exec -it kimbap-core sh

# Access database containers
docker exec -it kimbap-console-postgres psql -U kimbap -d kimbap_console
docker exec -it kimbap-core-postgres psql -U kimbap -d kimbap_mcp_gateway
```

### View Resource Usage

```bash
docker stats
```

## 🗄️ Database Management

### Reset Database

#### Reset Console Database

```bash
# Stop services and remove data volumes
docker compose --profile console down -v

# Restart
docker compose --profile console up -d
```

#### Reset Core Database

```bash
# Stop services and remove data volumes
docker compose --profile core down -v

# Restart
docker compose --profile core up -d
```

#### Reset All Databases

```bash
# Stop all services and remove all data volumes
docker compose down -v

# Restart
docker compose --profile console --profile core up -d
```

### Backup Database

```bash
# Backup Console database
docker exec kimbap-console-postgres pg_dump -U kimbap kimbap_console > console_backup.sql

# Backup Core database
docker exec kimbap-core-postgres pg_dump -U kimbap kimbap_mcp_gateway > core_backup.sql
```

### Restore Database

```bash
# Restore Console database
docker exec -i kimbap-console-postgres psql -U kimbap kimbap_console < console_backup.sql

# Restore Core database
docker exec -i kimbap-core-postgres psql -U kimbap kimbap_mcp_gateway < core_backup.sql
```

### Connect to Database

```bash
# Console database
docker exec -it kimbap-console-postgres psql -U kimbap -d kimbap_console

# Core database
docker exec -it kimbap-core-postgres psql -U kimbap -d kimbap_mcp_gateway
```

## 🔍 Troubleshooting

### Port Conflicts

If you encounter port already in use errors:

```bash
# Check port usage (macOS/Linux)
lsof -i :3000
lsof -i :3002
lsof -i :5432

# Stop the process using the port
kill <PID>
```

Or modify the port configuration in the `.env` file.

### Services Won't Start

1. **Check logs**:
   ```bash
   docker compose logs <service-name>
   ```

2. **Check health status**:
   ```bash
   docker compose ps
   ```

3. **Restart service**:
   ```bash
   docker compose restart <service-name>
   ```

### Database Connection Failed

1. **Check if database is healthy**:
   ```bash
   docker compose ps
   # Check status of postgres-console or postgres-core
   ```

2. **Check database logs**:
   ```bash
   docker compose logs postgres-console
   docker compose logs postgres-core
   ```

3. **Verify database connection**:
   ```bash
   docker exec kimbap-console-postgres pg_isready -U kimbap -d kimbap_console
   docker exec kimbap-core-postgres pg_isready -U kimbap -d kimbap_mcp_gateway
   ```

### Network Issues

If services cannot communicate with each other:

1. **Check network**:
   ```bash
   docker network ls
   docker network inspect testconsole_kimbap-network
   ```

2. **Recreate network**:
   ```bash
   docker compose down
   docker compose up -d
   ```

### Clean Up All Resources

If you need to completely clean up (including data volumes):

```bash
# Stop and remove all containers, networks, and data volumes
docker compose down -v

# Remove images (optional)
docker rmi kimbapio/kimbap-console:latest kimbapio/kimbap-core:latest
```

## ⚠️ Important Notes

1. **Production Configuration**:
   - Change default passwords (`kimbap123`)
   - Set strong JWT_SECRET (minimum 32 characters)
   - Use environment variables or secret management services to store sensitive information
   - Configure HTTPS

2. **Data Persistence**:
   - Database data is stored in Docker volumes
   - Regularly backup important data
   - For production, consider using external databases or configure database backup strategies

3. **Resource Limits**:
   - Adjust container resource limits based on actual needs
   - Monitor memory and CPU usage

4. **Security Recommendations**:
   - Do not commit `.env` files containing sensitive information to version control
   - Use Docker secrets to manage sensitive configurations
   - Regularly update image versions

5. **Performance Optimization**:
   - Adjust `LOG_SYNC_INTERVAL_MINUTES` based on log sync requirements
   - Adjust `MAX_LOGS_PER_REQUEST` and `LOG_BATCH_SIZE` based on log volume

## 📚 Related Resources

- [Docker Compose Official Documentation](https://docs.docker.com/compose/)
- [PostgreSQL Official Documentation](https://www.postgresql.org/docs/)

## 🤝 Support

For issues, please check the logs or contact technical support.

---

**Last Updated**: 2025-12-04
