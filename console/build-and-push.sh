#!/bin/bash
###
 # @Author: xudada 1820064201@qq.com
 # @Date: 2025-12-03 16:54:45
 # @LastEditors: xudada 1820064201@qq.com
 # @LastEditTime: 2025-12-05 00:02:00
 # @FilePath: /kimbap-console/build-and-push.sh
 # @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
###

# Kimbap Console Docker 镜像构建和推送脚本
# 使用方法：./build-and-push.sh
#
# 功能：
# - 自动从 package.json 读取版本号
# - 同时推送多个标签（版本号、主次版本、主版本、latest）
# - 支持 linux/amd64 和 linux/arm64 平台

set -e

# 配置
IMAGE_NAME="dunialabs/kimbap-console"
PLATFORMS="linux/amd64,linux/arm64"

# 从 package.json 读取版本号
echo "读取版本信息..."
VERSION=$(node -p "require('./package.json').version")

if [ -z "$VERSION" ]; then
    echo "❌ 错误: 无法读取 package.json 中的版本号"
    exit 1
fi

# 解析版本号 (例如: 1.0.4 -> MAJOR=1, MINOR=0, PATCH=4)
MAJOR=$(echo $VERSION | cut -d. -f1)
MINOR=$(echo $VERSION | cut -d. -f2)
MAJOR_MINOR="$MAJOR.$MINOR"

# 构建标签列表
TAGS="-t $IMAGE_NAME:$VERSION -t $IMAGE_NAME:$MAJOR_MINOR -t $IMAGE_NAME:$MAJOR -t $IMAGE_NAME:latest"

echo "======================================"
echo "  Kimbap Console 镜像构建"
echo "======================================"
echo "版本号: $VERSION"
echo "镜像名称: $IMAGE_NAME"
echo "标签列表:"
echo "  - $IMAGE_NAME:$VERSION (完整版本)"
echo "  - $IMAGE_NAME:$MAJOR_MINOR (主次版本)"
echo "  - $IMAGE_NAME:$MAJOR (主版本)"
echo "  - $IMAGE_NAME:latest (最新版本)"
echo "支持平台: $PLATFORMS"
echo "======================================"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker 未运行，请先启动 Docker"
    exit 1
fi

echo "✓ Docker 运行正常"
echo ""

# 检查 buildx 是否可用
if ! docker buildx version > /dev/null 2>&1; then
    echo "❌ 错误: docker buildx 不可用"
    exit 1
fi

echo "✓ Docker Buildx 可用"
echo ""

# 检查 Docker 登录状态
echo "检查 Docker 登录状态..."
# 检查是否有 Docker Hub 的认证信息（检查 ~/.docker/config.json）
if [ ! -f ~/.docker/config.json ] || ! grep -q '"auths"' ~/.docker/config.json 2>/dev/null; then
    echo "⚠️  警告: 未检测到 Docker 登录信息"
    echo ""
    echo "请先登录 Docker Hub:"
    echo "  docker login"
    echo ""
    echo "或者登录到其他注册表:"
    echo "  docker login <registry-url>"
    echo ""
    read -p "是否现在登录? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker login
        if [ $? -ne 0 ]; then
            echo "❌ 错误: Docker 登录失败"
            exit 1
        fi
    else
        echo "❌ 错误: 需要先登录 Docker 才能推送镜像"
        exit 1
    fi
else
    echo "✓ 检测到 Docker 认证配置"
    echo "  提示: 如果推送失败，请运行 'docker login' 重新登录"
fi

echo ""

# 创建或使用 multiarch builder
# 优先使用 kimbap-multiarch-builder（与 kimbap 项目共用）
if docker buildx ls | grep -q "^kimbap-multiarch-builder"; then
    echo "使用已存在的 kimbap-multiarch-builder（与其他项目共用）"
    docker buildx use kimbap-multiarch-builder
elif docker buildx ls | grep -q "^multiarch-builder "; then
    echo "使用已存在的 multiarch-builder"
    docker buildx use multiarch-builder
else
    echo "创建 kimbap-multiarch-builder..."
    docker buildx create --name kimbap-multiarch-builder --driver docker-container --use
    docker buildx inspect --bootstrap
fi

echo ""
echo "开始构建和推送镜像..."
echo ""

# 构建并推送
if docker buildx build \
    --platform "$PLATFORMS" \
    $TAGS \
    --push \
    .; then
    echo ""
    echo "======================================"
    echo "✅ 构建成功！"
    echo "======================================"
    echo "已推送的镜像标签："
    echo "  - $IMAGE_NAME:$VERSION"
    echo "  - $IMAGE_NAME:$MAJOR_MINOR"
    echo "  - $IMAGE_NAME:$MAJOR"
    echo "  - $IMAGE_NAME:latest"
    echo ""
    echo "验证镜像："
    docker buildx imagetools inspect "$IMAGE_NAME:$VERSION"
    echo ""
    echo "用户可以使用以下任意标签拉取："
    echo "  docker pull $IMAGE_NAME:$VERSION    # 精确版本"
    echo "  docker pull $IMAGE_NAME:$MAJOR_MINOR      # $MAJOR_MINOR.x 系列最新"
    echo "  docker pull $IMAGE_NAME:$MAJOR            # $MAJOR.x.x 系列最新"
    echo "  docker pull $IMAGE_NAME:latest     # 总是最新版本"
else
    echo ""
    echo "❌ 构建或推送失败"
    echo ""
    echo "可能的原因："
    echo "  1. Docker 未登录或登录已过期 - 运行 'docker login'"
    echo "  2. 没有推送到 $IMAGE_NAME 的权限"
    echo "  3. 仓库 $IMAGE_NAME 不存在于注册表中"
    echo "  4. 网络连接问题"
    echo ""
    echo "解决步骤："
    echo "  1. 确认已登录: docker login"
    echo "  2. 确认有权限推送到 $IMAGE_NAME"
    echo "  3. 确认仓库已创建（如果需要）"
    exit 1
fi
