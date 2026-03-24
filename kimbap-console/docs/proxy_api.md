# MCP Gateway 管理API协议文档

## 概述

本文档描述了 MCP Gateway 管理API的完整协议规范，包括请求格式、响应格式、错误处理等。所有接口都通过统一的 `/admin` 端点提供服务。

## 基础信息

- **接口地址**: `POST /admin`
- **认证方式**: Bearer Token
- **内容类型**: `application/json`
- **字符编码**: UTF-8

## 通用数据结构

### 1. 请求基础结构

```typescript
interface AdminRequest<T = any> {
  action: AdminActionType;  // 操作类型
  data: T;                  // 操作数据
}
```

### 2. 响应基础结构

```typescript
interface AdminResponse<T = any> {
  success: boolean;         // 操作是否成功
  data?: T;                // 成功时的返回数据
  error?: {                // 失败时的错误信息
    code: number;          // 错误码
    message: string;       // 错误消息
  };
}
```

### 3. 通用标识符

```typescript
interface TargetIdentifier {
  targetId: string;        // 目标用户ID或服务器ID
}

export type McpServerCapabilities = {
  [serverID: string]: {
    enabled: boolean;
    tools: { [toolName: string]: { enabled: boolean, description?: string, dangerLevel?: number } };
    resources: { [resourceName: string]: { enabled: boolean, description?: string } };
    prompts: { [promptName: string]: { enabled: boolean, description?: string } };
  };
};

export type ServerConfigCapabilities = {
  tools: { [toolName: string]: { enabled: boolean, description?: string, dangerLevel?: number } };
  resources: { [resourceName: string]: { enabled: boolean, description?: string } };
  prompts: { [promptName: string]: { enabled: boolean, description?: string } };
};

export class ServerEntity {
  /**
   * server唯一ID
   */
  @PrimaryColumn({ type: "varchar", length: 128 })
  server_id!: string;

    /**
   * server名称
   */
  @Column({ type: "varchar", length: 128 })
  server_name!: string;

  /**
   * 是否启用（true: 启用, false: 关闭）
   */
  @Column({ type: "boolean", default: true })
  enabled!: boolean;

  /**
   * 启动参数（加密后的JSON字符串，包含command/args/env等）
   */
  @Column({ type: "text"})
  launch_config!: string;

  /**
   * 能力描述（JSON，包含tools/resources/prompts等）
   */
  @Column({ type: "simple-json" })
  capabilities!: Record<string, any>;

  /**
   * 创建时间
   */
  @CreateDateColumn({ type: "integer" })
  created_at!: number;

  /**
   * 更新时间
   */
  @UpdateDateColumn({ type: "integer" })
  updated_at!: number;

  /**
   * 允许用户通过Kimbap MCP Desk输入（true: 允许, false: 不允许）
   */
  @Column({ type: "boolean", default: false })
  allow_user_input!: boolean;
}
```

## 操作类型枚举

```typescript
enum AdminActionType {
  // 用户操作 (1000-1999)
  DISABLE_USER = 1001,                    // 关闭指定用户的访问权限
  UPDATE_USER_PERMISSIONS = 1002,         // 更新用户权限

  // 服务器操作 (2000-2999)
  START_SERVER = 2001,                    // 启动指定服务器
  STOP_SERVER = 2002,                     // 关闭指定服务器
  UPDATE_SERVER_CAPABILITIES = 2003,      // 更新服务器能力配置
  UPDATE_SERVER_LAUNCH_CMD = 2004,        // 更新启动命令
  CONNECT_ALL_SERVERS = 2005,             // 连接所有服务器

  // 查询操作 (3000-3999)
  GET_AVAILABLE_SERVERS_CAPABILITIES = 3002, // 获取所有服务器能力配置
  GET_USER_AVAILABLE_SERVERS_CAPABILITIES = 3003, // 获取用户可访问服务器能力配置
  GET_SERVERS_STATUS = 3004,              // 获取所有服务器状态
  GET_SERVERS_CAPABILITIES = 3005,        // 获取指定服务器能力配置

  // 安全配置操作 (4000-4999)
  UPDATE_IP_WHITELIST = 4001              // 更新IP白名单
}
```

## 错误码定义

```typescript
enum AdminErrorCode {
  // 通用错误 (1000-1999)
  INVALID_REQUEST = 1001,        // 无效请求格式
  UNAUTHORIZED = 1002,           // 未授权
  FORBIDDEN = 1003,              // 禁止访问

  // 用户相关错误 (2000-2999)
  USER_NOT_FOUND = 2001,         // 用户未找到
  USER_ALREADY_DISABLED = 2002,  // 用户已被禁用

  // 服务器相关错误 (3000-2999)
  SERVER_NOT_FOUND = 3001,       // 服务器未找到
  SERVER_ALREADY_RUNNING = 3002, // 服务器已在运行

  // 权限相关错误 (4000-4999)
  INSUFFICIENT_PERMISSIONS = 4001, // 权限不足
  INVALID_PERMISSION_FORMAT = 4002  // 权限格式无效
}
```

## API接口详细说明

### 1. 禁用用户 (DISABLE_USER)

**请求格式:**
```json
{
  "action": 1001,
  "data": {
    "targetId": "user123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": null
}
```

**说明:**
- 断开指定用户的所有活跃会话
- 返回 `void`，无具体数据返回

### 2. 更新用户权限 (UPDATE_USER_PERMISSIONS)

**请求格式:**
```json
{
  "action": 1002,
  "data": {
    "targetId": "user123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": null
}
```

**说明:**
- 更新指定用户的权限配置
- 实时生效到所有活跃会话
- 返回 `void`，无具体数据返回

### 3. 启动服务器 (START_SERVER)

**请求格式:**
```json
{
  "action": 2001,
  "data": {
    "targetId": "server123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": null
}
```

**说明:**
- 启动指定的服务器
- 通知相关用户服务器能力变更
- 返回 `void`，无具体数据返回

### 4. 停止服务器 (STOP_SERVER)

**请求格式:**
```json
{
  "action": 2002,
  "data": {
    "targetId": "server123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": null
}
```

**说明:**
- 停止指定的服务器
- 通知相关用户服务器能力变更
- 返回 `void`，无具体数据返回

### 5. 更新服务器能力配置 (UPDATE_SERVER_CAPABILITIES)

**请求格式:**
```json
{
  "action": 2003,
  "data": {
    "targetId": "server123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": null
}
```

**说明:**
- 更新指定服务器的能力配置
- 如果服务器未启用，直接返回
- 返回 `void`，无具体数据返回

### 6. 更新服务器启动命令 (UPDATE_SERVER_LAUNCH_CMD)

**请求格式:**
```json
{
  "action": 2004,
  "data": {
    "targetId": "server123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": null
}
```

**说明:**
- 更新指定服务器的启动命令配置
- 如果服务器未启用，直接返回
- 返回 `void`，无具体数据返回

### 7. 连接所有服务器 (CONNECT_ALL_SERVERS)

**请求格式:**
```json
{
  "action": 2005,
  "data": {}
}
```

**响应格式:**
```json
{
  "success": true,
  "data": {
    "successServers": [ServerEntity, ServerEntity],
    "failedServers": [ServerEntity]
  }
}


```

**说明:**
- 尝试连接所有配置的服务器
- 返回成功和失败的服务器列表
- 服务器对象会被转换为JSON字符串

### 8. 获取所有服务器能力配置 (GET_AVAILABLE_SERVERS_CAPABILITIES)

**请求格式:**
```json
{
  "action": 3002,
  "data": {}
}
```

**响应格式:**
```json
{
  "success": true,
  "data": {
    "capabilities": {
      "server1": {
        "enabled": true;
        "tools": {"toolName": {"enabled": true, "description": "Tool description", "dangerLevel": 0}},
        "resources": {"resourceName": {"enabled": true, "description": "Resource description"}},
        "prompts": {"promptName": {"enabled": true, "description": "Prompt description"}}
      }
    }
  }
}
```

**说明:**
- 获取所有可用服务器的能力配置
- 返回完整的服务器能力信息

### 9. 获取用户可用服务器能力配置 (GET_USER_AVAILABLE_SERVERS_CAPABILITIES)

**请求格式:**
```json
{
  "action": 3003,
  "data": {
    "targetId": "user123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": {
    "capabilities": {
      "server1": {
        "enabled": true;
        "tools": {"toolName": {"enabled": true, "description": "Tool description", "dangerLevel": 0}},
        "resources": {"resourceName": {"enabled": true, "description": "Resource description"}},
        "prompts": {"promptName": {"enabled": true, "description": "Prompt description"}}
      }
    }
  }
}
```

**说明:**
- 获取指定用户可访问的服务器能力配置
- 优先从用户会话获取，如果没有则从用户权限获取

### 10. 获取所有服务器状态 (GET_SERVERS_STATUS)

**请求格式:**
```json
{
  "action": 3004,
  "data": {}
}
```

**响应格式:**
```json
{
  "success": true,
  "data": {
    "serversStatus": {
      "server1": 1,
      "server2": 2
    }
  }
}

export enum ServerStatus {
  Online = 0,   // 在线
  Offline = 1,  // 离线
  Connecting = 2, // 连接中
  Error = 3,    // 异常
}

export enum DangerLevel {
  Silent = 0,        // Execute Silently - Function executes automatically without user notification
  Notification = 1,  // Execute with Notification - Function executes automatically and displays result to user
  Approval = 2       // Require Manual Approval - User must manually approve before function execution
}
```

**说明:**
- 获取所有服务器的当前状态
- 状态值包括：Online、Offline等

### 11. 获取指定服务器能力配置 (GET_SERVERS_CAPABILITIES)

**请求格式:**
```json
{
  "action": 3005,
  "data": {
    "targetId": "server123"
  }
}
```

**响应格式:**
```json
{
  "success": true,
  "data": {
    "capabilities": {
      "tools": {"toolName": {"enabled": true, "description": "Tool description", "dangerLevel": 0}},
      "resources": {"resourceName": {"enabled": true, "description": "Resource description"}},
      "prompts": {"promptName": {"enabled": true, "description": "Prompt description"}}
    }
  }
}
```

**说明:**
- 获取指定服务器的能力配置
- 如果服务器未连接，返回数据库中的配置

### 12. 更新IP白名单 (UPDATE_IP_WHITELIST)

**请求格式:**
```json
{
  "action": 4001,
  "data": {}
}
```

**响应格式:**
```json
{
  "success": true,
  "data": {
    "message": "IP whitelist reloaded successfully",
    "count": 5,
    "enabled": true
  }
}
```

**说明:**
- 通知MCP网关从数据库重新加载IP访问白名单
- 请求的`data`字段为空对象，不需要传递白名单数据
- 系统会从数据库的`ip_whitelist`表中读取最新的白名单配置
- 响应中包含：
  - `count`: 加载的白名单条目数量
  - `enabled`: 白名单是否启用（false表示包含"0.0.0.0/0"）

**白名单管理说明:**
- 白名单数据应通过其他管理接口预先写入数据库
- 支持的IP格式：
  - 单个IP地址：`"192.168.1.100"`
  - CIDR格式：`"192.168.1.0/24"`
  - 特殊值：`"0.0.0.0/0"` 表示允许所有IP访问（相当于禁用IP白名单）
- 判断逻辑：
  - 如果白名单包含 `"0.0.0.0/0"`，则允许所有IP访问
  - 否则，只允许白名单中的IP或IP段访问
- IP白名单在 `/mcp` 端点生效，在认证之前进行检查
- 被拒绝的访问会返回 403 Forbidden 错误
- 配置更新后立即生效
- **注意**: 当前实现仅在内存中保存配置，服务重启后会恢复默认值 `["0.0.0.0/0"]`

**IP格式验证规则:**
- IPv4地址必须是4个0-255之间的数字，用点分隔
- CIDR格式的掩码位必须是0-32之间的数字
- 系统会自动处理IPv6映射的IPv4地址（如 `::ffff:192.168.1.1`）

**示例:**
```bash
# 限制只允许特定网段访问
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -d '{
    "action": 4001,
    "data": {
      "whitelist": ["192.168.1.0/24", "10.0.0.0/8", "127.0.0.1"]
    }
  }'

# 允许所有IP访问（禁用白名单）
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -d '{
    "action": 4001,
    "data": {
      "whitelist": ["0.0.0.0/0"]
    }
  }'
```

## 错误响应格式

### 1. 请求格式错误

```json
{
  "success": false,
  "error": {
    "code": 1001,
    "message": "Invalid admin request format"
  }
}
```

### 2. 未知操作类型

```json
{
  "success": false,
  "error": {
    "code": 1001,
    "message": "Unknown action type: 9999"
  }
}
```

### 3. 用户未找到

```json
{
  "success": false,
  "error": {
    "code": 2001,
    "message": "User user123 not found"
  }
}
```

### 4. 服务器未找到

```json
{
  "success": false,
  "error": {
    "code": 3001,
    "message": "Server server123 not found"
  }
}
```

### 5. 无效目标标识符

```json
{
  "success": false,
  "error": {
    "code": 1001,
    "message": "Invalid target identifier"
  }
}
```

### 6. 内部服务器错误

```json
{
  "success": false,
  "error": {
    "code": 1001,
    "message": "Internal server error"
  }
}
```

## 请求验证规则

### 1. 基础请求验证

- `action` 字段必须存在且为有效的枚举值
- `data` 字段必须存在（可以为空对象）

### 2. 目标标识符验证

对于需要 `targetId` 的操作，`data` 必须满足：
- 不能为 `null` 或 `undefined`
- 必须是对象类型
- 必须包含 `targetId` 字段
- `targetId` 必须是字符串类型

## HTTP状态码

- **200 OK**: 请求成功
- **400 Bad Request**: 请求格式错误或参数无效
- **500 Internal Server Error**: 服务器内部错误

## 注意事项

1. **认证要求**: 所有请求都需要有效的管理权限Token
2. **参数验证**: 系统会自动验证请求格式和目标标识符
3. **错误处理**: 所有错误都会返回标准化的错误响应格式
4. **数据一致性**: 操作成功后，相关数据会实时同步到内存和用户会话
5. **异步操作**: 某些操作（如服务器启动/停止）是异步的，响应只表示操作已接受

## 示例代码

### JavaScript/TypeScript 客户端示例

```typescript
async function disableUser(userId: string): Promise<void> {
  const response = await fetch('/admin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      action: 1001, // DISABLE_USER
      data: {
        targetId: userId
      }
    })
  });

  const result = await response.json();

  if (!result.success) {
    throw new Error(`Operation failed: ${result.error.message}`);
  }
}

async function getServerStatus(): Promise<{[serverId: string]: string}> {
  const response = await fetch('/admin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      action: 3004, // GET_SERVERS_STATUS
      data: {}
    })
  });

  const result = await response.json();

  if (!result.success) {
    throw new Error(`Operation failed: ${result.error.message}`);
  }

  return result.data.serversStatus;
}
```

### cURL 示例

```bash
# 禁用用户
curl -X POST http://localhost:3000/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "action": 1001,
    "data": {
      "targetId": "user123"
    }
  }'

# 获取服务器状态
curl -X POST http://localhost:3000/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "action": 3004,
    "data": {}
  }'
```

---

# 数据库操作API

## 概述

数据库操作API提供对数据库表的直接CRUD操作和备份恢复功能。所有接口都通过统一的 `/admin` 端点提供服务。

## 基础信息

- **接口地址**: `POST /admin`
- **认证方式**: Bearer Token (需要Owner或Admin权限)
- **内容类型**: `application/json`
- **字符编码**: UTF-8

## 数据库操作类型枚举(新增)

```typescript
enum AdminActionType {
  // User操作 (1000-1999)
  CREATE_USER = 1010,                     // 创建用户
  GET_USERS = 1011,                       // 查询用户列表
  UPDATE_USER = 1012,                     // 更新用户
  DELETE_USER = 1013,                     // 删除用户
  DELETE_USERS_BY_PROXY = 1014,           // 按proxy批量删除用户
  COUNT_USERS = 1015,                     // 统计用户数量

  // Server操作 (2000-2999)
  CREATE_SERVER = 2010,                   // 创建服务器
  GET_SERVERS = 2011,                     // 查询服务器列表
  UPDATE_SERVER = 2012,                   // 更新服务器
  DELETE_SERVER = 2013,                   // 删除服务器
  DELETE_SERVERS_BY_PROXY = 2014,         // 按proxy批量删除服务器
  COUNT_SERVERS = 2015,                   // 统计服务器数量

  // IpWhitelist操作 (4000-4999)
  UPDATE_IP_WHITELIST = 4001,             // 替换模式：删除所有现有IP，保存新IP列表到数据库并加载到内存
  GET_IP_WHITELIST = 4002,                // 查询IP白名单
  DELETE_IP_WHITELIST = 4003,             // 删除指定IP白名单
  ADD_IP_WHITELIST = 4004,                // 追加模式：新增IP到白名单（不删除现有IP）
  SPECIAL_IP_WHITELIST_OPERATION = 4005,  // IP过滤开关：allow-all关闭过滤/deny-all启用过滤

  // Proxy操作 (5000-5099)
  GET_PROXY = 5001,                       // 查询proxy信息
  CREATE_PROXY = 5002,                    // 创建proxy
  UPDATE_PROXY = 5003,                    // 更新proxy
  DELETE_PROXY = 5004,                    // 删除proxy

  // 备份恢复 (6000-6099)
  BACKUP_DATABASE = 6001,                 // 全量备份数据库
  RESTORE_DATABASE = 6002                 // 全量恢复数据库
}
```

## 新增错误码

```typescript
enum AdminErrorCode {
  // User相关错误
  USER_ALREADY_EXISTS = 2003,

  // Server相关错误
  SERVER_ALREADY_EXISTS = 3003,

  // Proxy相关错误 (5000-5099)
  PROXY_NOT_FOUND = 5001,
  PROXY_ALREADY_EXISTS = 5002,

  // IpWhitelist相关错误 (5100-5199)
  IPWHITELIST_NOT_FOUND = 5101,
  INVALID_IP_FORMAT = 5102,

  // 数据库操作错误 (5200-5299)
  DATABASE_OPERATION_FAILED = 5201,
  TRANSACTION_FAILED = 5202,

  // 备份恢复错误 (5300-5399)
  BACKUP_FAILED = 5301,
  RESTORE_FAILED = 5302,
  INVALID_BACKUP_DATA = 5303
}
```

## User操作API

### 1. 创建用户 (CREATE_USER)

**请求:**
```json
{
  "action": 1010,
  "data": {
    "userId": "string",
    "status": 1,
    "role": 3,
    "permissions": "{}",
    "serverApiKeys": "[]",
    "expiresAt": 0,
    "ratelimit": 10,
    "name": "string",
    "encryptedToken": "string",
    "proxyId": 1,
    "notes": "string"
  }
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "user": { /* User对象 */ }
  }
}
```

### 2. 查询用户列表 (GET_USERS)

**请求:**
```json
{
  "action": 1011,
  "data": {
    "proxyId": 1,          // 可选
    "role": 1,             // 可选
    "excludeRole": 1,      // 可选
    "userId": "string"     // 可选,精确查询
  }
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "users": [/* User对象数组 */]
  }
}
```

### 3. 更新用户 (UPDATE_USER)

**请求:**
```json
{
  "action": 1012,
  "data": {
    "userId": "string",
    "name": "string",      // 可选
    "notes": "string",     // 可选
    "permissions": {},     // 可选
    "serverApiKeys": [],   // 可选
    "status": 1           // 可选
  }
}
```

### 4. 删除用户 (DELETE_USER)

**请求:**
```json
{
  "action": 1013,
  "data": {
    "userId": "string"
  }
}
```

### 5. 按proxy批量删除用户 (DELETE_USERS_BY_PROXY)

**请求:**
```json
{
  "action": 1014,
  "data": {
    "proxyId": 1
  }
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "deletedCount": 10
  }
}
```

### 6. 统计用户数量 (COUNT_USERS)

**请求:**
```json
{
  "action": 1015,
  "data": {
    "excludeRole": 1  // 可选,排除owner
  }
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "count": 50
  }
}
```

## Server操作API

### 7. 创建服务器 (CREATE_SERVER)

**请求:**
```json
{
  "action": 2010,
  "data": {
    "serverId": "string",
    "serverName": "string",
    "enabled": true,
    "launchConfig": {},
    "capabilities": {},
    "allowUserInput": false,
    "proxyId": 1,
    "toolTmplId": "string",
    "authType": 0
  }
}
```

### 8. 查询服务器列表 (GET_SERVERS)

**请求:**
```json
{
  "action": 2011,
  "data": {
    "proxyId": 1,        // 可选
    "enabled": true,     // 可选
    "serverId": "string" // 可选,精确查询
  }
}
```

### 9-12. 更新/删除/批量删除/统计服务器

类似User操作,action分别为: 2012, 2013, 2014, 2015

## IpWhitelist操作API

### 13. 更新IP白名单 (UPDATE_IP_WHITELIST) - 已修改

**请求:**
```json
{
  "action": 4001,
  "data": {
    "whitelist": ["192.168.1.0/24", "10.0.0.1"]
  }
}
```

**说明:**
- 保存IP列表到数据库
- 从数据库加载到内存
- 立即生效

### 14. 查询IP白名单 (GET_IP_WHITELIST)

**请求:**
```json
{
  "action": 4002,
  "data": {}
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "ipWhitelist": [
      {"id": 1, "ip": "192.168.1.0/24", "addtime": 1234567890}
    ]
  }
}
```

### 15. 删除IP白名单 (DELETE_IP_WHITELIST)

**请求:**
```json
{
  "action": 4003,
  "data": {
    "ids": [1, 2, 3],           // 按ID删除
    // 或
    "ips": ["192.168.1.0/24"]   // 按IP删除
  }
}
```

### 16. 添加IP到白名单 (ADD_IP_WHITELIST)

**请求:**
```json
{
  "action": 4004,
  "data": {
    "ips": ["192.168.1.100", "10.0.0.0/8", "172.16.0.0/16"]
  }
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "addedIds": [10, 11, 12],
    "addedCount": 3,
    "skippedCount": 0,
    "message": "3 IP(s) added to whitelist, 0 skipped (duplicates)"
  }
}
```

**说明:**
- **追加模式**：不删除现有IP，只添加新IP到白名单
- 自动跳过已存在的重复IP
- 添加后自动加载到内存，立即生效
- 返回新增记录的ID列表和统计信息
- 验证IP格式（支持单个IP和CIDR格式）

### 17. IP过滤开关操作 (SPECIAL_IP_WHITELIST_OPERATION)

**请求格式 - allow-all (关闭IP过滤):**
```json
{
  "action": 4005,
  "data": {
    "operation": "allow-all"
  }
}
```

**请求格式 - deny-all (启用IP过滤):**
```json
{
  "action": 4005,
  "data": {
    "operation": "deny-all"
  }
}
```

**成功响应:**
```json
{
  "success": true,
  "data": null
}
```

**错误响应（deny-all时无其他IP配置）:**
```json
{
  "success": false,
  "error": {
    "code": 1001,
    "message": "Cannot enable IP filtering: No IP addresses configured in whitelist. Please add specific IPs before enabling filtering."
  }
}
```

**说明:**
- **allow-all（关闭IP过滤）**:
  - 在数据库中添加"0.0.0.0/0"（如已存在则不重复添加）
  - 效果：允许所有IP访问，不再使用数据库中的其他IP配置进行过滤
  - 用途：临时关闭IP限制或测试环境

- **deny-all（启用IP过滤）**:
  - 从数据库中删除所有"0.0.0.0/0"记录
  - 效果：启用IP白名单过滤，只允许数据库中配置的具体IP访问
  - **前置条件**：数据库中必须有其他IP配置，否则返回错误
  - 用途：启用严格的IP访问控制

- 操作后自动加载到内存，立即生效

**使用流程建议:**
1. 先使用 ADD_IP_WHITELIST (4004) 添加允许访问的IP地址
2. 再使用 deny-all 操作启用IP过滤
3. 需要临时关闭过滤时使用 allow-all 操作

## Proxy操作API

### 16. 查询proxy (GET_PROXY)

**请求:**
```json
{
  "action": 5001,
  "data": {}
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "proxy": {
      "id": 1,
      "name": "My MCP Server",
      "proxyKey": "xxx",
      "addtime": 1234567890,
      "startPort": 3002
    }
  }
}
```

### 17-19. 创建/更新/删除proxy

类似操作,action分别为: 5002, 5003, 5004

## 备份恢复API

### 20. 备份数据库 (BACKUP_DATABASE)

**请求:**
```json
{
  "action": 6001,
  "data": {}
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "proxy": [/* Proxy记录数组 */],
    "user": [/* User记录数组 */],
    "server": [/* Server记录数组 */],
    "ip_whitelist": [/* IpWhitelist记录数组 */],
    "backup_time": "2025-10-16T10:30:00Z"
  }
}
```

### 21. 恢复数据库 (RESTORE_DATABASE)

**请求:**
```json
{
  "action": 6002,
  "data": {
    "proxy": [/* 备份的Proxy数据 */],
    "user": [/* 备份的User数据 */],
    "server": [/* 备份的Server数据 */],
    "ip_whitelist": [/* 备份的IpWhitelist数据 */]
  }
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "message": "Database restored successfully",
    "restored": {
      "proxy": 1,
      "user": 50,
      "server": 10,
      "ip_whitelist": 5
    }
  }
}
```

**恢复流程:**
1. 验证owner token(中间件完成)
2. 停止所有服务器连接
3. 清空所有会话
4. 使用事务清空并恢复数据
5. 重新加载IP白名单到内存
6. 重新初始化服务器连接

## 使用示例

### TypeScript示例

```typescript
// 备份数据库
async function backupDatabase(): Promise<any> {
  const response = await fetch('/admin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ownerToken}`
    },
    body: JSON.stringify({
      action: 6001,
      data: {}
    })
  });

  const result = await response.json();
  if (!result.success) {
    throw new Error(result.error.message);
  }

  // 保存备份数据到文件
  const backupData = result.data;
  fs.writeFileSync('backup.json', JSON.stringify(backupData, null, 2));

  return backupData;
}

// 恢复数据库
async function restoreDatabase(backupData: any): Promise<void> {
  const response = await fetch('/admin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ownerToken}`
    },
    body: JSON.stringify({
      action: 6002,
      data: backupData
    })
  });

  const result = await response.json();
  if (!result.success) {
    throw new Error(result.error.message);
  }

  console.log('Database restored:', result.data);
}
```

### cURL示例

```bash
# 查询用户列表
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer owner-token" \
  -d '{
    "action": 1011,
    "data": {
      "proxyId": 1
    }
  }'

# 备份数据库
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer owner-token" \
  -d '{
    "action": 6001,
    "data": {}
  }' > backup.json

# 恢复数据库
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer owner-token" \
  -d @backup.json
```

## 版本信息

- **协议版本**: 1.3
- **最后更新**: 2025年10月16日
- **更新内容**:
  - 新增数据库CRUD操作API (User, Server, Proxy, IpWhitelist)
  - 新增备份恢复功能
  - UPDATE_IP_WHITELIST改为"保存到数据库+加载到内存"模式
  - 新增完整的错误码定义
- **兼容性**: 向后兼容，支持扩展新的操作类型
