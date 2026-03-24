# Cloudflared 快速开始指南

本指南帮助您在 5 分钟内设置 Cloudflare Tunnel，将您的 Kimbap Console 服务安全地暴露到互联网。

## 前提条件

- ✅ Docker 已安装并运行
- ✅ Cloudflare 账户（免费即可）
- ✅ 一个域名（已添加到 Cloudflare）

## 快速设置步骤

### 1️⃣ 安装 Cloudflared

```bash
# 自动安装（推荐）
npm run cloudflared:setup

# 或在启动时自动安装
npm run dev
```

### 2️⃣ 创建隧道

```bash
# 登录 Cloudflare
docker run -it --rm -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest tunnel login

# 创建名为 kimbap-console 的隧道
docker run -it --rm -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest tunnel create kimbap-console
```

> 💡 记下生成的隧道 ID，格式类似：`6ff42ae2-765d-4adf-8112-31c55c1551ef`

### 3️⃣ 配置 DNS

```bash
# 将您的域名指向隧道
docker run -it --rm -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest tunnel route dns kimbap-console example.com

# 如需 API 子域名
docker run -it --rm -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest tunnel route dns kimbap-console api.example.com
```

### 4️⃣ 配置隧道

编辑 `./cloudflared/config.yml`：

```yaml
tunnel: YOUR_TUNNEL_ID  # 替换为步骤 2 中的隧道 ID
credentials-file: /etc/cloudflared/YOUR_TUNNEL_ID.json

ingress:
  # 前端
  - hostname: example.com
    service: http://host.docker.internal:3000
  # API
  - hostname: api.example.com
    service: http://host.docker.internal:3002
  # 默认
  - service: http_status:404
```

### 5️⃣ 复制凭据

```bash
# 复制隧道凭据文件到项目
cp ~/.cloudflared/YOUR_TUNNEL_ID.json ./cloudflared/
```

### 6️⃣ 启动隧道

```bash
# 启动 cloudflared 服务
docker compose up -d cloudflared

# 查看日志确认运行
npm run cloudflared:logs
```

## ✅ 验证设置

访问您配置的域名：
- 🌐 前端：`https://example.com`
- 🔌 API：`https://api.example.com`

## 常用命令

| 命令 | 说明 |
|------|------|
| `npm run cloudflared:setup` | 安装 cloudflared |
| `npm run cloudflared:start` | 启动隧道 |
| `npm run cloudflared:stop` | 停止隧道 |
| `npm run cloudflared:logs` | 查看日志 |
| `docker compose restart cloudflared` | 重启隧道 |

## 简单示例配置

### 最小配置（单域名）

```yaml
tunnel: 6ff42ae2-765d-4adf-8112-31c55c1551ef
credentials-file: /etc/cloudflared/6ff42ae2-765d-4adf-8112-31c55c1551ef.json

ingress:
  - hostname: myapp.com
    service: http://host.docker.internal:3000
  - service: http_status:404
```

### 开发环境配置

```yaml
tunnel: dev-tunnel-id
credentials-file: /etc/cloudflared/dev-tunnel.json

ingress:
  # 所有流量转发到前端
  - hostname: dev.myapp.com
    service: http://host.docker.internal:3000
  # API 路径
  - hostname: dev.myapp.com
    path: /api/*
    service: http://host.docker.internal:3002
  - service: http_status:404
```

### 生产环境配置

```yaml
tunnel: prod-tunnel-id
credentials-file: /etc/cloudflared/prod-tunnel.json
loglevel: info

ingress:
  # 主站点
  - hostname: myapp.com
    service: http://kimbap-console-app:3000
  # API
  - hostname: api.myapp.com
    service: http://kimbap-console-app:3002
  # 管理面板（需要身份验证）
  - hostname: admin.myapp.com
    service: http://kimbap-adminer:8080
  - service: http_status:404
```

## 🔧 故障排查

### 问题：隧道无法启动

```bash
# 检查 Docker 网络
docker network ls | grep kimbap

# 验证配置文件
docker run --rm -v $(pwd)/cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest tunnel ingress validate
```

### 问题：502 错误

```bash
# 确认服务正在运行
docker ps | grep kimbap-console

# 测试本地连接
curl http://localhost:3000
```

### 问题：DNS 不解析

```bash
# 检查 DNS 记录
dig your-domain.com

# 等待 DNS 传播（可能需要几分钟）
```

## 🔒 安全提醒

1. **不要提交凭据文件到 Git**
   
   添加到 `.gitignore`：
   ```
   cloudflared/*.json
   cloudflared/cert.pem
   ```

2. **使用环境变量存储敏感信息**
   
   在 `.env` 文件中：
   ```env
   CLOUDFLARE_TUNNEL_TOKEN=your_token_here
   ```

3. **定期轮换隧道凭据**
   ```bash
   docker run -v ~/.cloudflared:/etc/cloudflared \
     cloudflare/cloudflared:latest tunnel delete old-tunnel
   ```

## 📚 更多信息

- 详细配置说明：[CLOUDFLARED_SETUP.md](./CLOUDFLARED_SETUP.md)
- 官方文档：[Cloudflare Tunnel Docs](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)
- 视频教程：[YouTube - Cloudflare Tunnel Setup](https://www.youtube.com/results?search_query=cloudflare+tunnel+docker)

## 需要帮助？

- 📝 查看详细日志：`npm run cloudflared:logs`
- 💬 [Cloudflare 社区论坛](https://community.cloudflare.com/)
- 🐛 [提交 Issue](https://github.com/your-repo/issues)