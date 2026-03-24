# 如何获取 Cloudflare Tunnel Token

## 📍 获取位置

**Cloudflare Dashboard**: https://dash.cloudflare.com/

## 🚀 快速步骤（3 步）

### 步骤 1：登录并进入 Zero Trust

1. 访问 https://dash.cloudflare.com/
2. 登录你的 Cloudflare 账户（免费账户即可）
3. 在左侧菜单找到 **Zero Trust**（如果没有，点击 "Add a product" 添加）
4. 进入 **Networks** → **Tunnels**

### 步骤 2：创建隧道

1. 点击 **"Create a tunnel"** 按钮
2. 选择 **"Cloudflared"** 类型
3. 输入隧道名称：`kimbap-console`（或任意名称）
4. 点击 **"Save tunnel"**

### 步骤 3：配置路由并获取 Token

#### 方式 A：使用 Tunnel Token（推荐）

1. 创建隧道后，在隧道详情页找到 **"Configure"** 按钮
2. 点击 **"Configure"**
3. 添加 **Public Hostname**：
   - **Subdomain**: `console`（或你想要的子域名）
   - **Domain**: 选择你的域名（如 `example.com`）
   - **Service**: `http://kimbap-console:3000`
   - 点击 **"Save hostname"**
4. 如果需要 API 子域名，再添加一个：
   - **Subdomain**: `api`
   - **Domain**: 选择你的域名
   - **Service**: `http://kimbap-core:3002`
   - 点击 **"Save hostname"**
5. 在隧道详情页，找到 **"Tunnel token"** 或 **"Token"**
6. 点击 **"Copy token"** 复制 Token

#### 方式 B：使用配置文件（高级）

如果你更喜欢使用配置文件，参考 [CLOUDFLARED_QUICK_START.md](./CLOUDFLARED_QUICK_START.md)

## 📝 Token 格式

Token 格式类似：
```
eyJhIjoiY2xvdWRmbGFyZS10dW5uZWwtdG9rZW4tZXhhbXBsZSIsInQiOiJ0dW5uZWwtdG9rZW4ifQ==
```

这是一个很长的字符串（通常 200+ 字符）。

## ✅ 配置到项目

### 方法 1：环境变量

```bash
export CLOUDFLARE_TUNNEL_TOKEN="你的token"
```

### 方法 2：.env 文件

编辑 `.env` 文件，添加：

```bash
CLOUDFLARE_TUNNEL_TOKEN=你的token
```

### 方法 3：docker-compose 命令

```bash
CLOUDFLARE_TUNNEL_TOKEN="你的token" docker compose up -d
```

## 🔍 验证配置

配置完成后，重启服务：

```bash
docker compose restart cloudflared
```

查看日志确认连接成功：

```bash
docker compose logs -f cloudflared
```

如果看到类似以下输出，说明连接成功：
```
INF Connection established
INF +--------------------------------------------------------------------------------------------+
INF |  Your quick Tunnel has been created! Visit it:                                             |
INF |  https://console.yourdomain.com                                                             |
INF +--------------------------------------------------------------------------------------------+
```

## 🆘 常见问题

### Q: 找不到 Zero Trust？

**A**: 
- 免费账户需要手动添加 Zero Trust
- 访问：https://one.dash.cloudflare.com/
- 或点击 Dashboard 右上角 "Add a product" → 找到 "Zero Trust"

### Q: Token 在哪里？

**A**: 
- 在隧道详情页
- 找到 "Tunnel token" 或 "Token" 字段
- 点击复制按钮

### Q: 如何更新 Token？

**A**: 
- 在 Cloudflare Dashboard 中删除旧隧道
- 创建新隧道，获取新 Token
- 更新 `.env` 文件中的 `CLOUDFLARE_TUNNEL_TOKEN`
- 重启服务

### Q: 域名如何配置？

**A**: 
- 在 Cloudflare Dashboard 的隧道配置中
- 添加 Public Hostname
- 选择你的域名和子域名
- 设置 Service 为 `http://kimbap-console:3000`

## 📚 更多信息

- [HTTPS 远程部署指南](./HTTPS_REMOTE_DEPLOYMENT.md)
- [Cloudflare Tunnel 快速开始](./CLOUDFLARED_QUICK_START.md)
- [官方文档](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)

