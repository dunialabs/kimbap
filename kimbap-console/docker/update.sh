#!/bin/bash

echo "🔄 Updating KIMBAP Console..."

# Pull latest image
echo "📥 Pulling latest image..."
docker compose pull kimbap-console

# Restart service
echo "🔄 Restarting service..."
docker compose up -d kimbap-console

# Clean up old images
echo "🧹 Cleaning up old images..."
docker image prune -f

echo "✅ Update complete!"
echo "🌐 Visit: http://localhost:3000"
