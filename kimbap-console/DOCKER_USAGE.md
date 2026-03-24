# KIMBAP Console Docker 使用指南

本指南介绍如何使用提供的脚本快速启动和管理 KIMBAP Console Docker 服务。

## 快速开始

### 1. 首次启动

```bash
cd kimbap-console
./start-console.sh
```

脚本会自动：
- ✅ 检查 Docker 运行状态
- ✅ 检查并创建 `.env` 配置文件
- ✅ 拉取最新 Docker 镜像
- ✅ 启动服务（Console + PostgreSQL）
- ✅ 显示服务状态和访问地址

**首次启动时**，如果没有 `.env` 文件，脚本会提示：
1. 从 `.env.console.example` 创建配置文件
2. 编辑关键配置（密钥、Core 地址等）

### 2. 配置说明

**必须修改的配置** (在 `.env` 文件中):

```bash
# 认证密钥（至少32字符，生产环境必须修改）
CONSOLE_JWT_SECRET=your-very-long-secret-key-here-at-least-32-chars

# Kimbap Core 服务地址
MCP_GATEWAY_URL=http://localhost:3002        # 或远程地址
PROXY_ADMIN_URL=http://localhost:3002/admin
```

**可选配置**:

```bash
# 端口配置
CONSOLE_PORT=3000
POSTGRES_PORT=5432

# 数据库配置
POSTGRES_USER=kimbap
POSTGRES_PASSWORD=kimbap123
POSTGRES_DB=kimbap_console

# 日志同步配置
LOG_SYNC_ENABLED=true
LOG_SYNC_INTERVAL_MINUTES=2
```

## 管理脚本

### 启动服务

```bash
./start-console.sh
```

**功能**:
- 检查运行环境
- 自动配置 .env 文件
- 拉取最新镜像
- 启动完整服务栈
- 显示启动状态和日志

### 停止服务

```bash
./stop-console.sh
```

**选项**:
1. **仅停止服务（保留数据）** - 数据库数据保留，下次启动可恢复
2. **停止并删除数据** - 完全清理，包括数据库（⚠️ 不可恢复）

### 重启服务

```bash
./restart-console.sh
```

**功能**: 快速重启服务，保留所有数据

### 更新服务

```bash
./update-console.sh
```

**功能**:
- 拉取最新 Docker 镜像
- 自动重启服务
- 显示更新后的状态

### 查看日志

```bash
# 查看所有服务日志
./logs-console.sh

# 查看 Console 日志
./logs-console.sh kimbap-console

# 查看数据库日志
./logs-console.sh postgres-console
```

按 `Ctrl+C` 退出日志查看

## 构建脚本

### 构建并推送到 Docker Hub

```bash
./build-and-push.sh          # 构建 latest 标签
./build-and-push.sh v1.0.0   # 构建指定标签
```

**功能**:
- 自动创建多架构构建器
- 构建 linux/amd64 和 linux/arm64 双架构
- 推送到 Docker Hub
- 验证推送结果

### 本地构建（开发用）

```bash
./build-local.sh                    # 默认当前平台
./build-local.sh latest linux/arm64 # 指定平台
```

**功能**:
- 快速本地构建
- 不推送到远程
- 适合开发测试

## 常用命令参考

### Docker Compose 命令

```bash
# 进入项目根目录
cd /path/to/kimbap

# 查看服务状态
docker-compose -f docker-compose.console.yml ps

# 查看日志
docker-compose -f docker-compose.console.yml logs -f

# 重启特定服务
docker-compose -f docker-compose.console.yml restart kimbap-console

# 停止服务
docker-compose -f docker-compose.console.yml down

# 停止并删除数据
docker-compose -f docker-compose.console.yml down -v
```

### Docker 命令

```bash
# 查看运行中的容器
docker ps

# 查看所有容器（包括停止的）
docker ps -a

# 查看容器日志
docker logs -f kimbap-console

# 进入容器 shell
docker exec -it kimbap-console sh

# 查看镜像
docker images | grep kimbap-console

# 清理未使用的镜像
docker image prune -a
```

## 访问服务

启动成功后：

- **Console Web 界面**: http://localhost:3000 （或配置的端口）
- **PostgreSQL 数据库**: localhost:5432

## 故障排查

### 1. 启动失败

**检查 Docker 状态**:
```bash
docker info
```

**查看详细日志**:
```bash
./logs-console.sh
```

### 2. 无法访问

**检查端口占用**:
```bash
lsof -i :3000
lsof -i :5432
```

**检查容器状态**:
```bash
docker ps
```

### 3. 数据库连接失败

**查看数据库日志**:
```bash
./logs-console.sh postgres-console
```

**检查数据库健康状态**:
```bash
docker exec kimbap-console-postgres pg_isready -U kimbap
```

### 4. 无法连接到 Core 服务

**测试连接**:
```bash
docker exec -it kimbap-console curl http://your-core-url:3002/health
```

**检查环境变量**:
```bash
docker exec kimbap-console env | grep MCP_GATEWAY_URL
```

## 数据管理

### 备份数据库

```bash
docker exec kimbap-console-postgres pg_dump -U kimbap kimbap_console > backup.sql
```

### 恢复数据库

```bash
docker exec -i kimbap-console-postgres psql -U kimbap kimbap_console < backup.sql
```

### 重置数据库

```bash
./stop-console.sh
# 选择选项 2 - 停止并删除数据
./start-console.sh
```

## 生产环境建议

1. ✅ **修改默认密码**: 更改 `POSTGRES_PASSWORD`
2. ✅ **使用强密钥**: `JWT_SECRET` 至少 32 字符
3. ✅ **使用 HTTPS**: 配置反向代理（Nginx/Caddy）
4. ✅ **定期备份**: 定期备份 PostgreSQL 数据
5. ✅ **监控日志**: 使用 `./logs-console.sh` 监控服务运行状态
6. ✅ **限制访问**: 配置防火墙，限制数据库端口访问

## 目录结构

```
kimbap-console/
├── start-console.sh      # 启动服务
├── stop-console.sh       # 停止服务
├── restart-console.sh    # 重启服务
├── update-console.sh     # 更新服务
├── logs-console.sh       # 查看日志
├── build-and-push.sh     # 构建并推送镜像
├── build-local.sh        # 本地构建
└── DOCKER_USAGE.md       # 本文档
```

## 技术支持

如有问题，请查看：
- 完整部署文档: `/path/to/kimbap/CONSOLE_DEPLOYMENT.md`
- 日志输出: `./logs-console.sh`
- 容器状态: `docker-compose -f ../docker-compose.console.yml ps`
