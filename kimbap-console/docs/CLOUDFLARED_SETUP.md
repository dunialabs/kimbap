# Cloudflared Docker 配置和使用指南

## 概述

Cloudflared 是 Cloudflare Tunnel 的命令行客户端，允许您将本地服务安全地暴露到互联网，无需开放防火墙端口或配置复杂的网络设置。本指南将帮助您在 Docker 环境中设置和使用 cloudflared。

## 目录

- [快速开始](#快速开始)
- [详细配置步骤](#详细配置步骤)
- [配置文件说明](#配置文件说明)
- [常用命令](#常用命令)
- [故障排查](#故障排查)
- [安全建议](#安全建议)

## 快速开始

### 1. 自动安装

运行以下命令自动检查并安装 cloudflared：

```bash
npm run cloudflared:setup
```

或者在启动开发环境时自动安装：

```bash
npm run dev
```

### 2. 配置隧道

编辑 `./cloudflared/config.yml` 文件，添加您的隧道配置。

### 3. 启动服务

```bash
docker compose up -d cloudflared
```

## 详细配置步骤

### 步骤 1：创建 Cloudflare 隧道

首先，您需要在本地创建一个 Cloudflare 隧道：

```bash
# 登录到 Cloudflare
docker run -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel login

# 创建隧道
docker run -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel create kimbap-console

# 列出隧道以获取隧道 ID
docker run -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel list
```

### 步骤 2：配置 DNS 路由

将您的域名路由到隧道：

```bash
# 路由主域名
docker run -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel route dns kimbap-console your-domain.com

# 路由 API 子域名
docker run -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel route dns kimbap-console api.your-domain.com
```

### 步骤 3：配置隧道

创建或编辑 `./cloudflared/config.yml` 文件：

```yaml
# Cloudflare Tunnel 配置
tunnel: YOUR_TUNNEL_ID  # 替换为您的隧道 ID
credentials-file: /etc/cloudflared/YOUR_TUNNEL_ID.json

# 入口规则
ingress:
  # 前端应用
  - hostname: your-domain.com
    service: http://kimbap-console-app:3000
    originRequest:
      noTLSVerify: true
      connectTimeout: 30s
  
  # 后端 API
  - hostname: api.your-domain.com
    service: http://kimbap-console-app:3002
    originRequest:
      noTLSVerify: true
      connectTimeout: 30s
  
  # WebSocket 支持（如果需要）
  - hostname: ws.your-domain.com
    service: ws://kimbap-console-app:3002
    originRequest:
      noTLSVerify: true
  
  # 404 规则（必须放在最后）
  - service: http_status:404
```

### 步骤 4：复制凭据文件

将隧道凭据文件复制到项目目录：

```bash
# 找到凭据文件（通常在 ~/.cloudflared/ 目录下）
cp ~/.cloudflared/YOUR_TUNNEL_ID.json ./cloudflared/
```

### 步骤 5：设置环境变量（可选）

如果使用 Token 认证，在 `.env` 文件中添加：

```env
CLOUDFLARE_TUNNEL_TOKEN=your_tunnel_token_here
```

## 配置文件说明

### config.yml 完整示例

```yaml
# 基本配置
tunnel: 6ff42ae2-765d-4adf-8112-31c55c1551ef
credentials-file: /etc/cloudflared/credentials.json

# 日志配置
loglevel: info
logfile: /var/log/cloudflared.log

# 性能优化
protocol: quic  # 使用 QUIC 协议以获得更好的性能
retries: 5      # 重试次数

# 健康检查
metrics: localhost:2000  # Prometheus 指标端口

# 入口规则
ingress:
  # 主站点 - Next.js 前端
  - hostname: example.com
    service: http://kimbap-console-app:3000
    originRequest:
      httpHostHeader: example.com
      originServerName: example.com
      noTLSVerify: true
      connectTimeout: 30s
      tcpKeepAlive: 30s
      keepAliveConnections: 100
      keepAliveTimeout: 90s
  
  # API 端点 - Express 后端
  - hostname: api.example.com
    service: http://kimbap-console-app:3002
    originRequest:
      noTLSVerify: true
      connectTimeout: 60s
  
  # 数据库管理界面（可选）
  - hostname: adminer.example.com
    service: http://kimbap-adminer:8080
    originRequest:
      noTLSVerify: true
      # 添加访问控制
      access:
        required: true
        teamName: "your-team"
  
  # 静态文件优化
  - hostname: static.example.com
    service: http://kimbap-console-app:3000
    originRequest:
      noTLSVerify: true
      # 缓存静态资源
      cacheEverything: true
      cacheTTL: 86400
  
  # 默认规则
  - service: http_status:404
```

### Docker Compose 集成

在 `docker-compose.yml` 中的 cloudflared 服务配置：

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
    - ./cloudflared:/etc/cloudflared:ro
  depends_on:
    - kimbap-console
  healthcheck:
    test: ["CMD", "cloudflared", "tunnel", "info"]
    interval: 30s
    timeout: 10s
    retries: 3
```

## 常用命令

### 管理命令

```bash
# 设置 cloudflared
npm run cloudflared:setup

# 启动 cloudflared
npm run cloudflared:start
# 或
docker compose up -d cloudflared

# 停止 cloudflared
npm run cloudflared:stop
# 或
docker compose stop cloudflared

# 查看日志
npm run cloudflared:logs
# 或
docker compose logs -f cloudflared

# 重启服务
docker compose restart cloudflared

# 检查隧道状态
docker exec kimbap-cloudflared cloudflared tunnel info

# 列出活动连接
docker exec kimbap-cloudflared cloudflared tunnel list
```

### 调试命令

```bash
# 验证配置文件
docker run --rm -v $(pwd)/cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel ingress validate

# 测试隧道规则
docker run --rm -v $(pwd)/cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel ingress rule https://your-domain.com

# 查看指标
curl http://localhost:2000/metrics
```

## 故障排查

### 常见问题

#### 1. 隧道无法连接

**症状**：日志显示 "failed to connect to tunnel"

**解决方案**：
```bash
# 检查凭据文件是否存在
ls -la ./cloudflared/

# 验证隧道 ID
docker exec kimbap-cloudflared cloudflared tunnel list

# 检查网络连接
docker network inspect kimbap-network
```

#### 2. 502 Bad Gateway 错误

**症状**：通过隧道访问时返回 502 错误

**解决方案**：
```bash
# 检查目标服务是否运行
docker ps | grep kimbap-console-app

# 测试内部连接
docker exec kimbap-cloudflared curl http://kimbap-console-app:3000

# 检查服务名称解析
docker exec kimbap-cloudflared nslookup kimbap-console-app
```

#### 3. 配置文件错误

**症状**：服务启动失败，提示配置错误

**解决方案**：
```bash
# 验证 YAML 语法
docker run --rm -v $(pwd)/cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel ingress validate

# 检查文件权限
chmod 600 ./cloudflared/*.json
chmod 644 ./cloudflared/config.yml
```

#### 4. DNS 解析问题

**症状**：域名无法解析到隧道

**解决方案**：
```bash
# 检查 DNS 记录
dig your-domain.com
nslookup your-domain.com

# 重新配置路由
docker run -v ~/.cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel route dns --overwrite-dns kimbap-console your-domain.com
```

### 日志级别

调整日志级别以获取更多调试信息：

```yaml
# 在 config.yml 中设置
loglevel: debug  # 选项：trace, debug, info, warn, error, fatal
```

或通过命令行参数：

```bash
docker run -d \
  --name kimbap-cloudflared \
  -v $(pwd)/cloudflared:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel --loglevel debug run
```

## 安全建议

### 1. 凭据保护

- **不要** 将凭据文件提交到版本控制
- 添加到 `.gitignore`：
  ```gitignore
  cloudflared/*.json
  cloudflared/cert.pem
  .env
  ```

### 2. 访问控制

使用 Cloudflare Access 添加身份验证：

```yaml
ingress:
  - hostname: admin.your-domain.com
    service: http://kimbap-adminer:8080
    originRequest:
      access:
        required: true
        teamName: "your-team"
        audTag: "admin-access"
```

### 3. 环境隔离

为不同环境使用不同的隧道：

```bash
# 开发环境
docker run -v ./cloudflared/dev:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel create kimbap-console-dev

# 生产环境
docker run -v ./cloudflared/prod:/etc/cloudflared \
  cloudflare/cloudflared:latest \
  tunnel create kimbap-console-prod
```

### 4. 监控和告警

启用指标收集：

```yaml
# config.yml
metrics: 0.0.0.0:2000

# docker-compose.yml
cloudflared:
  ports:
    - "2000:2000"  # Prometheus metrics
```

配置 Prometheus 抓取：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'cloudflared'
    static_configs:
      - targets: ['cloudflared:2000']
```

## 高级配置

### 负载均衡

使用多个隧道实例实现负载均衡：

```yaml
# docker-compose.yml
cloudflared-1:
  extends:
    service: cloudflared
  container_name: kimbap-cloudflared-1

cloudflared-2:
  extends:
    service: cloudflared
  container_name: kimbap-cloudflared-2
```

### 自定义健康检查

```yaml
ingress:
  - hostname: health.your-domain.com
    service: http://kimbap-console-app:3000/api/health
    originRequest:
      noTLSVerify: true
      httpHostHeader: localhost
```

### WebSocket 支持

```yaml
ingress:
  - hostname: ws.your-domain.com
    service: ws://kimbap-console-app:3002
    originRequest:
      noTLSVerify: true
      proxyType: websocket
```

## 相关资源

- [Cloudflare Tunnel 官方文档](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)
- [Cloudflared GitHub 仓库](https://github.com/cloudflare/cloudflared)
- [Cloudflare Access 文档](https://developers.cloudflare.com/cloudflare-one/policies/access/)
- [故障排查指南](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/troubleshooting/)

## 支持

如遇到问题，请：

1. 查看 cloudflared 日志：`npm run cloudflared:logs`
2. 检查 [故障排查](#故障排查) 章节
3. 访问 [Cloudflare 社区论坛](https://community.cloudflare.com/)
4. 提交问题到项目的 GitHub Issues