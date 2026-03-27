#!/bin/bash

echo "========================================" 
echo "    Kimbap Console Docker Deployment Script"
echo "========================================"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    echo "   Mac: brew install docker"
    echo "   Linux: apt install docker.io"
    echo "   Windows: download Docker Desktop"
    exit 1
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker."
    exit 1
fi

echo "✅ Docker environment check passed"

# Build and start services
echo "🔨 Building and starting Kimbap Console..."

# Stop existing services (if any)
docker compose -f docker-compose.prod.yml down 2>/dev/null || true

# Build and start
docker compose -f docker-compose.prod.yml up --build -d

echo ""
echo "🎉 Deployment complete!"
echo ""

# Check if Cloudflare Tunnel Token is configured
if [ -n "$CLOUDFLARE_TUNNEL_TOKEN" ]; then
    echo "📱 Access URLs:"
    echo "   Console (HTTPS): https://your-domain.com (configured in Cloudflare Dashboard)"
    echo "   Console (Local): http://localhost:3000"
    echo "   API: http://localhost:3002"
    echo "   Database Admin: http://localhost:8080 (optional)"
    echo ""
    echo "💡 Cloudflare Tunnel is enabled - your service is accessible via HTTPS!"
    echo "   Make sure you've configured the tunnel route in Cloudflare Dashboard"
else
    echo "📱 Access URLs:"
    echo "   App: http://localhost:3000"
    echo "   API: http://localhost:3002"
    echo "   Database Admin: http://localhost:8080 (optional)"
    echo ""
    echo "💡 To enable HTTPS remote access:"
    echo "   1. Get your Cloudflare Tunnel Token from: https://dash.cloudflare.com/"
    echo "   2. Set CLOUDFLARE_TUNNEL_TOKEN environment variable"
    echo "   3. Configure tunnel route in Cloudflare Dashboard"
    echo "   4. Restart services: docker compose -f docker-compose.prod.yml restart"
fi

echo ""
echo "🔧 Management commands:"
echo "   View logs: docker compose -f docker-compose.prod.yml logs -f"
echo "   Stop services: docker compose -f docker-compose.prod.yml down"
echo "   Restart services: docker compose -f docker-compose.prod.yml restart"
echo ""
echo "⚠️  First start may take 1-2 minutes for database initialization"
