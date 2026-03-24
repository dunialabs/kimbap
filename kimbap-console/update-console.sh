#!/bin/bash

# KIMBAP Console Update Script
# Compatible with macOS, Linux, Windows (Git Bash/WSL)

set -e

echo "=========================================="
echo "  KIMBAP Console Update"
echo "=========================================="
echo ""

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    exit 1
fi

# Check docker-compose or docker compose command
if command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    echo -e "${RED}Error: docker-compose not found${NC}"
    exit 1
fi

# Get current working directory
WORK_DIR="$(pwd)"
COMPOSE_FILE="$WORK_DIR/docker-compose.console.yml"

# Check configuration file
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}Error: docker-compose.console.yml not found${NC}"
    echo "Please run start-console.sh first"
    exit 1
fi

echo "Pulling latest images..."
$COMPOSE_CMD -f "$COMPOSE_FILE" pull

echo ""
echo "Restarting services..."
$COMPOSE_CMD -f "$COMPOSE_FILE" up -d

echo ""
echo "Waiting for services to start..."
sleep 10

echo ""
echo "Service status:"
$COMPOSE_CMD -f "$COMPOSE_FILE" ps

echo ""
echo "Recent logs:"
$COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=20 kimbap-console

echo ""
echo -e "${GREEN}Update complete!${NC}"
echo ""
echo "To view full logs, run:"
echo "$COMPOSE_CMD -f docker-compose.console.yml logs -f"
