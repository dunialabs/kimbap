#!/bin/bash

# Kimbap Console 本地 Docker 镜像构建脚本（不推送）
# 使用方法：./build-local.sh

set -e

# 配置
IMAGE_NAME="kimbapio/kimbap-console"
TAG="${1:-latest}"
PLATFORM="${2:-linux/amd64}"

echo "======================================"
echo "  Kimbap Console 本地镜像构建"
echo "======================================"
echo "镜像名称: $IMAGE_NAME:$TAG"
echo "构建平台: $PLATFORM"
echo "======================================"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker 未运行，请先启动 Docker"
    exit 1
fi

echo "✓ Docker 运行正常"
echo ""
echo "开始构建本地镜像..."
echo ""

# 构建本地镜像
docker buildx build \
    --platform "$PLATFORM" \
    -t "$IMAGE_NAME:$TAG" \
    --load \
    .

if [ $? -eq 0 ]; then
    echo ""
    echo "======================================"
    echo "✅ 本地构建成功！"
    echo "======================================"
    echo "镜像: $IMAGE_NAME:$TAG"
    echo ""
    echo "查看镜像："
    docker images | grep kimbap-console | head -5
else
    echo ""
    echo "❌ 构建失败"
    exit 1
fi
