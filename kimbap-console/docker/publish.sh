#!/bin/bash
###
 # @Author: xudada 1820064201@qq.com
 # @Date: 2025-09-02 11:16:04
 # @LastEditors: xudada 1820064201@qq.com
 # @LastEditTime: 2025-09-02 11:21:48
 # @FilePath: /kimbap-console/docker/publish.sh
 # @Description: Default settings placeholder. Configure `customMade` via koroFileHeader: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
###

echo "========================================"
echo "    Publish Kimbap Console to Docker Hub"
echo "========================================"

# Check whether Docker Hub login is active (more reliable approach)
if ! docker system info 2>/dev/null | grep -q "Registry" || ! docker pull hello-world >/dev/null 2>&1; then
    echo "⚠️  Please log in to Docker Hub first:"
    echo "   docker login"
    echo ""
    echo "If already logged in, you can skip this check:"
    echo "   SKIP_LOGIN_CHECK=1 npm run docker:publish"

    if [ "$SKIP_LOGIN_CHECK" != "1" ]; then
        exit 1
    fi
fi

# Set image information
# Use environment variables or defaults
DOCKER_USERNAME=${DOCKER_USERNAME:-"dunialabs"}
IMAGE_NAME="$DOCKER_USERNAME/kimbap-console"
VERSION=${1:-"latest"}
FULL_TAG="$IMAGE_NAME:$VERSION"

echo "🏷️  Building image: $FULL_TAG"

# Build image
docker build -t $FULL_TAG .

if [ $? -ne 0 ]; then
    echo "❌ Image build failed"
    exit 1
fi

echo "✅ Image build succeeded"

# Push to Docker Hub
echo "📤 Pushing image to Docker Hub..."
docker push $FULL_TAG

if [ $? -eq 0 ]; then
    echo ""
    echo "🎉 Publish succeeded!"
    echo ""
    echo "📋 Usage command for users:"
    echo "   docker run -d -p 3000:3000 -p 3002:3002 \\"
    echo "     -e DB_HOST=your-db-host \\"
    echo "     -e DB_USER=kimbap \\"
    echo "     -e DB_PASSWORD=kimbap123 \\"
    echo "     -e DB_NAME=kimbap_db \\"
    echo "     $FULL_TAG"
    echo ""
    echo "Or use docker-compose:"
    echo "   docker compose -f docker-compose.prod.yml up -d"
else
    echo "❌ Push failed"
    exit 1
fi
