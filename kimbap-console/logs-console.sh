#!/bin/bash

# KIMBAP Console Logs Viewer Script
# Compatible with macOS, Linux, Windows (Git Bash/WSL)

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

echo "=========================================="
echo "  KIMBAP Console Logs"
echo "=========================================="
echo ""
echo "Press Ctrl+C to exit log view"
echo ""

# If parameter provided, view specific service logs
if [ -n "$1" ]; then
    $COMPOSE_CMD -f "$COMPOSE_FILE" logs -f "$1"
else
    # Default: view all service logs
    $COMPOSE_CMD -f "$COMPOSE_FILE" logs -f
fi
