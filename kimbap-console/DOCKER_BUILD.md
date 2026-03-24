# KIMBAP Console - Docker 构建指南

本文档说明如何构建和推送 KIMBAP Console 的 Docker 镜像。

## 前置要求

- Docker Desktop 已安装并运行
- Docker Buildx 已启用（Docker Desktop 默认包含）
- Docker Hub 账号已登录

## 构建架构

KIMBAP Console 支持多架构构建：
- `linux/amd64` - Intel/AMD 服务器
- `linux/arm64` - ARM 服务器（Apple Silicon 等）

## 快速开始

### 1. 登录 Docker Hub

```bash
docker login
```

### 2. 构建并推送镜像

```bash
cd /path/to/kimbap-console

# 构建多架构镜像并推送到 Docker Hub
docker buildx build --platform linux/amd64,linux/arm64 -t kimbapio/kimbap-console:latest --push .
```

## 构建详解

### Dockerfile 结构

镜像采用多阶段构建：

1. **base** - 基础环境（Node.js 20 Alpine + PostgreSQL 客户端）
2. **builder** - 构建阶段（安装依赖、生成 Prisma Client、构建 Next.js）
3. **production** - 生产环境（复制构建产物、安装生产依赖）

### 包含的组件

镜像包含以下组件：
- Next.js 前端应用（standalone 模式）
- API 路由（/app/api/*）
- Prisma ORM 和数据库脚本
- **日志同步脚本**（/app/jobs/log-sync.js）- 自动从 kimbap-core 同步日志

### 启动流程

容器启动时（docker-entrypoint.sh）按以下顺序执行：

1. 配置数据库连接
2. 运行数据库迁移（Prisma migrate deploy）
3. **启动日志同步任务**（后台运行，每 2 分钟同步一次）
4. 启动 Next.js 前端服务器（端口 3000）

## 高级用法

### 构建特定架构

```bash
# 仅构建 AMD64
docker buildx build --platform linux/amd64 -t kimbapio/kimbap-console:latest --push .

# 仅构建 ARM64
docker buildx build --platform linux/arm64 -t kimbapio/kimbap-console:latest --push .
```

### 本地构建（不推送）

```bash
# 构建到本地 Docker
docker buildx build --platform linux/amd64 -t kimbapio/kimbap-console:latest --load .

# 查看本地镜像
docker images | grep kimbap-console
```

### 后台构建并查看日志

```bash
# 启动后台构建
nohup docker buildx build --platform linux/amd64,linux/arm64 -t kimbapio/kimbap-console:latest --push . > /tmp/build-console-$(date +%s).log 2>&1 &

# 查看构建进度
tail -f /tmp/build-console-*.log
```

### 使用自定义标签

```bash
# 构建带版本号的镜像
docker buildx build --platform linux/amd64,linux/arm64 \
  -t kimbapio/kimbap-console:v1.0.0 \
  -t kimbapio/kimbap-console:latest \
  --push .
```

## 验证构建

### 检查镜像详情

```bash
# 查看镜像支持的架构
docker buildx imagetools inspect kimbapio/kimbap-console:latest
```

输出示例：
```
Name:      docker.io/kimbapio/kimbap-console:latest
MediaType: application/vnd.oci.image.index.v1+json
Digest:    sha256:9eb7d8464cada3702d158efc3596ca95047e6f19c23d7242f09fcd049e6b5964

Manifests:
  Name:      docker.io/kimbapio/kimbap-console:latest@sha256:932fb3a...
  Platform:  linux/amd64

  Name:      docker.io/kimbapio/kimbap-console:latest@sha256:425f4a1...
  Platform:  linux/arm64
```

### 本地测试运行

```bash
# 拉取镜像
docker pull kimbapio/kimbap-console:latest

# 运行容器（需要配置环境变量）
docker run -d \
  --name kimbap-console-test \
  -p 3000:3000 \
  -e DATABASE_URL="postgresql://user:password@host:5432/dbname" \
  -e MCP_GATEWAY_URL="http://kimbap-core:3002" \
  kimbapio/kimbap-console:latest

# 查看日志
docker logs -f kimbap-console-test
```

## 环境变量配置

镜像启动时需要以下环境变量：

### 必需变量

- `DATABASE_URL` - PostgreSQL 连接字符串

### 可选变量

- `MCP_GATEWAY_URL` - KIMBAP Core 网关地址（默认：`http://localhost:3002`）
- `PROXY_ADMIN_URL` - KIMBAP Core 管理接口（默认：`http://localhost:3002/admin`）
- `LOG_SYNC_ENABLED` - 是否启用日志同步（默认：`true`）
- `LOG_SYNC_INTERVAL_MINUTES` - 日志同步间隔（分钟，默认：2）
- `PORT` - 前端服务端口（默认：3000）

## 日志同步功能

镜像内置日志同步脚本，会自动：

1. 从数据库读取 KIMBAP Core 配置
2. 获取 owner 用户的 access token
3. 调用 KIMBAP Core API 获取日志
4. 增量同步日志到本地数据库
5. 每 2 分钟自动执行一次

可通过环境变量控制：
```bash
-e LOG_SYNC_ENABLED=false          # 禁用日志同步
-e LOG_SYNC_INTERVAL_MINUTES=5     # 修改同步间隔为 5 分钟
-e MAX_LOGS_PER_REQUEST=1000       # 每次最多获取 1000 条日志
```

## 常见问题

### 构建超时

如果构建过程中遇到网络超时，可以：
1. 检查网络连接
2. 重试构建（Docker 会使用缓存加速）
3. 使用后台构建方式

### 磁盘空间不足

```bash
# 清理未使用的镜像和构建缓存
docker system prune -a

# 查看磁盘使用情况
docker system df
```

### Buildx 构建器问题

```bash
# 重新创建构建器
docker buildx rm multiarch-builder
docker buildx create --name multiarch-builder --driver docker-container --use
docker buildx inspect --bootstrap
```

## 生产部署

完整的生产部署指南请参考：
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 完整部署指南
- [CONSOLE_ONLY_DEPLOYMENT.md](../CONSOLE_ONLY_DEPLOYMENT.md) - 单独部署 Console

## 相关文档

- [Dockerfile](./Dockerfile) - Docker 镜像定义
- [docker-entrypoint.sh](./docker/docker-entrypoint.sh) - 容器启动脚本
- [docker-compose.yml](../docker-compose.yml) - Docker Compose 配置
