#!/bin/bash

# Kimbap Console Stop Script
# Compatible with macOS, Linux, Windows (Git Bash/WSL)

set -e

echo "=========================================="
echo "  Kimbap Console Stop"
echo "=========================================="
echo ""

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Ask for stop method
echo "Choose stop method:"
echo "  1) Stop services only (keep data)"
echo "  2) Stop services and delete all data (including database)"
echo ""
read -p "Enter option (1/2): " choice

case $choice in
    1)
        echo ""
        echo "Stopping services..."
        $COMPOSE_CMD -f "$COMPOSE_FILE" down
        echo -e "${GREEN}Services stopped (data preserved)${NC}"
        ;;
    2)
        echo ""
        echo -e "${YELLOW}Warning: This will delete all database data!${NC}"
        read -p "Confirm delete all data? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            echo ""
            echo "Stopping services and deleting data..."
            $COMPOSE_CMD -f "$COMPOSE_FILE" down -v
            echo -e "${GREEN}Services stopped, data deleted${NC}"
        else
            echo "Cancelled"
            exit 0
        fi
        ;;
    *)
        echo -e "${RED}Invalid option${NC}"
        exit 1
        ;;
esac

echo ""
echo "Current container status:"
$COMPOSE_CMD -f "$COMPOSE_FILE" ps
