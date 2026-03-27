#!/bin/bash

# Docker 平台兼容性设置脚本
# 自动检测系统架构并设置合适的 Docker 配置

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检测系统架构
detect_architecture() {
    local arch=$(uname -m)
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    
    case $arch in
        x86_64|amd64)
            PLATFORM="linux/amd64"
            ARCH_NAME="x64"
            ;;
        arm64|aarch64)
            PLATFORM="linux/arm64"
            ARCH_NAME="arm64"
            ;;
        *)
            log_error "不支持的架构: $arch"
            exit 1
            ;;
    esac
    
    log_info "检测到系统: $os/$arch"
    log_info "Docker平台: $PLATFORM"
}

# 检查Docker是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker 未运行，请启动 Docker"
        exit 1
    fi
    
    log_success "Docker 检查通过"
}

# 设置 Docker Compose 平台
setup_docker_compose() {
    local compose_file="docker-compose.yml"
    local temp_file=$(mktemp)
    
    if [[ ! -f "$compose_file" ]]; then
        log_error "找不到 docker-compose.yml 文件"
        exit 1
    fi
    
    # 备份原文件
    cp "$compose_file" "${compose_file}.backup"
    
    # 根据架构更新配置
    if [[ "$PLATFORM" == "linux/amd64" ]]; then
        # AMD64 平台，添加平台指定
        sed 's|# platform: linux/amd64.*|platform: linux/amd64|g' "$compose_file" > "$temp_file"
        log_info "配置 Docker Compose 使用 AMD64 平台"
    else
        # ARM64 平台，移除平台限制让Docker自动选择
        sed 's|platform: linux/amd64|# platform: linux/amd64  # 让Docker自动选择合适的架构|g' "$compose_file" > "$temp_file"
        log_info "配置 Docker Compose 自动检测平台"
    fi
    
    mv "$temp_file" "$compose_file"
    log_success "Docker Compose 配置已更新"
}

# 构建镜像
build_image() {
    log_info "开始构建 Docker 镜像..."
    
    # 使用 buildx 构建适合当前平台的镜像
    if docker buildx build --platform "$PLATFORM" -t dunialabs/kimbap-console:latest . --load; then
        log_success "镜像构建成功"
    else
        log_error "镜像构建失败"
        exit 1
    fi
}

# 验证镜像
verify_image() {
    log_info "验证镜像架构..."
    
    local image_arch=$(docker inspect dunialabs/kimbap-console:latest --format='{{.Architecture}}')
    local expected_arch=$(echo "$PLATFORM" | cut -d'/' -f2)
    
    if [[ "$image_arch" == "$expected_arch" ]]; then
        log_success "镜像架构验证通过: $image_arch"
    else
        log_warning "镜像架构不匹配: 期望 $expected_arch, 实际 $image_arch"
    fi
}

# 主函数
main() {
    echo "🚀 Kimbap Console Docker 平台兼容性设置"
    echo "========================================="
    
    detect_architecture
    check_docker
    setup_docker_compose
    
    # 询问是否重新构建镜像
    read -p "是否重新构建镜像? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        build_image
        verify_image
    fi
    
    echo
    log_success "设置完成! 现在可以使用以下命令启动服务:"
    echo "  docker compose up -d kimbap-console"
    
    # 显示平台信息
    echo
    echo "📋 平台信息:"
    echo "  系统架构: $(uname -m)"
    echo "  Docker平台: $PLATFORM"
    echo "  镜像标签: dunialabs/kimbap-console:latest"
}

# 运行主函数
main "$@"