# Docker 部署指南

## 🐳 概述

KIMBAP Console 提供了完整的 Docker 容器化部署方案，支持快速部署、自动数据库初始化和一键启动。

## 📋 前置要求

- Docker Engine 20.10+ 或 Docker Desktop
- Docker Compose v2.0+
- 可用端口：3000 (前端)、3002 (后端)、5432 (PostgreSQL)、8080 (Adminer，可选)

### 🖥️ 系统兼容性

- **Linux**: 支持 ARM64 和 AMD64 架构
- **macOS**: 支持 ARM64 (Apple Silicon) 和 AMD64 (Intel) 架构  
- **Windows**: 支持 AMD64 架构

> ✅ **跨平台优化**: Docker Compose 配置已优化，所有镜像明确指定 `platform: linux/amd64` 确保在所有系统上的兼容性。

## 🚀 快速部署

### 方式一：智能部署（自动端口分配，推荐）

🎯 **智能端口分配**: 自动检测可用端口，避免端口占用冲突

#### 📱 一键启动

**Linux/Mac 系统:**
```bash
# 克隆项目并自动部署
git clone <repository-url>
cd kimbap-console
./docker-run-with-auto-ports.sh
```

**Windows 系统:**
```cmd
REM 克隆项目并自动部署
git clone <repository-url>
cd kimbap-console
docker-run-with-auto-ports.cmd
```

#### 🔧 手动步骤

如果你需要分步执行：

```bash
# 1. 查找可用端口
npm run docker:find-ports

# 2. 使用分配的端口启动服务
docker compose --env-file .env.ports up -d

# 或者使用 npm 脚本
npm run docker:up:auto-ports
```

### 方式二：标准部署（固定端口）

#### 🌐 所有系统（Linux / macOS / Windows）

1. **创建 docker-compose.yml**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: kimbap-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: kimbap
      POSTGRES_PASSWORD: kimbap123
      POSTGRES_DB: kimbap_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kimbap -d kimbap_db"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kimbap-network

  kimbap-console:
    image: kimbapio/kimbap-console:latest
    container_name: kimbap-console-app
    restart: unless-stopped
    pull_policy: always
    ports:
      - "3000:3000"
      - "3002:3002"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: kimbap
      DB_PASSWORD: kimbap123
      DB_NAME: kimbap_db
      DATABASE_URL: postgresql://kimbap:kimbap123@postgres:5432/kimbap_db
      NODE_ENV: production
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - kimbap-network

  # 可选：数据库管理界面
  adminer:
    image: adminer:latest
    container_name: kimbap-adminer
    restart: unless-stopped
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    environment:
      ADMINER_DEFAULT_SERVER: postgres
    networks:
      - kimbap-network

volumes:
  postgres_data:
    driver: local

networks:
  kimbap-network:
    driver: bridge
```

2. **启动服务**

```bash
# 拉取最新镜像并启动
docker compose pull
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f
```

3. **访问应用**

- 前端界面：http://localhost:3000
- 后端API：http://localhost:3002
- 数据库管理（如果启用）：http://localhost:8080

### 方式二：本地构建镜像

1. **克隆项目**

```bash
git clone https://github.com/your-org/kimbap-console.git
cd kimbap-console
```

2. **构建镜像**

```bash
# 构建镜像
npm run docker:build

# 或使用 Docker 命令直接构建
docker build -t kimbapio/kimbap-console:latest .
```

3. **启动服务**

```bash
# 使用项目提供的 docker-compose.yml
docker compose up -d
```

## 🔧 端口管理

### 🎯 智能端口分配

KIMBAP Console 提供智能端口分配功能，自动检测并分配可用端口：

#### 端口检测原理

1. **检测默认端口**: 从默认端口开始检测（3000、3002、5432、8080）
2. **递增查找**: 如果端口被占用，自动递增查找下一个可用端口
3. **生成配置**: 自动生成 `.env.ports` 和 `.port-config.json` 配置文件
4. **启动服务**: 使用分配的端口启动 Docker 服务

#### 可用的 NPM 命令

```bash
# 查找可用端口并生成配置
npm run docker:find-ports

# 使用自动分配的端口启动服务
npm run docker:up:auto-ports

# 一键智能部署
npm run docker:deploy:auto
```

#### 端口配置文件

**`.env.ports` 文件示例:**
```env
# Auto-generated port configuration
FRONTEND_PORT=3000
BACKEND_PORT=3002
POSTGRES_PORT=5432
ADMINER_PORT=8080

# For Docker Compose
KIMBAP_FRONTEND_PORT=3000
KIMBAP_BACKEND_PORT=3002
KIMBAP_POSTGRES_PORT=5432
KIMBAP_ADMINER_PORT=8080
```

**`.port-config.json` 文件示例:**
```json
{
  "frontend": 3000,
  "backend": 3002,
  "postgres": 5432,
  "adminer": 8080
}
```

### 🔧 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `DB_HOST` | 数据库主机 | postgres | ✅ |
| `DB_PORT` | 数据库端口 | 5432 | ✅ |
| `DB_USER` | 数据库用户 | kimbap | ✅ |
| `DB_PASSWORD` | 数据库密码 | kimbap123 | ✅ |
| `DB_NAME` | 数据库名称 | kimbap_db | ✅ |
| `DATABASE_URL` | 完整数据库连接字符串 | - | ✅ |
| `NODE_ENV` | 运行环境 | production | ✅ |
| `CLOUDFLARE_TUNNEL_TOKEN` | Cloudflare隧道令牌 | - | ❌ |

### 端口映射

| 服务 | 容器端口 | 主机端口 | 说明 |
|------|----------|----------|------|
| 前端 | 3000 | 3000 | Next.js 应用 |
| 后端 | 3002 | 3002 | MCP Gateway API |
| PostgreSQL | 5432 | 5432 | 数据库服务 |
| Adminer | 8080 | 8080 | 数据库管理界面（可选） |

## 🛠️ 管理命令

### 基本操作

```bash
# 启动所有服务
npm run docker:up

# 停止所有服务
npm run docker:down

# 重启服务
docker compose restart

# 查看服务状态
docker compose ps

# 查看日志
npm run docker:logs

# 清理（删除容器和网络，保留数据卷）
docker compose down

# 完全清理（包括数据卷）
docker compose down -v
```

### 构建和发布

```bash
# 构建镜像
npm run docker:build

# 推送到 Docker Hub
npm run docker:push

# 构建并推送
npm run docker:build:push

# 完整部署流程（构建、推送、拉取、启动）
npm run docker:deploy:full
```

### 调试和故障排除

```bash
# 查看容器日志
docker logs kimbap-console-app

# 进入容器内部
docker exec -it kimbap-console-app sh

# 检查数据库连接
docker exec kimbap-postgres pg_isready -U kimbap -d kimbap_db

# 查看容器资源使用
docker stats

# 检查网络连接
docker network inspect kimbap-console_kimbap-network
```

## 🗃️ 数据持久化

### 数据卷

- **postgres_data**: PostgreSQL 数据目录
- 数据存储在 Docker 卷中，容器重启后数据不会丢失
- 备份数据：`docker run --rm -v postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_backup.tar.gz -C /data .`
- 恢复数据：`docker run --rm -v postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/postgres_backup.tar.gz -C /data`

## 🔐 安全建议

1. **修改默认密码**
   - 修改数据库密码（POSTGRES_PASSWORD）

2. **网络隔离**
   - 生产环境建议只暴露必要端口
   - 使用自定义网络进行服务隔离

3. **HTTPS 配置**
   - 使用反向代理（如 Nginx）配置 SSL
   - 或使用 Cloudflare Tunnel 提供 HTTPS

## 🌐 生产部署

### 使用环境文件

1. **创建 .env 文件**

```env
# 数据库配置
DB_HOST=postgres
DB_PORT=5432
DB_USER=kimbap
DB_PASSWORD=your_secure_password
DB_NAME=kimbap_db
DATABASE_URL=postgresql://kimbap:your_secure_password@postgres:5432/kimbap_db

# 应用配置
NODE_ENV=production

# 可选：Cloudflare Tunnel
CLOUDFLARE_TUNNEL_TOKEN=your_tunnel_token
```

2. **启动服务**

```bash
# 使用环境文件启动
docker compose --env-file .env up -d
```

### 使用 Docker Swarm

```bash
# 初始化 Swarm
docker swarm init

# 部署服务栈
docker stack deploy -c docker-compose.yml kimbap-console

# 查看服务
docker service ls
```

### 使用 Kubernetes

参考项目中的 `k8s/` 目录获取 Kubernetes 部署配置（如果有）。

## 📊 监控和日志

### 健康检查

服务包含内置健康检查：
- PostgreSQL：每 10 秒检查一次连接
- KIMBAP Console：自动重启失败的容器

### 日志管理

```bash
# 实时查看所有服务日志
docker compose logs -f

# 查看特定服务日志
docker compose logs kimbap-console

# 导出日志到文件
docker compose logs > logs.txt

# 限制日志行数
docker compose logs --tail=100
```

## ❗ 常见问题

### 1. 数据库初始化失败

**问题**: `P3009: migrate found failed migrations`

**解决方案**:
- 应用会自动处理失败的迁移
- 如果问题持续，清理数据卷重新开始：`docker compose down -v`

### 2. 端口已被占用

**问题**: `bind: address already in use`

**解决方案**:
```bash
# 查找占用端口的进程
lsof -i :3000
# 或修改 docker-compose.yml 中的端口映射
# 例如：改为 "3001:3000"
```

### 3. 容器无法连接数据库

**问题**: `Database not reachable`

**解决方案**:
- 确保 PostgreSQL 容器健康运行
- 检查网络配置
- 验证环境变量设置正确

### 4. 镜像拉取失败

**问题**: `pull access denied`

**解决方案**:
```bash
# 确保使用最新镜像
docker pull kimbapio/kimbap-console:latest

# 或本地构建
npm run docker:build
```

## 🔄 更新升级

1. **备份数据**
```bash
docker exec kimbap-postgres pg_dump -U kimbap kimbap_db > backup.sql
```

2. **拉取新版本**
```bash
docker compose pull
```

3. **重启服务**
```bash
docker compose up -d
```

4. **验证更新**
```bash
docker compose logs kimbap-console | grep version
```

## 📚 相关文档

- [项目README](../README.md)
- [部署指南](./DEPLOYMENT_GUIDE.md)
- [数据库管理](./DATABASE_MANAGEMENT.md)
- [故障排除](./PORTABLE_BUILD_TROUBLESHOOTING.md)

## 💬 支持

如遇到问题，请查看：
- GitHub Issues: https://github.com/your-org/kimbap-console/issues
- 文档站点: https://docs.kimbap.io