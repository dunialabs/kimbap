#!/bin/bash

# KIMBAP Console - Docker Deployment with Auto Port Allocation
# This script automatically finds available ports and starts Docker services

set -e

echo "🐳 KIMBAP Console - Auto Port Deployment"
echo "======================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

print_error() {
    echo -e "${RED}❌${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ️${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

print_status "Docker is running"

# Check if Node.js is available for port detection
if ! command -v node > /dev/null 2>&1; then
    print_error "Node.js is required for port detection. Please install Node.js."
    exit 1
fi

print_status "Node.js is available"

# Check if .env file exists, create from example if not
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        print_info "Creating .env file from .env.example"
        cp .env.example .env
        print_warning "Please review and edit .env file with your configuration"
    else
        print_warning ".env file not found, continuing with default configuration"
    fi
fi

# Find available ports
print_info "Finding available ports..."
node scripts/find-available-ports.js

if [ $? -ne 0 ]; then
    print_error "Failed to find available ports"
    exit 1
fi

# Check if .env.ports was created
if [ ! -f .env.ports ]; then
    print_error "Port configuration file not created"
    exit 1
fi

print_status "Port allocation completed"

# Stop existing services if running
print_info "Stopping existing services (if any)..."
docker compose --env-file .env.ports down --remove-orphans > /dev/null 2>&1 || true

# Pull latest images
print_info "Pulling latest Docker images..."
docker compose --env-file .env.ports pull

# Start services with auto-allocated ports
print_info "Starting KIMBAP Console services..."
docker compose --env-file .env.ports up -d

if [ $? -ne 0 ]; then
    print_error "Failed to start Docker services"
    exit 1
fi

# Wait for services to initialize
print_info "Waiting for services to initialize..."
sleep 10

# Check service status
print_info "Checking service status..."
docker compose --env-file .env.ports ps

# Read port configuration and display URLs
if command -v node > /dev/null 2>&1; then
    echo ""
    echo "🎉 KIMBAP Console is now running!"
    echo "==============================="
    
    # Parse port configuration and display URLs
    FRONTEND_PORT=$(grep "KIMBAP_FRONTEND_PORT=" .env.ports | cut -d'=' -f2)
    BACKEND_PORT=$(grep "KIMBAP_BACKEND_PORT=" .env.ports | cut -d'=' -f2)
    ADMINER_PORT=$(grep "KIMBAP_ADMINER_PORT=" .env.ports | cut -d'=' -f2)
    
    echo ""
    echo "🌐 Service URLs:"
    echo "   • Frontend:      http://localhost:${FRONTEND_PORT:-3000}"
    echo "   • Backend API:   http://localhost:${BACKEND_PORT:-3002}"
    echo "   • Database Admin: http://localhost:${ADMINER_PORT:-8080}"
    echo ""
    echo "📋 Management Commands:"
    echo "   • View logs:     docker compose --env-file .env.ports logs -f"
    echo "   • Stop services: docker compose --env-file .env.ports down"
    echo "   • Restart:       docker compose --env-file .env.ports restart"
    echo "   • Status:        docker compose --env-file .env.ports ps"
    echo ""
    echo "📁 Configuration:"
    echo "   • Port config:   .env.ports"
    echo "   • JSON config:   .port-config.json"
    echo ""
fi

print_status "Deployment completed successfully!"

# Optionally open browser
if command -v open > /dev/null 2>&1 && [ -n "$FRONTEND_PORT" ]; then
    echo ""
    read -p "📱 Open application in browser? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        open "http://localhost:${FRONTEND_PORT}"
    fi
elif command -v xdg-open > /dev/null 2>&1 && [ -n "$FRONTEND_PORT" ]; then
    echo ""
    read -p "📱 Open application in browser? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        xdg-open "http://localhost:${FRONTEND_PORT}"
    fi
fi