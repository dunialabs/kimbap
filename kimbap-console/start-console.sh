#!/bin/bash

# Kimbap Console One-Click Start Script
# Compatible with macOS, Linux, Windows (Git Bash/WSL)
# All configuration files are automatically generated in the current directory

set -e

echo "=========================================="
echo "  Kimbap Console Startup"
echo "=========================================="
echo ""

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)    echo "macos";;
        Linux*)     echo "linux";;
        CYGWIN*|MINGW*|MSYS*)    echo "windows";;
        *)          echo "unknown";;
    esac
}

OS_TYPE=$(detect_os)
echo "Detected OS: $OS_TYPE"
echo ""

# Check if Docker is running
echo "Checking environment..."
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    case $OS_TYPE in
        macos)
            echo "Please start Docker Desktop for Mac"
            ;;
        windows)
            echo "Please start Docker Desktop for Windows"
            ;;
        linux)
            echo "Please run: sudo systemctl start docker"
            ;;
    esac
    exit 1
fi
echo -e "${GREEN}✓${NC} Docker is running"

# Check docker-compose or docker compose command
if command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    echo -e "${RED}Error: docker-compose not found${NC}"
    echo "Please install Docker Compose"
    exit 1
fi
echo -e "${GREEN}✓${NC} Docker Compose available ($COMPOSE_CMD)"
echo ""

# Get current working directory
WORK_DIR="$(pwd)"
COMPOSE_FILE="$WORK_DIR/docker-compose.console.yml"
ENV_FILE="$WORK_DIR/.env"

echo "Working directory: $WORK_DIR"
echo ""

# Generate docker-compose.console.yml
echo "Generating docker-compose.console.yml..."
cat > "$COMPOSE_FILE" << 'EOF'
version: '3.8'

services:
  # PostgreSQL for kimbap-console
  postgres-console:
    image: postgres:16-alpine
    container_name: kimbap-console-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-kimbap}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-kimbap123}
      POSTGRES_DB: ${POSTGRES_DB:-kimbap_console}
    volumes:
      - postgres-console-data:/var/lib/postgresql/data
    ports:
      - '${POSTGRES_PORT:-5432}:5432'
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U ${POSTGRES_USER:-kimbap} -d ${POSTGRES_DB:-kimbap_console}']
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kimbap-network

  # Kimbap Console Service
  kimbap-console:
    image: dunialabs/kimbap-console:latest
    container_name: kimbap-console
    restart: unless-stopped
    depends_on:
      postgres-console:
        condition: service_healthy
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://${POSTGRES_USER:-kimbap}:${POSTGRES_PASSWORD:-kimbap123}@postgres-console:5432/${POSTGRES_DB:-kimbap_console}?schema=public
      PORT: ${CONSOLE_PORT:-3000}
      JWT_SECRET: ${CONSOLE_JWT_SECRET:-your-console-jwt-secret-change-in-production-min-32-chars}
      MCP_GATEWAY_URL: ${MCP_GATEWAY_URL:-http://kimbap-core:3002}
      PROXY_ADMIN_URL: ${PROXY_ADMIN_URL:-http://kimbap-core:3002/admin}
      LOG_SYNC_ENABLED: ${LOG_SYNC_ENABLED:-true}
      LOG_SYNC_INTERVAL_MINUTES: ${LOG_SYNC_INTERVAL_MINUTES:-2}
      MAX_LOGS_PER_REQUEST: ${MAX_LOGS_PER_REQUEST:-5000}
      LOG_BATCH_SIZE: ${LOG_BATCH_SIZE:-500}
      LOG_SYNC_TIMEOUT: ${LOG_SYNC_TIMEOUT:-180000}
      LOG_SYNC_RETRY_ATTEMPTS: ${LOG_SYNC_RETRY_ATTEMPTS:-2}
      GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID:-}
      GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET:-}
      NOTION_CLIENT_ID: ${NOTION_CLIENT_ID:-}
      NOTION_CLIENT_SECRET: ${NOTION_CLIENT_SECRET:-}
    ports:
      - '${CONSOLE_PORT:-3000}:3000'
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:3000']
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s

  # Cloudflare Tunnel (Optional - for HTTPS remote access)
  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: kimbap-cloudflared
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN:-}
    networks:
      - kimbap-network
    volumes:
      - ./cloudflared:/etc/cloudflared
    depends_on:
      kimbap-console:
        condition: service_started
    profiles:
      - cloudflared

volumes:
  postgres-console-data:
    driver: local

networks:
  kimbap-network:
    driver: bridge
EOF

echo -e "${GREEN}✓${NC} docker-compose.console.yml generated"

# Check or generate .env file
if [ ! -f "$ENV_FILE" ]; then
    echo ""
    echo -e "${YELLOW}Warning: .env file does not exist, generating default configuration...${NC}"
    echo ""

    # Generate random secret
    generate_secret() {
        if command -v openssl &> /dev/null; then
            openssl rand -base64 32 | tr -d "=+/" | cut -c1-32
        else
            # fallback: use date and random
            echo "$(date +%s)${RANDOM}${RANDOM}" | md5sum 2>/dev/null | cut -c1-32 || echo "change-this-secret-key-in-prod-$(date +%s)"
        fi
    }

    JWT_SECRET=$(generate_secret)

    cat > "$ENV_FILE" << EOF
# PostgreSQL Database Configuration
POSTGRES_USER=kimbap
POSTGRES_PASSWORD=kimbap123
POSTGRES_DB=kimbap_console
POSTGRES_PORT=5432

# Console Service Port
CONSOLE_PORT=3000

# Authentication Secrets (auto-generated, recommend changing in production)
CONSOLE_JWT_SECRET=$JWT_SECRET

# Kimbap Core Connection Configuration
# Same machine: http://localhost:3002 or http://kimbap-core:3002
# Different server: http://192.168.1.100:3002 or https://core.example.com
MCP_GATEWAY_URL=http://kimbap-core:3002
PROXY_ADMIN_URL=http://kimbap-core:3002/admin

# Log Sync Configuration
LOG_SYNC_ENABLED=true
LOG_SYNC_INTERVAL_MINUTES=2
MAX_LOGS_PER_REQUEST=5000
LOG_BATCH_SIZE=500
LOG_SYNC_TIMEOUT=180000
LOG_SYNC_RETRY_ATTEMPTS=2

# OAuth Configuration (optional)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
NOTION_CLIENT_ID=
NOTION_CLIENT_SECRET=

# Cloudflare Tunnel Configuration (Optional - for HTTPS remote access)
# Get your tunnel token from: https://dash.cloudflare.com/
# Zero Trust → Networks → Tunnels → Create tunnel
# See docs/HTTPS_REMOTE_DEPLOYMENT.md for detailed instructions
CLOUDFLARE_TUNNEL_TOKEN=
EOF

    echo -e "${GREEN}✓${NC} .env file generated (secrets auto-generated)"
    echo ""
    echo -e "${BLUE}Important Configuration Notes:${NC}"
    echo "   1. Authentication secrets have been auto-generated (check .env file)"
    echo "   2. Please modify MCP_GATEWAY_URL and PROXY_ADMIN_URL to your Kimbap Core address"
    echo "   3. To enable HTTPS remote access, set CLOUDFLARE_TUNNEL_TOKEN in .env file"
    echo "   4. To modify ports or other settings, edit .env file"
    echo ""

    # Ask if user wants to edit now
    read -p "Edit .env file now to configure Core address? (y/n): " edit_choice
    if [[ "$edit_choice" =~ ^[Yy]$ ]]; then
        # Choose editor based on OS
        case $OS_TYPE in
            macos)
                ${EDITOR:-nano} "$ENV_FILE"
                ;;
            windows)
                ${EDITOR:-notepad} "$ENV_FILE" 2>/dev/null || nano "$ENV_FILE"
                ;;
            linux)
                ${EDITOR:-nano} "$ENV_FILE"
                ;;
        esac
    fi
else
    echo -e "${GREEN}✓${NC} .env file already exists"
fi

echo ""
echo "=========================================="
echo "  Pulling Images"
echo "=========================================="
echo ""

# Pull latest images
echo "Pulling latest images..."
$COMPOSE_CMD -f "$COMPOSE_FILE" pull

echo ""
echo "=========================================="
echo "  Starting Services"
echo "=========================================="
echo ""

# Start services
$COMPOSE_CMD -f "$COMPOSE_FILE" up -d

# Wait for services to start
echo ""
echo "Waiting for services to start..."
sleep 5

# Check service status
echo ""
echo "Service Status:"
$COMPOSE_CMD -f "$COMPOSE_FILE" ps

echo ""
echo "=========================================="
echo -e "  ${GREEN}Startup Complete!${NC}"
echo "=========================================="
echo ""

# Get port configuration
CONSOLE_PORT=$(grep "^CONSOLE_PORT=" "$ENV_FILE" 2>/dev/null | cut -d '=' -f2)
CONSOLE_PORT=${CONSOLE_PORT:-3000}

# Check if Cloudflare Tunnel Token is configured
TUNNEL_TOKEN=$(grep "^CLOUDFLARE_TUNNEL_TOKEN=" "$ENV_FILE" 2>/dev/null | cut -d '=' -f2 | tr -d ' ')

echo "Access URLs:"
if [ -n "$TUNNEL_TOKEN" ] && [ "$TUNNEL_TOKEN" != "" ]; then
    echo "   Console (HTTPS): https://your-domain.com (configured in Cloudflare Dashboard)"
    echo "   Console (Local): http://localhost:$CONSOLE_PORT"
    echo ""
    echo -e "${GREEN}💡 Cloudflare Tunnel is enabled - your service is accessible via HTTPS!${NC}"
    echo "   Make sure you've configured the tunnel route in Cloudflare Dashboard"
else
    echo "   Console: http://localhost:$CONSOLE_PORT"
    echo ""
    echo -e "${BLUE}💡 To enable HTTPS remote access:${NC}"
    echo "   1. Get your Cloudflare Tunnel Token from: https://dash.cloudflare.com/"
    echo "   2. Add CLOUDFLARE_TUNNEL_TOKEN to .env file"
    echo "   3. Configure tunnel route in Cloudflare Dashboard"
    echo "   4. Restart services: $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml restart"
fi
echo ""

# Display different commands based on OS
case $OS_TYPE in
    windows)
        COMPOSE_CMD_DISPLAY="docker compose"
        ;;
    *)
        COMPOSE_CMD_DISPLAY="$COMPOSE_CMD"
        ;;
esac

echo "Management Commands:"
echo "   View logs:        $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml logs -f"
echo "   Stop services:    $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml down"
echo "   Restart services: $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml restart"
echo "   Update images:    $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml pull && $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml up -d"
echo "   Reset database:   $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml down -v"
echo ""

# Wait and show initial logs
sleep 5

if $COMPOSE_CMD -f "$COMPOSE_FILE" ps | grep -q "Up"; then
    echo -e "${GREEN}Services running normally${NC}"
    echo ""
    echo "Recent logs:"
    $COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=15 kimbap-console
else
    echo -e "${RED}Warning: Services may not have started properly, check logs${NC}"
    echo ""
    $COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=30
fi

echo ""
echo "=========================================="
echo "  Tips"
echo "=========================================="
echo ""
echo "Generated files (in current directory):"
echo "  - docker-compose.console.yml"
echo "  - .env"
echo ""
echo "To modify configuration:"
echo "  1. Edit .env file"
echo "  2. Run $COMPOSE_CMD_DISPLAY -f docker-compose.console.yml restart"
echo ""
echo "Script location: $0"
echo "=========================================="
