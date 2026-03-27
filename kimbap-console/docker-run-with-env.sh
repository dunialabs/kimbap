#!/bin/bash

# Docker Run Script with .env Support
# This script automatically loads .env file and adjusts DATABASE_URL for Docker

echo "🚀 Starting Kimbap Console with local .env configuration..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found in current directory"
    echo "Please create a .env file with DATABASE_URL and other configurations"
    exit 1
fi

# Load .env file and export variables
set -a
source .env
set +a

# Adjust host-local URLs for Docker
# Use host.docker.internal on all platforms (Docker 20.10+ supports it on Linux via host-gateway)
rewrite_localhost() {
    local value="$1"
    value="${value//localhost/host.docker.internal}"
    value="${value//127.0.0.1/host.docker.internal}"
    value="${value//\[::1\]/host.docker.internal}"
    echo "$value"
}

DOCKER_EXTRA_ARGS=""
if [[ "$OSTYPE" == "darwin"* ]] || [[ "$OSTYPE" == "msys" ]]; then
    DOCKER_DATABASE_URL="$(rewrite_localhost "$DATABASE_URL")"
else
    DOCKER_EXTRA_ARGS="--add-host=host.docker.internal:host-gateway"
    DOCKER_DATABASE_URL="$(rewrite_localhost "$DATABASE_URL")"
fi

DOCKER_KIMBAP_CORE_URL="$(rewrite_localhost "${KIMBAP_CORE_URL:-}")"
DOCKER_PROXY_ADMIN_URL="$(rewrite_localhost "${PROXY_ADMIN_URL:-}")"

# Set default values if not in .env
NODE_ENV="${NODE_ENV:-production}"
PORT="${PORT:-3000}"
BACKEND_PORT="${BACKEND_PORT:-3002}"

# Container name
CONTAINER_NAME="${CONTAINER_NAME:-kimbap-console}"

# Stop and remove existing container if exists
echo "🔄 Checking for existing container..."
if docker ps -a | grep -q "$CONTAINER_NAME"; then
    echo "   Stopping existing container..."
    docker stop "$CONTAINER_NAME" 2>/dev/null
    docker rm "$CONTAINER_NAME" 2>/dev/null
fi

# Run Docker container
echo "🐳 Starting Docker container..."
echo "   DATABASE_URL: ${DOCKER_DATABASE_URL%@*}@***" # Hide password in output
echo "   Frontend: http://localhost:$PORT"
echo "   Backend: http://localhost:$BACKEND_PORT"

docker run -d \
    --name "$CONTAINER_NAME" \
    -p "$PORT:3000" \
    -p "$BACKEND_PORT:3002" \
    ${DOCKER_EXTRA_ARGS} \
    -e DATABASE_URL="$DOCKER_DATABASE_URL" \
    -e NODE_ENV="$NODE_ENV" \
    -e KIMBAP_CORE_URL="$DOCKER_KIMBAP_CORE_URL" \
    -e PROXY_ADMIN_URL="$DOCKER_PROXY_ADMIN_URL" \
    -e PROXY_ADMIN_TOKEN="${PROXY_ADMIN_TOKEN:-}" \
    -e JWT_SECRET="${JWT_SECRET:-}" \
    -e LOG_SYNC_ENABLED="${LOG_SYNC_ENABLED:-true}" \
    dunialabs/kimbap-console:latest

# Check if container started successfully
if [ $? -eq 0 ]; then
    echo "✅ Container started successfully!"
    echo ""
    echo "📋 Useful commands:"
    echo "   View logs:    docker logs -f $CONTAINER_NAME"
    echo "   Stop:         docker stop $CONTAINER_NAME"
    echo "   Restart:      docker restart $CONTAINER_NAME"
    echo "   Remove:       docker stop $CONTAINER_NAME && docker rm $CONTAINER_NAME"
    echo ""
    
    # Wait a moment and check status
    sleep 5
    if docker ps | grep -q "$CONTAINER_NAME"; then
        echo "✅ Container is running"
        echo ""
        echo "🌐 Access the application:"
        echo "   Frontend: http://localhost:$PORT"
        echo "   Backend API: http://localhost:$BACKEND_PORT"
        echo ""
        
        # Show initial logs
        echo "📝 Initial logs:"
        docker logs --tail 10 "$CONTAINER_NAME"
    else
        echo "⚠️  Container stopped unexpectedly. Check logs:"
        docker logs "$CONTAINER_NAME"
    fi
else
    echo "❌ Failed to start container"
    exit 1
fi
