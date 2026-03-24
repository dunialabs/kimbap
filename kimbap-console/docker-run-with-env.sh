#!/bin/bash

# Docker Run Script with .env Support
# This script automatically loads .env file and adjusts DATABASE_URL for Docker

echo "🚀 Starting KIMBAP Console with local .env configuration..."

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

# Adjust DATABASE_URL for Docker
# Replace localhost with host.docker.internal for Mac/Windows
# For Linux, use host network or actual IP
if [[ "$OSTYPE" == "darwin"* ]] || [[ "$OSTYPE" == "msys" ]]; then
    # macOS or Windows
    DOCKER_DATABASE_URL="${DATABASE_URL//localhost/host.docker.internal}"
    DOCKER_DATABASE_URL="${DOCKER_DATABASE_URL//127.0.0.1/host.docker.internal}"
else
    # Linux - use host network mode
    DOCKER_NETWORK="--network host"
    DOCKER_DATABASE_URL="$DATABASE_URL"
fi

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
    ${DOCKER_NETWORK} \
    -e DATABASE_URL="$DOCKER_DATABASE_URL" \
    -e NODE_ENV="$NODE_ENV" \
    -e PROXY_ADMIN_URL="$PROXY_ADMIN_URL" \
    -e PROXY_ADMIN_TOKEN="$PROXY_ADMIN_TOKEN" \
    kimbapio/kimbap-console:latest

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