# KIMBAP 部署使用指南

欢迎使用 KIMBAP - 零知识加密的 AI 工具连接平台！本指南将帮助您快速部署和使用 KIMBAP 系统。

## 📋 目录

- [系统介绍](#系统介绍)
- [快速开始](#快速开始)
- [环境要求](#环境要求)
- [部署步骤](#部署步骤)
- [配置说明](#配置说明)
- [使用指南](#使用指南)
- [常见问题](#常见问题)
- [故障排查](#故障排查)

## 🎯 系统介绍

KIMBAP 是一个零知识加密的 Model Context Protocol (MCP) 管理平台，包含两个核心组件：

- **KIMBAP Console**: Web 管理控制台，提供可视化的用户界面
- **KIMBAP Core**: MCP Gateway 后端服务，支持 Socket.IO 实时通信

### 主要功能

- ✅ MCP 服务器管理和配置
- ✅ 用户权限和访问控制
- ✅ 实时日志和监控
- ✅ OAuth 2.0 认证支持
- ✅ IP 白名单管理
- ✅ 零知识加密通信

## 🚀 快速开始

### 一键部署

**步骤 1: 创建部署目录**

```bash
mkdir kimbap-deployment
cd kimbap-deployment
```

**步骤 2: 创建 docker-compose.yml 文件**

复制以下内容到 `docker-compose.yml` 文件：

```yaml
version: '3.8'

services:
  # PostgreSQL for kimbap-console
  postgres-console:
    image: postgres:16-alpine
    container_name: kimbap-console-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: kimbap
      POSTGRES_PASSWORD: kimbap123
      POSTGRES_DB: kimbap_console
    volumes:
      - postgres-console-data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U kimbap -d kimbap_console']
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kimbap-network

  # PostgreSQL for kimbap-core
  postgres-core:
    image: postgres:16-alpine
    container_name: kimbap-core-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: kimbap
      POSTGRES_PASSWORD: kimbap123
      POSTGRES_DB: kimbap_mcp_gateway
    volumes:
      - postgres-core-data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U kimbap -d kimbap_mcp_gateway']
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kimbap-network

  # Kimbap Core Service (MCP Gateway)
  kimbap-core:
    image: kimbapio/kimbap-core:latest
    container_name: kimbap-core
    restart: unless-stopped
    depends_on:
      postgres-core:
        condition: service_healthy
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://kimbap:kimbap123@postgres-core:5432/kimbap_mcp_gateway?schema=public
      BACKEND_PORT: 3002
      JWT_SECRET: your-secret-key-change-in-production
      SKIP_CLOUDFLARED: true
    ports:
      - '3002:3002'
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:3002/health']
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s

  # Kimbap Console Service
  kimbap-console:
    image: kimbapio/kimbap-console:latest
    container_name: kimbap-console
    restart: unless-stopped
    depends_on:
      postgres-console:
        condition: service_healthy
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://kimbap:kimbap123@postgres-console:5432/kimbap_console?schema=public
      PORT: 3000
      JWT_SECRET: your-secret-key-change-in-production
      MCP_GATEWAY_URL: http://kimbap-core:3002
      PROXY_ADMIN_URL: http://kimbap-core:3002/admin
    ports:
      - '3000:3000'
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:3000']
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s

volumes:
  postgres-console-data:
    driver: local
  postgres-core-data:
    driver: local

networks:
  kimbap-network:
    driver: bridge
```

**步骤 3: 创建 .env 文件（可选）**

如果需要自定义配置，创建 `.env` 文件并复制以下内容：

```bash
# ====================================
# KIMBAP Docker 部署环境变量配置
# ====================================

# -------------------- 服务端口配置 --------------------
CONSOLE_PORT=3000
CORE_PORT=3002

# -------------------- JWT 密钥配置 --------------------
# ⚠️ 生产环境请务必修改为强密码（至少 32 字符）
CONSOLE_JWT_SECRET=your-console-jwt-secret-change-in-production-min-32-chars
CORE_JWT_SECRET=your-core-jwt-secret-change-in-production-min-32-chars

# -------------------- 服务地址配置 --------------------
# 如果需要公网访问，修改为实际的域名或 IP
MCP_GATEWAY_URL=http://kimbap-core:3002
PROXY_ADMIN_URL=http://kimbap-core:3002/admin

# -------------------- 其他配置 --------------------
NODE_ENV=production
```

**步骤 4: 启动服务**

```bash
docker compose up -d
```

**步骤 5: 访问服务**

- Web 控制台: http://localhost:3000
- API 服务: http://localhost:3002

## 💻 环境要求

### 系统要求

- **操作系统**: Linux / macOS / Windows (支持 Docker)
- **CPU**: 2 核心或以上
- **内存**: 4GB RAM 或以上
- **磁盘**: 10GB 可用空间

### 软件要求

- **Docker**: 20.10 或更高版本
- **Docker Compose**: 2.0 或更高版本

### 端口要求

确保以下端口未被占用：

- `3000` - KIMBAP Console 前端
- `3002` - KIMBAP Core API

## 📦 部署步骤

### 方式一：使用 Docker Compose（推荐）

#### 1. 创建项目目录

```bash
mkdir kimbap-deployment
cd kimbap-deployment
```

#### 2. 创建 docker-compose.yml 文件

```yaml
version: '3.8'

services:
  # PostgreSQL for KIMBAP Console
  postgres-console:
    image: postgres:16-alpine
    container_name: kimbap-console-postgres
    environment:
      POSTGRES_USER: ${CONSOLE_DB_USER:-kimbap}
      POSTGRES_PASSWORD: ${CONSOLE_DB_PASSWORD:-kimbap123}
      POSTGRES_DB: ${CONSOLE_DB_NAME:-kimbap_console}
    ports:
      - '${CONSOLE_DB_PORT:-5432}:5432'
    volumes:
      - postgres-console-data:/var/lib/postgresql/data
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U ${CONSOLE_DB_USER:-kimbap}']
      interval: 5s
      timeout: 5s
      retries: 5

  # PostgreSQL for KIMBAP Core
  postgres-core:
    image: postgres:16-alpine
    container_name: kimbap-core-postgres
    environment:
      POSTGRES_USER: ${CORE_DB_USER:-kimbap}
      POSTGRES_PASSWORD: ${CORE_DB_PASSWORD:-kimbap123}
      POSTGRES_DB: ${CORE_DB_NAME:-kimbap_mcp_gateway}
    ports:
      - '${CORE_DB_PORT:-5433}:5432'
    volumes:
      - postgres-core-data:/var/lib/postgresql/data
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U ${CORE_DB_USER:-kimbap}']
      interval: 5s
      timeout: 5s
      retries: 5

  # KIMBAP Core (MCP Gateway)
  kimbap-core:
    image: kimbapio/kimbap-core:latest
    container_name: kimbap-core
    depends_on:
      postgres-core:
        condition: service_healthy
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://${CORE_DB_USER:-kimbap}:${CORE_DB_PASSWORD:-kimbap123}@postgres-core:5432/${CORE_DB_NAME:-kimbap_mcp_gateway}?schema=public
      PORT: ${CORE_PORT:-3002}
      JWT_SECRET: ${CORE_JWT_SECRET:-your-core-jwt-secret-change-in-production}
    ports:
      - '${CORE_PORT:-3002}:3002'
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:3002/health']
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # KIMBAP Console
  kimbap-console:
    image: kimbapio/kimbap-console:latest
    container_name: kimbap-console
    depends_on:
      postgres-console:
        condition: service_healthy
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://${CONSOLE_DB_USER:-kimbap}:${CONSOLE_DB_PASSWORD:-kimbap123}@postgres-console:5432/${CONSOLE_DB_NAME:-kimbap_console}?schema=public
      PORT: ${CONSOLE_PORT:-3000}
      JWT_SECRET: ${CONSOLE_JWT_SECRET:-your-secret-key-change-in-production}
      MCP_GATEWAY_URL: ${MCP_GATEWAY_URL:-http://kimbap-core:3002}
      PROXY_ADMIN_URL: ${PROXY_ADMIN_URL:-http://kimbap-core:3002/admin}
    ports:
      - '${CONSOLE_PORT:-3000}:3000'
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:3000']
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

networks:
  kimbap-network:
    driver: bridge

volumes:
  postgres-console-data:
  postgres-core-data:
```

#### 3. 创建 .env 文件

```bash
# ====================================
# KIMBAP Docker 部署环境变量配置
# ====================================

# -------------------- 服务端口配置 --------------------
CONSOLE_PORT=3000
CORE_PORT=3002

# -------------------- JWT 密钥配置 --------------------
# ⚠️ 生产环境请务必修改为强密码（至少 32 字符）
CONSOLE_JWT_SECRET=your-console-jwt-secret-change-in-production-min-32-chars
CORE_JWT_SECRET=your-core-jwt-secret-change-in-production-min-32-chars

# -------------------- 服务地址配置 --------------------
# 如果需要公网访问，修改为实际的域名或 IP
MCP_GATEWAY_URL=http://kimbap-core:3002
PROXY_ADMIN_URL=http://kimbap-core:3002/admin

# -------------------- 其他配置 --------------------
NODE_ENV=production
```

#### 4. 启动服务

```bash
# 启动所有服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f
```

#### 5. 访问服务

- **Web 控制台**: http://localhost:3000
- **API 文档**: http://localhost:3002/health

### 方式二：手动拉取镜像

```bash
# 拉取镜像
docker pull kimbapio/kimbap-console:latest
docker pull kimbapio/kimbap-core:latest
docker pull postgres:16-alpine

# 创建网络
docker network create kimbap-network

# 启动数据库
docker run -d --name kimbap-console-postgres \
  --network kimbap-network \
  -e POSTGRES_USER=kimbap \
  -e POSTGRES_PASSWORD=kimbap123 \
  -e POSTGRES_DB=kimbap_console \
  -p 5432:5432 \
  postgres:16-alpine

docker run -d --name kimbap-core-postgres \
  --network kimbap-network \
  -e POSTGRES_USER=kimbap \
  -e POSTGRES_PASSWORD=kimbap123 \
  -e POSTGRES_DB=kimbap_mcp_gateway \
  -p 5433:5432 \
  postgres:16-alpine

# 启动 KIMBAP Core
docker run -d --name kimbap-core \
  --network kimbap-network \
  -e DATABASE_URL="postgresql://kimbap:kimbap123@kimbap-core-postgres:5432/kimbap_mcp_gateway?schema=public" \
  -e JWT_SECRET="your-secret-key" \
  -p 3002:3002 \
  kimbapio/kimbap-core:latest

# 启动 KIMBAP Console
docker run -d --name kimbap-console \
  --network kimbap-network \
  -e DATABASE_URL="postgresql://kimbap:kimbap123@kimbap-console-postgres:5432/kimbap_console?schema=public" \
  -e MCP_GATEWAY_URL="http://kimbap-core:3002" \
  -p 3000:3000 \
  kimbapio/kimbap-console:latest
```

## ⚙️ 配置说明

### 必须修改的配置（生产环境）

```bash
# JWT 密钥（至少 32 字符的随机字符串）
CONSOLE_JWT_SECRET=$(openssl rand -base64 32)
CORE_JWT_SECRET=$(openssl rand -base64 32)
```

### 可选配置

#### 端口修改

如果默认端口被占用，可以在 `.env` 文件中修改端口：

```bash
# 修改服务端口
CONSOLE_PORT=4000        # Console Web 界面端口（默认 3000）
CORE_PORT=4002          # Core API 端口（默认 3002）
```

**启动服务：**

```bash
# 1. 修改 .env 文件中的端口配置
vim .env

# 2. 重启服务使配置生效
docker compose down
docker compose up -d

# 3. 使用新端口访问服务
# Console: http://localhost:4000
# Core API: http://localhost:4002
```

**示例：使用自定义端口部署**

```bash
# .env 文件内容示例
CONSOLE_PORT=8080
CORE_PORT=8082

# 启动后访问：
# - Web 控制台: http://localhost:8080
# - API 服务: http://localhost:8082
```

**注意事项：**
1. 如果使用反向代理（Nginx/Caddy），需要同步更新代理配置
3. 确保新端口未被其他程序占用：
   ```bash
   # 检查端口是否可用
   lsof -i :8080
   lsof -i :8082
   ```

#### 公网访问配置

如果需要通过公网访问：

```bash
# 使用您的域名
MCP_GATEWAY_URL=https://api.yourdomain.com
```

建议配合 Nginx 或 Caddy 等反向代理使用：

```nginx
# Nginx 配置示例
server {
    listen 80;
    server_name console.yourdomain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:3002;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # WebSocket 支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## 📖 使用指南

### 首次登录

1. 访问 http://localhost:3000
2. 首次使用需要创建管理员账户
3. 设置主密码（Master Password）用于加密敏感数据

### 创建 MCP 服务器

1. 登录控制台后，点击「创建服务器」
2. 填写服务器信息：
   - 服务器名称
   - 启动命令
   - 工具配置
3. 保存后即可启动服务器

### 用户管理

1. 进入「成员管理」页面
2. 可以添加用户、分配权限
3. 支持三种角色：
   - Owner（所有者）
   - Admin（管理员）
   - Member（成员）

### API 使用

KIMBAP 提供完整的 REST API 和 Socket.IO 接口：

```bash
# 健康检查
curl http://localhost:3002/health

# 获取服务器列表（需要认证）
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:3002/api/servers
```

详细 API 文档请访问：http://localhost:3002/api-docs

## ❓ 常见问题

### Q1: 服务启动失败？

**A**: 检查端口是否被占用：

```bash
# 检查端口占用
lsof -i :3000
lsof -i :3002

# 停止占用端口的进程
kill -9 <PID>
```

### Q2: 如何更新到最新版本？

**A**: 拉取最新镜像并重启：

```bash
# 拉取最新镜像
docker compose pull

# 重启服务
docker compose up -d

# 查看是否成功更新
docker compose ps
```

## 🔧 故障排查

### 查看日志

```bash
# 查看所有服务日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f kimbap-console
docker compose logs -f kimbap-core

# 查看最近 100 行日志
docker compose logs --tail 100 kimbap-console
```

### 重启服务

```bash
# 重启所有服务
docker compose restart

# 重启特定服务
docker compose restart kimbap-console
docker compose restart kimbap-core
```

### 完全重置

```bash
# ⚠️ 警告：这将删除所有数据！
docker compose down -v
docker compose up -d
```

### 健康检查

```bash
# 检查服务健康状态
curl http://localhost:3002/health

# 应该返回：
{
  "status": "healthy",
  "uptime": 12345.67,
  "sessions": {
    "active": 0,
    "total": 0
  },
  "socketio": {
    "onlineUsers": 0,
    "totalConnections": 0
  }
}
```


## 📊 监控和维护

### 资源监控

```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
docker system df

# 清理未使用的资源
docker system prune -a
```

### 定期维护

建议每周执行：

```bash
# 1. 备份数据（见上方备份说明）
# 2. 检查日志是否有异常
docker compose logs --since 7d | grep -i error
# 3. 更新镜像
docker compose pull
docker compose up -d
```

## 🔐 安全建议

1. **修改默认密码**: 生产环境务必修改所有默认密码
2. **使用 HTTPS**: 配置 SSL 证书，启用 HTTPS
3. **防火墙配置**: 限制数据库端口（5432、5433）仅允许本地访问
4. **定期备份**: 设置自动备份任务
5. **日志审计**: 定期检查访问日志
6. **更新维护**: 及时更新到最新版本

## 📞 支持和反馈

- **文档**: https://docs.kimbap.io
- **问题反馈**: https://github.com/kimbapio/kimbap/issues
- **邮件支持**: support@kimbap.io

## 📄 许可证

KIMBAP 采用 MIT 许可证。详见 LICENSE 文件。

---

**祝您使用愉快！** 🎉

如有任何问题，请随时联系我们。
