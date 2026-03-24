# 使用 Cloudflare Tunnel 的部署脚本修改指南

本文档说明使用方案1（Cloudflare Tunnel）进行 HTTPS 远程部署时，需要修改哪些脚本和配置文件。

## 📋 需要修改的文件清单

### 1. `docker-compose.yml` ✅ 已配置（需优化）

**当前状态**：已有 `cloudflared` 服务配置

**需要优化的地方**：

```yaml
cloudflared:
  image: cloudflare/cloudflared:latest
  container_name: kimbap-cloudflared
  restart: unless-stopped
  command: tunnel --no-autoupdate run
  environment:
    - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN:-}  # ⚠️ 需要确保有默认值处理
  networks:
    - kimbap-network
  volumes:
    - ./cloudflared:/etc/cloudflared
  depends_on:  # ⚠️ 需要添加依赖
    - kimbap-console
    - kimbap-core
```

**建议修改**：

```yaml
cloudflared:
  image: cloudflare/cloudflared:latest
  container_name: kimbap-cloudflared
  restart: unless-stopped
  command: tunnel --no-autoupdate run
  environment:
    - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN:-}
  networks:
    - kimbap-network
  volumes:
    - ./cloudflared:/etc/cloudflared
  depends_on:
    kimbap-console:
      condition: service_started
    kimbap-core:
      condition: service_started
  # 可选：如果 TUNNEL_TOKEN 未设置，则不启动此服务
  profiles:
    - cloudflared
```

---

### 2. `start-console.sh` ⚠️ 需要修改

**需要添加的功能**：

1. 检查 `CLOUDFLARE_TUNNEL_TOKEN` 环境变量
2. 在生成的 `docker-compose.console.yml` 中添加 `cloudflared` 服务
3. 在 `.env` 文件生成时添加 `CLOUDFLARE_TUNNEL_TOKEN` 配置
4. 更新访问 URL 提示（显示 HTTPS 域名而非 localhost）

**修改位置 1：在生成 docker-compose.console.yml 时添加 cloudflared 服务**

在 `start-console.sh` 的第 77-145 行（生成 docker-compose 文件的部分），添加：

```bash
# 在 kimbap-console 服务后添加
  # Cloudflare Tunnel (Optional - for HTTPS remote access)
  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: kimbap-cloudflared
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN:-}
    networks:
      - kimbap-network
    depends_on:
      kimbap-console:
        condition: service_started
    profiles:
      - cloudflared
```

**修改位置 2：在生成 .env 文件时添加 Cloudflare 配置**

在第 167-199 行（生成 .env 文件的部分），添加：

```bash
# Cloudflare Tunnel Configuration (Optional - for HTTPS remote access)
# Get your tunnel token from: https://dash.cloudflare.com/
# Zero Trust → Networks → Tunnels → Create tunnel
CLOUDFLARE_TUNNEL_TOKEN=
```

**修改位置 3：更新访问 URL 提示**

在第 268-270 行（显示访问 URL 的部分），修改为：

```bash
# 检查是否有 Cloudflare Tunnel Token
if [ -n "$CLOUDFLARE_TUNNEL_TOKEN" ] || grep -q "^CLOUDFLARE_TUNNEL_TOKEN=" "$ENV_FILE" 2>/dev/null && [ -n "$(grep "^CLOUDFLARE_TUNNEL_TOKEN=" "$ENV_FILE" 2>/dev/null | cut -d '=' -f2)" ]; then
    echo "Access URLs:"
    echo "   Console (HTTPS): https://your-domain.com (configured in Cloudflare)"
    echo "   Console (Local): http://localhost:$CONSOLE_PORT"
    echo ""
    echo "💡 Cloudflare Tunnel is enabled - your service is accessible via HTTPS!"
else
    echo "Access URLs:"
    echo "   Console: http://localhost:$CONSOLE_PORT"
    echo ""
    echo "💡 To enable HTTPS remote access, set CLOUDFLARE_TUNNEL_TOKEN in .env file"
fi
```

---

### 3. `docker-compose.prod.yml` ⚠️ 需要修改

**当前状态**：没有 `cloudflared` 服务

**需要添加**：

```yaml
  # Cloudflare Tunnel for HTTPS remote access
  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: kimbap-cloudflared-prod
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      - TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN:-}
    networks:
      - kimbap-network
    volumes:
      - ./cloudflared:/etc/cloudflared
    depends_on:
      kimbap-console:
        condition: service_started
    profiles:
      - cloudflared
```

---

### 4. `docker/deploy.sh` ⚠️ 需要修改

**当前状态**：使用 `docker-compose.prod.yml`，但没有处理 Cloudflare Tunnel

**需要添加**：

1. 检查 `CLOUDFLARE_TUNNEL_TOKEN` 环境变量
2. 更新访问 URL 提示

**修改位置**：在第 36-40 行（显示访问 URL 的部分），修改为：

```bash
echo ""
echo "🎉 Deployment complete!"
echo ""

# Check if Cloudflare Tunnel is configured
if [ -n "$CLOUDFLARE_TUNNEL_TOKEN" ]; then
    echo "📱 Access URLs:"
    echo "   Console (HTTPS): https://your-domain.com (configured in Cloudflare)"
    echo "   Console (Local): http://localhost:3000"
    echo "   API: http://localhost:3002"
    echo ""
    echo "💡 Cloudflare Tunnel is enabled - your service is accessible via HTTPS!"
else
    echo "📱 Access URLs:"
    echo "   App: http://localhost:3000"
    echo "   API: http://localhost:3002"
    echo "   Database Admin: http://localhost:8080 (optional)"
    echo ""
    echo "💡 To enable HTTPS remote access, set CLOUDFLARE_TUNNEL_TOKEN environment variable"
fi
```

---

### 5. 创建 `.env.example` 文件（可选但推荐）

创建示例环境变量文件，方便用户配置：

```bash
# .env.example
# PostgreSQL Database Configuration
POSTGRES_USER=kimbap
POSTGRES_PASSWORD=kimbap123
POSTGRES_DB=kimbap_console
POSTGRES_PORT=5432

# Console Service Port
CONSOLE_PORT=3000

# Authentication Secrets
CONSOLE_JWT_SECRET=your-console-jwt-secret-change-in-production-min-32-chars

# Kimbap Core Connection Configuration
MCP_GATEWAY_URL=http://kimbap-core:3002
PROXY_ADMIN_URL=http://kimbap-core:3002/admin

# Cloudflare Tunnel Configuration (Optional - for HTTPS remote access)
# Get your tunnel token from: https://dash.cloudflare.com/
# Zero Trust → Networks → Tunnels → Create tunnel
CLOUDFLARE_TUNNEL_TOKEN=
```

---

## 🔧 具体修改步骤

### 步骤 1：修改 `start-console.sh`

```bash
# 1. 在生成 docker-compose.console.yml 时添加 cloudflared 服务
# 2. 在生成 .env 文件时添加 CLOUDFLARE_TUNNEL_TOKEN
# 3. 更新访问 URL 提示逻辑
```

### 步骤 2：修改 `docker-compose.prod.yml`

```bash
# 添加 cloudflared 服务配置
```

### 步骤 3：修改 `docker/deploy.sh`

```bash
# 更新访问 URL 提示，支持 Cloudflare Tunnel
```

### 步骤 4：更新 `docker-compose.yml`（可选优化）

```bash
# 添加 depends_on 和 profiles 配置
```

---

## 📝 修改后的使用流程

### 用户使用流程：

1. **运行启动脚本**
   ```bash
   ./start-console.sh
   ```

2. **配置 Cloudflare Tunnel Token**
   - 脚本会生成 `.env` 文件
   - 用户编辑 `.env` 文件，添加 `CLOUDFLARE_TUNNEL_TOKEN`

3. **在 Cloudflare Dashboard 配置路由**
   - 访问：https://dash.cloudflare.com/
   - Zero Trust → Networks → Tunnels
   - 配置域名路由到 `http://kimbap-console:3000`

4. **重启服务（如果需要）**
   ```bash
   docker compose -f docker-compose.console.yml restart
   ```

5. **访问 HTTPS 域名**
   - `https://your-domain.com`

---

## ✅ 验证修改

修改完成后，验证以下功能：

1. ✅ 脚本能正常生成包含 `cloudflared` 的 docker-compose 文件
2. ✅ `.env` 文件包含 `CLOUDFLARE_TUNNEL_TOKEN` 配置项
3. ✅ 当设置了 `CLOUDFLARE_TUNNEL_TOKEN` 时，`cloudflared` 服务能正常启动
4. ✅ 访问 URL 提示正确显示 HTTPS 信息
5. ✅ 未设置 Token 时，服务仍能正常启动（cloudflared 不启动）

---

## 🚀 快速修改脚本

如果需要，我可以帮你直接修改这些文件。请告诉我：

1. 是否现在就开始修改这些文件？
2. 是否需要保留向后兼容（未设置 Token 时也能正常运行）？
3. 是否需要添加交互式配置（询问用户是否要配置 Cloudflare Tunnel）？

---

## 📚 相关文档

- [HTTPS 远程部署指南](./HTTPS_REMOTE_DEPLOYMENT.md)
- [Cloudflare Tunnel 快速开始](./CLOUDFLARED_QUICK_START.md)

