# KIMBAP Console One-Click Scripts

## ✨ Features
- ✅ Fully self-contained: ships all config without extra yml files
- ✅ Cross-platform: macOS, Linux, Windows (Git Bash/WSL)
- ✅ Auto config: generates docker-compose.yml and .env
- ✅ Secure keys: creates random 32-char secrets
- ✅ Smart detection: finds docker-compose vs docker compose
- ✅ Interactive wizard with prompts (formerly Chinese UI)

## 🚀 Quick Start

### 1) Download the script
```bash
curl -O https://your-url/start-console.sh
chmod +x start-console.sh
# or copy from this repo
cp /path/to/kimbap-console/start-console.sh ~/my-console/
cd ~/my-console
```

### 2) Run
```bash
./start-console.sh
```
The script will:
1. Detect OS and Docker availability
2. Generate `docker-compose.console.yml`
3. Create `.env` with random secrets
4. Pull the latest Docker images
5. Start services
6. Print URLs and management commands

### 3) First-run config
- Generates secure random keys (32 chars)
- Prompts for the Kimbap Core connection URL
- Asks whether to edit the config now

Key values to set:
```bash
MCP_GATEWAY_URL=http://localhost:3002      # Update to your Core address
PROXY_ADMIN_URL=http://localhost:3002/admin
```

## 📋 Management Scripts
- `start-console.sh` – generate config, pull images, and start Console + Postgres
- `stop-console.sh` – stop services (optionally delete data)
- `restart-console.sh` – restart services while keeping data
- `update-console.sh` – pull the latest image and restart
- `logs-console.sh` – stream logs (`logs-console.sh kimbap-console` or `postgres-console`)

## 🌍 Platform Notes
- **macOS:** run `./start-console.sh`; uses `nano` or `$EDITOR`; works with Docker Desktop
- **Linux:** run `./start-console.sh`; supports systemd-managed Docker; distro agnostic
- **Windows:** run via Git Bash `./start-console.sh` or PowerShell/WSL `bash start-console.sh`; works with Docker Desktop

## 📁 Generated Files
```
your-directory/
├── start-console.sh
├── docker-compose.console.yml   # auto-generated
├── .env                         # auto-generated
└── (other helper scripts)
```
`docker-compose.console.yml` includes Postgres 16, Kimbap Console, networking/volumes, and health checks.
`.env` includes generated secrets, DB/port settings, Core URLs, and log sync config.

## 🔧 Config Notes
- Keys are generated via `openssl` when available, otherwise `md5sum` + timestamp; always 32+ chars.
- Edit `.env` to adjust values:
```bash
# Required
MCP_GATEWAY_URL=http://your-core-server:3002
PROXY_ADMIN_URL=http://your-core-server:3002/admin

# Optional
CONSOLE_PORT=3000
POSTGRES_PORT=5432
POSTGRES_PASSWORD=your-password

# Auto-generated (generally keep)
CONSOLE_JWT_SECRET=auto-generated-32-chars
```

## 📊 Usage Scenarios
- **Local dev/test:** run in any folder and access http://localhost:3000
- **Server deploy:** run under `/opt/kimbap-console`, edit `.env`, then `./restart-console.sh`
- **Multiple environments:** create separate folders (dev/test) with different ports in `.env`
- **Quick run:** download and run, then open http://localhost:3000

## 🛠️ Troubleshooting
- **Docker not running:** start Docker Desktop (macOS/Windows) or `sudo systemctl start docker` (Linux)
- **docker-compose missing:** install via Homebrew (`brew install docker-compose`) or `apt-get install docker-compose`; `docker compose version` for plugin
- **Port conflicts:** change `CONSOLE_PORT`/`POSTGRES_PORT` in `.env`, then `./restart-console.sh`
- **Cannot reach Core:** ensure Core is up; set `MCP_GATEWAY_URL` appropriately (localhost, LAN IP, or HTTPS domain) and restart

## 📦 Data Management
- Backup: `docker exec kimbap-console-postgres pg_dump -U kimbap kimbap_console > backup.sql`
- Restore: `docker exec -i kimbap-console-postgres psql -U kimbap kimbap_console < backup.sql`
- Reset DB: run `./stop-console.sh` (option 2 to remove data) then `./start-console.sh`

## 🔒 Security Tips
- Generated secrets are strong, but change `POSTGRES_PASSWORD` for production.
- Restrict access with firewall rules and HTTPS via a reverse proxy (Nginx/Caddy).
- Schedule regular backups.

## 🆘 Help
- Read more: `cat DOCKER_USAGE.md`
- View logs: `./logs-console.sh`
- Check services: `docker compose -f docker-compose.console.yml ps`

## 📝 Changelog
- **v2.0:** self-contained config, cross-platform, auto secrets, smart docker-compose detection, generates files locally
- **v1.0:** initial feature set
