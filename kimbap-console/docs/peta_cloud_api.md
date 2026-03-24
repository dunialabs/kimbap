# Kimbap Cloud API 调用说明文档

本文档描述了 Kimbap Cloud 应用提供的所有 API 接口的使用方法。

## 基础信息

- **Base URL**: `https://your-worker-domain.workers.dev`
- **Content-Type**: `application/json`
- **响应格式**: JSON

---

## 1. 创建 Cloudflare Tunnel

### 接口信息
- **URL**: `POST /tunnel/create`
- **功能**: 创建 Cloudflare Tunnel 并自动配置 DNS 记录
- **用途**: 将本地应用通过安全隧道暴露到公网

### 请求参数
```json
{
  "appId": "string"      // 应用标识符（必需）
}
```

#### 参数说明
- `appId`: 应用的唯一标识符，用于生成子域名
  - ≤ 8 位：使用完整 appId
  - > 8 位：取前 8 位
  - 最终格式：`appId.mcpsub.kimbap.io`

### 请求示例
```bash
curl -X POST https://your-worker-domain.workers.dev/tunnel/create \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "myapp"
  }'
```

### 成功响应 (200)
```json
{
  "tunnelId": "12345678-1234-1234-1234-123456789abc",
  "subdomain": "myapp.mcpsub.kimbap.io",
  "credentials": {
    "AccountTag": "your-account-id",
    "TunnelSecret": "base64-encoded-secret",
    "TunnelID": "12345678-1234-1234-1234-123456789abc"
  }
}
```

#### 响应字段说明
- `tunnelId`: Cloudflare Tunnel 的唯一标识符
- `subdomain`: 自动生成的完整子域名
- `credentials`: 隧道凭据对象
  - `AccountTag`: Cloudflare 账户标签
  - `TunnelSecret`: 隧道密钥（base64 编码）
  - `TunnelID`: 隧道 ID

### 错误响应

#### 400 - 参数错误
```json
{
  "error": "appId is required"
}
```

#### 500 - 服务器错误
```json
{
  "error": "Failed to create tunnel"
}
```

---

## 2. 获取 Tunnel 凭据

### 接口信息
- **URL**: `POST /tunnel/credentials`
- **功能**: 根据 tunnelId 获取隧道的完整配置和凭据
- **用途**: 重新获取隧道凭据或验证隧道状态

### 请求参数
```json
{
  "tunnelId": "string"  // 隧道ID（必需）
}
```

#### 参数说明
- `tunnelId`: Cloudflare Tunnel 的唯一标识符

### 请求示例
```bash
curl -X POST https://your-worker-domain.workers.dev/tunnel/credentials \
  -H "Content-Type: application/json" \
  -d '{
    "tunnelId": "12345678-1234-1234-1234-123456789abc"
  }'
```

### 成功响应 (200)
```json
{
  "tunnelId": "12345678-1234-1234-1234-123456789abc",
  "name": "tunnel-myapp-1234567890",
  "subdomain": "myapp.mcpsub.kimbap.io",
  "credentials": {
    "AccountTag": "your-account-id",
    "TunnelSecret": "base64-encoded-secret",
    "TunnelID": "12345678-1234-1234-1234-123456789abc"
  },
  "created_at": "2023-01-01T12:00:00Z"
}
```

#### 响应字段说明
- `tunnelId`: 隧道的唯一标识符
- `name`: 隧道名称
- `subdomain`: 关联的域名（如果找不到则为 null）
- `credentials`: 隧道凭据对象
- `created_at`: 隧道创建时间

### 错误响应

#### 400 - 参数错误
```json
{
  "error": "tunnelId is required"
}
```

#### 500 - 服务器错误
```json
{
  "error": "Failed to get tunnel credentials"
}
```

---

## 3. 删除 Tunnel

### 接口信息
- **URL**: `POST /tunnel/delete`
- **功能**: 删除指定的 Cloudflare Tunnel 及其关联的 DNS 记录
- **用途**: 清理不再使用的隧道资源

### 请求参数
```json
{
  "tunnelId": "string"  // 隧道ID（必需）
}
```

#### 参数说明
- `tunnelId`: 要删除的 Cloudflare Tunnel 的唯一标识符

### 请求示例
```bash
curl -X POST https://your-worker-domain.workers.dev/tunnel/delete \
  -H "Content-Type: application/json" \
  -d '{
    "tunnelId": "12345678-1234-1234-1234-123456789abc"
  }'
```

### 成功响应 (200)
```json
{
  "success": true,
  "message": "Tunnel and associated DNS records deleted successfully",
  "tunnelId": "12345678-1234-1234-1234-123456789abc",
  "deletedDnsRecords": [
    "myapp.mcpsub",
    "another-app"
  ]
}
```

#### 响应字段说明
- `success`: 删除操作是否成功
- `message`: 操作结果描述
- `tunnelId`: 被删除的隧道 ID
- `deletedDnsRecords`: 被删除的 DNS 记录名称列表

### 错误响应

#### 400 - 参数错误
```json
{
  "error": "tunnelId is required"
}
```

#### 500 - 服务器错误
```json
{
  "error": "Failed to delete tunnel"
}
```

### 注意事项
- 删除操作不可逆，请谨慎操作
- 会自动删除所有关联的 DNS CNAME 记录
- 如果隧道正在使用中，删除后相关服务将无法访问

---

## 4. 获取工具模板列表

### 接口信息
- **URL**: `POST /tool/templates`
- **功能**: 查询所有激活状态的工具模板
- **用途**: 获取可用的工具配置模板列表

### 请求参数
无需请求体参数。

### 请求示例
```bash
curl -X POST https://your-worker-domain.workers.dev/tool/templates \
  -H "Content-Type: application/json"
```

### 成功响应 (200)
```json
[
  {
    "toolId": "tool_001",
    "toolType": "api",
    "name": "GitHub API Tool",
    "description": "Tool for interacting with GitHub API",
    "tags": "git,api,development",
    "authtags": "oauth,token",
    "credentials": "{\"token_type\": \"bearer\"}",
    "mcpJsonConf": "{\"servers\": [{\"name\": \"github\"}]}"
  },
  {
    "toolId": "tool_002",
    "toolType": "database",
    "name": "MySQL Connector",
    "description": "Database connection tool for MySQL",
    "tags": "database,mysql,sql",
    "authtags": "username,password",
    "credentials": "{\"host\": \"localhost\"}",
    "mcpJsonConf": "{\"config\": {\"port\": 3306}}"
  }
]
```

#### 响应字段说明
- `toolId`: 工具的唯一标识符
- `toolType`: 工具类型（如 api、database 等）
- `name`: 工具名称
- `description`: 工具描述
- `tags`: 工具标签（逗号分隔）
- `authtags`: 认证标签（逗号分隔）
- `credentials`: 凭据配置（JSON 字符串）
- `mcpJsonConf`: MCP JSON 配置（JSON 字符串）

### 无数据响应 (404)
```json
{
  "message": "No records found"
}
```

### 错误响应 (500)
```json
{
  "error": "Failed to query database",
  "details": "具体错误信息"
}
```

---

## 5. 健康检查接口

### 接口信息
- **URL**: `GET /message`
- **功能**: 简单的健康检查接口
- **用途**: 检查服务是否正常运行

### 请求示例
```bash
curl https://your-worker-domain.workers.dev/message
```

### 响应 (200)
```
Hello Hono!
```

---

## 配置要求

### 环境变量
在 `wrangler.jsonc` 中需要配置以下环境变量：

```json
{
  "vars": {
    "CF_ZONE_ID": "your-zone-id",
    "CF_API_TOKEN": "your-api-token",
    "CLOUDFLARE_ACCOUNT_ID": "your-account-id",
    "CLOUDFLARE_API_TOKEN": "your-tunnel-api-token",
    "YOUR_DOMAIN": "kimbap.io"
  }
}
```

#### 变量说明
- `CF_ZONE_ID`: Cloudflare Zone ID（用于 DNS 管理）
- `CF_API_TOKEN`: Cloudflare API Token（DNS 权限）
- `CLOUDFLARE_ACCOUNT_ID`: Cloudflare 账户 ID（Tunnel 权限）
- `CLOUDFLARE_API_TOKEN`: Cloudflare API Token（Tunnel 权限）
- `YOUR_DOMAIN`: 你的根域名

### 数据库表结构

#### tool_template 表
```sql
CREATE TABLE tool_template (
  tool_id TEXT PRIMARY KEY,
  tool_type TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  tags TEXT,
  authtags TEXT,
  credentials TEXT,
  mcp_json_conf TEXT,
  state INTEGER NOT NULL DEFAULT 1
);
```

---

## 错误处理

### 通用错误码
- `400`: 请求参数错误
- `404`: 资源未找到
- `500`: 服务器内部错误

### 错误响应格式
```json
{
  "error": "错误描述",
  "details": "详细错误信息（可选）"
}
```

---

## 使用场景

### Tunnel 管理流程

#### 创建隧道
1. 客户端调用 `/tunnel/create` 接口
2. 系统创建 Cloudflare Tunnel
3. 自动配置 DNS CNAME 记录
4. 返回隧道配置给客户端
5. 客户端使用返回的凭据启动本地隧道

#### 查询隧道
1. 客户端调用 `/tunnel/credentials` 接口
2. 系统查询隧道信息和 DNS 记录
3. 返回完整的隧道配置和域名

#### 删除隧道
1. 客户端调用 `/tunnel/delete` 接口
2. 系统先删除关联的 DNS 记录
3. 再删除 Cloudflare Tunnel
4. 返回删除结果和清理的资源列表

### 工具模板查询流程
1. 客户端调用 `/tool/templates` 接口
2. 系统查询数据库中激活的模板
3. 返回格式化的工具列表
4. 客户端使用模板配置工具

---

## 安全注意事项

1. **API Token 安全**: 确保 Cloudflare API Token 具有最小权限
2. **域名验证**: 生成的子域名会自动指向你的域名
3. **隧道管理**: 定期清理不使用的隧道资源
4. **数据库安全**: 敏感配置信息应加密存储

---

## 开发调试

### 本地开发
```bash
npm run dev
```

### 部署到 Cloudflare Workers
```bash
npm run deploy
```

### 查看日志
```bash
wrangler tail
```

---

## API 接口总览

| 接口路径 | 方法 | 功能描述 | 
|---------|------|---------|
| `/tunnel/create` | POST | 创建 Cloudflare Tunnel 并配置 DNS |
| `/tunnel/credentials` | POST | 获取隧道凭据和配置信息 |
| `/tunnel/delete` | POST | 删除隧道及其关联资源 |
| `/tool/templates` | POST | 获取工具模板列表 |
| `/message` | GET | 健康检查 |

## 更新记录

- **v1.1**: 新增 Tunnel 查询和删除功能
- **v1.0**: 初始版本，支持 Tunnel 创建和工具模板查询
- 更多功能持续开发中...