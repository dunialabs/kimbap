# HTTPS 远程部署指南（无需 Nginx）

本指南提供两种无需 Nginx 的 HTTPS 远程部署方案，推荐使用 **Cloudflare Tunnel**（方案一）。

## 🎯 方案对比

| 方案 | 优点 | 缺点 | 适用场景 |
|------|------|------|----------|
| **Cloudflare Tunnel** | ✅ 自动 HTTPS<br>✅ 无需开放端口<br>✅ 免费<br>✅ 零配置证书 | ⚠️ 需要 Cloudflare 账户 | **推荐：所有远程部署** |
| **Docker 挂载证书** | ✅ 完全自主控制<br>✅ 不依赖第三方 | ⚠️ 需要自己管理证书<br>⚠️ 需要开放端口 | 有自有证书且需要完全控制 |

---

## 方案一：Cloudflare Tunnel（推荐）⭐

Cloudflare Tunnel 是最简单、最安全的远程部署方案，自动提供 HTTPS，无需管理证书。

### 优势

- ✅ **自动 HTTPS**：由 Cloudflare 管理，自动续期
- ✅ **无需开放端口**：服务完全隐藏在 Cloudflare 后面
- ✅ **免费使用**：Cloudflare 免费套餐即可
- ✅ **零配置证书**：无需申请、安装、更新证书
- ✅ **DDoS 防护**：自动获得 Cloudflare 的 DDoS 保护

### 前提条件

- ✅ Docker 已安装并运行
- ✅ Cloudflare 账户（免费即可）
- ✅ 一个域名（已添加到 Cloudflare）

### 部署步骤

#### 1️⃣ 准备部署目录

```bash
mkdir -p ~/kimbap-deployment
cd ~/kimbap-deployment
```

#### 2️⃣ 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  # PostgreSQL 数据库
  postgres:
    image: postgres:16-alpine
    container_name: kimbap-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: kimbap
      POSTGRES_PASSWORD: ${DB_PASSWORD:-kimbap123}
      POSTGRES_DB: kimbap_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U kimbap -d kimbap_db']
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kimbap-network

  # KIMBAP Core (MCP Gateway)
  kimbap-core:
    image: kimbapio/kimbap-core:latest
    container_name: kimbap-core
    restart: unless-stopped
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://kimbap:${DB_PASSWORD:-kimbap123}@postgres:5432/kimbap_db?schema=public
      PORT: 3002
      BACKEND_PORT: 3002
      JWT_SECRET: ${CORE_JWT_SECRET:-change-this-secret-key}
    networks:
      - kimbap-network
    healthcheck:
      test: ['CMD-SHELL', 'curl -f http://localhost:3002 || exit 0']
      interval: 30s
      timeout: 10s
      retries: 3

  # KIMBAP Console
  kimbap-console:
    image: kimbapio/kimbap-console:latest
    container_name: kimbap-console
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      kimbap-core:
        condition: service_started
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://kimbap:${DB_PASSWORD:-kimbap123}@postgres:5432/kimbap_db
      PORT: 3000
      MCP_GATEWAY_URL: http://kimbap-core:3002
      PROXY_ADMIN_URL: http://kimbap-core:3002/admin
      JWT_SECRET: ${CONSOLE_JWT_SECRET:-change-this-secret-key}
    networks:
      - kimbap-network

  # Cloudflare Tunnel
  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: kimbap-cloudflared
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN}
    networks:
      - kimbap-network
    volumes:
      - ./cloudflared:/etc/cloudflared
    depends_on:
      - kimbap-console
      - kimbap-core

volumes:
  postgres_data:
    driver: local

networks:
  kimbap-network:
    driver: bridge
```

#### 3️⃣ 创建 Cloudflare Tunnel

##### 方式 A：使用 Tunnel Token（推荐，最简单）

1. **登录 Cloudflare Dashboard**
   - 访问：https://dash.cloudflare.com/
   - 选择你的账户

2. **创建 Tunnel**
   - 进入：Zero Trust → Networks → Tunnels
   - 点击 "Create a tunnel"
   - 选择 "Cloudflared" 类型
   - 输入名称：`kimbap-console`
   - 点击 "Save tunnel"

3. **配置路由**
   - 在隧道详情页，点击 "Configure"
   - 添加 Public Hostname：
     - **Subdomain**: `console`（或你想要的子域名）
     - **Domain**: 选择你的域名（如 `example.com`）
     - **Service**: `http://kimbap-console:3000`
   - 如果需要 API 子域名：
     - **Subdomain**: `api`
     - **Domain**: 选择你的域名
     - **Service**: `http://kimbap-core:3002`
   - 点击 "Save"

4. **复制 Tunnel Token**
   - 在隧道详情页，复制 "Tunnel token"
   - 格式类似：`eyJhIjoi...`（很长的字符串）

5. **创建 .env 文件**

```bash
# 在 ~/kimbap-deployment 目录下创建 .env 文件
cat > .env << EOF
# 数据库密码（生产环境请修改）
DB_PASSWORD=your-secure-password-here

# JWT 密钥（生产环境请修改，至少 32 字符）
CONSOLE_JWT_SECRET=$(openssl rand -base64 32)
CORE_JWT_SECRET=$(openssl rand -base64 32)

# Cloudflare Tunnel Token
CLOUDFLARE_TUNNEL_TOKEN=你的隧道token
EOF
```

##### 方式 B：使用配置文件（适合高级用户）

如果你更喜欢使用配置文件方式，参考 [CLOUDFLARED_QUICK_START.md](./CLOUDFLARED_QUICK_START.md)

#### 4️⃣ 启动服务

```bash
cd ~/kimbap-deployment

# 创建 cloudflared 配置目录
mkdir -p cloudflared

# 启动所有服务
docker compose up -d

# 查看日志
docker compose logs -f
```

#### 5️⃣ 验证部署

访问你的域名：
- 🌐 **Console**: `https://console.yourdomain.com`
- 🔌 **API**: `https://api.yourdomain.com`（如果配置了）

### 常用命令

```bash
# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f cloudflared
docker compose logs -f kimbap-console

# 重启服务
docker compose restart

# 更新镜像
docker compose pull
docker compose up -d

# 停止服务
docker compose down

# 停止并删除数据（⚠️ 危险）
docker compose down -v
```

### 故障排查

#### 问题：隧道无法连接

```bash
# 检查 Tunnel Token 是否正确
docker compose logs cloudflared | grep -i error

# 验证 Token 格式
echo $CLOUDFLARE_TUNNEL_TOKEN | head -c 50
```

#### 问题：502 Bad Gateway

```bash
# 检查服务是否正常运行
docker compose ps

# 测试容器内连接
docker compose exec kimbap-console curl http://localhost:3000
docker compose exec kimbap-core curl http://localhost:3002
```

#### 问题：DNS 不解析

- 等待 DNS 传播（通常 1-5 分钟）
- 检查 Cloudflare Dashboard 中的 DNS 记录是否自动创建

---

## 方案二：Docker 挂载证书（自主控制）

如果你有自己的 SSL 证书，可以直接在 Docker 中启用 HTTPS。

### 前提条件

- ✅ 已有 SSL 证书文件（`.pem` 和 `.key`）
- ✅ 证书已安装到服务器

### 部署步骤

#### 1️⃣ 准备证书目录

```bash
mkdir -p ~/kimbap-deployment/certs
# 将你的证书文件复制到此目录
# - fullchain.pem (证书链)
# - privkey.pem (私钥)
```

#### 2️⃣ 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: kimbap-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: kimbap
      POSTGRES_PASSWORD: ${DB_PASSWORD:-kimbap123}
      POSTGRES_DB: kimbap_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - kimbap-network

  kimbap-core:
    image: kimbapio/kimbap-core:latest
    container_name: kimbap-core
    restart: unless-stopped
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://kimbap:${DB_PASSWORD:-kimbap123}@postgres:5432/kimbap_db?schema=public
      PORT: 3002
      JWT_SECRET: ${CORE_JWT_SECRET:-change-this-secret-key}
    networks:
      - kimbap-network

  kimbap-console:
    image: kimbapio/kimbap-console:latest
    container_name: kimbap-console
    restart: unless-stopped
    depends_on:
      - postgres
      - kimbap-core
    environment:
      NODE_ENV: production
      DATABASE_URL: postgresql://kimbap:${DB_PASSWORD:-kimbap123}@postgres:5432/kimbap_db
      PORT: 3000
      FRONTEND_HTTPS_PORT: 3000
      ENABLE_HTTPS: 'true'
      SSL_CERT_PATH: /certs/fullchain.pem
      SSL_KEY_PATH: /certs/privkey.pem
      MCP_GATEWAY_URL: http://kimbap-core:3002
      JWT_SECRET: ${CONSOLE_JWT_SECRET:-change-this-secret-key}
    ports:
      - '443:3000'  # HTTPS 端口
    volumes:
      - ./certs:/certs:ro  # 只读挂载证书
    networks:
      - kimbap-network

volumes:
  postgres_data:

networks:
  kimbap-network:
    driver: bridge
```

#### 3️⃣ 创建 .env 文件

```bash
cat > .env << EOF
DB_PASSWORD=your-secure-password
CONSOLE_JWT_SECRET=$(openssl rand -base64 32)
CORE_JWT_SECRET=$(openssl rand -base64 32)
EOF
```

#### 4️⃣ 启动服务

```bash
docker compose up -d
```

#### 5️⃣ 访问服务

- 🌐 **HTTPS**: `https://your-server-ip` 或 `https://your-domain.com`

### 证书更新

当证书需要更新时：

```bash
# 1. 更新证书文件
cp new-fullchain.pem ~/kimbap-deployment/certs/fullchain.pem
cp new-privkey.pem ~/kimbap-deployment/certs/privkey.pem

# 2. 重启服务
docker compose restart kimbap-console
```

---

## 🔒 安全建议

### 生产环境必须修改

1. **数据库密码**
   ```bash
   DB_PASSWORD=强密码至少16字符
   ```

2. **JWT 密钥**
   ```bash
   CONSOLE_JWT_SECRET=$(openssl rand -base64 32)
   CORE_JWT_SECRET=$(openssl rand -base64 32)
   ```

3. **防火墙配置**
   ```bash
   # 如果使用方案二，只开放必要端口
   sudo ufw allow 443/tcp
   sudo ufw enable
   ```

### 定期维护

```bash
# 备份数据
docker compose exec postgres pg_dump -U kimbap kimbap_db > backup.sql

# 更新镜像
docker compose pull
docker compose up -d

# 清理未使用的资源
docker system prune -a
```

---

## 📊 方案选择建议

| 场景 | 推荐方案 |
|------|----------|
| **个人项目/小团队** | Cloudflare Tunnel（方案一） |
| **企业部署，有 Cloudflare 账户** | Cloudflare Tunnel（方案一） |
| **企业部署，需要完全自主控制** | Docker 挂载证书（方案二） |
| **内网部署** | 方案二（使用自签名证书） |
| **快速原型/测试** | Cloudflare Tunnel（方案一） |

---

## 🆘 需要帮助？

- 📝 查看详细日志：`docker compose logs -f`
- 🔍 Cloudflare Tunnel 文档：[CLOUDFLARED_QUICK_START.md](./CLOUDFLARED_QUICK_START.md)
- 💬 [Cloudflare 社区论坛](https://community.cloudflare.com/)

---

**推荐使用方案一（Cloudflare Tunnel）**，它是最简单、最安全、最省心的 HTTPS 部署方案！🎉

