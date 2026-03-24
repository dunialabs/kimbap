#!/bin/bash

# KIMBAP Console Restart Script
# Compatible with macOS, Linux, Windows (Git Bash/WSL)

set -e

echo "=========================================="
echo "  KIMBAP Console Restart"
echo "=========================================="
echo ""

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

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

echo "Restarting services..."
$COMPOSE_CMD -f "$COMPOSE_FILE" restart

echo ""
echo "Waiting for services to start..."
sleep 5

echo ""
echo "Service status:"
$COMPOSE_CMD -f "$COMPOSE_FILE" ps

echo ""
echo -e "${GREEN}Restart complete!${NC}"
echo ""
echo "View logs:"
echo "$COMPOSE_CMD -f docker-compose.console.yml logs -f"
