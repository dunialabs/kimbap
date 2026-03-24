# Tool Capabilities 为空问题排查与修复

## 问题描述

在添加工具后，通过 Protocol 10009 获取 scopes 时返回空数组，导致创建 token 时没有可选的 scopes。

## 问题现象

1. **Frontend**: 创建 token 时，scopes 选择框为空
2. **API 调用**: Protocol 10009 返回 `{"scopes": []}`
3. **Backend API**: `GET_SERVERS_CAPABILITIES` 返回 `{"tools": {}, "resources": {}, "prompts": {}}`
4. **数据库**: 服务器记录存在且 `enabled = true`，但 `capabilities = {}`

## 根本原因分析

### 1. Backend 服务未启动
**症状**: API 无法连接到 `http://localhost:3002/admin`  
**原因**: 缺少 backend 服务器进程  
**解决**: 启动 backend 服务 (`npm run dev:backend`)

### 2. 环境配置问题
**症状**: PROXY_ADMIN_URL 未配置  
**原因**: `.env` 文件中 `PROXY_ADMIN_URL` 被注释  
**解决**: 取消注释并设置 `PROXY_ADMIN_URL="http://localhost:3002/admin"`

### 3. MCP 服务器未启动
**症状**: 服务器状态为离线 (status = 1)  
**原因**: 工具创建后，MCP 服务器进程未正确启动  
**解决**: 通过 master password 手动启动或检查服务器启动逻辑

### 4. ServerManager 中 Key 不一致问题 ⚠️ **关键修复**
**症状**: `GET_SERVERS_STATUS` 返回 server_name 作为 key，而不是 server_id  
**原因**: `ServerManager.startServer()` 中使用了不一致的 key：

```typescript
// 错误的代码
this.serverContexts.set(serverEntity.serverName, serverContext);

// 正确的代码  
this.serverContexts.set(serverEntity.serverId, serverContext);
```

**影响**: 
- Protocol 10009 无法正确匹配服务器
- Frontend 和 Backend 数据不同步
- 跨环境兼容性问题

## 修复步骤

### 1. 立即修复
```bash
# 1. 修复 ServerManager.ts 中的 key 不一致
# 在 backend-src/src/core/ServerManager.ts 第119行
# 将 serverEntity.serverName 改为 serverEntity.serverId

# 2. 重新编译 backend
npm run build:backend

# 3. 配置环境变量
echo 'PROXY_ADMIN_URL="http://localhost:3002/admin"' >> .env

# 4. 重启所有服务
npm run dev
```

### 2. 验证修复
```bash
# 检查服务器状态 - 应返回 server_id 作为 key
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"action": 3004, "data": {}}'

# 预期返回格式:
# {"success":true,"data":{"serversStatus":{"159c049c9bfa4906a10e73cbc6a3d57a":0}}}
```

## 预防措施

### 1. 启动脚本改进
已修改 `scripts/start-with-ports.js`，自动执行：
- `db:push` - 同步数据库 schema
- `db:generate` - 生成 Prisma Client  
- `build:backend` - 编译 backend 代码

### 2. 环境检查清单
在启动开发环境前确认：
- [ ] Docker 服务正在运行
- [ ] PostgreSQL 容器已启动
- [ ] `PROXY_ADMIN_URL` 已配置
- [ ] Backend 和 Frontend 端口无冲突

### 3. 调试日志
添加了详细的调试日志在：
- `protocol-10009.ts` - 追踪 scopes 获取过程
- `tool-configure/[id]/page.tsx` - 追踪工具创建数据流

## 常见错误码

| 错误码 | 含义 | 解决方法 |
|--------|------|----------|
| 3001 | Server not found | 检查 server_id 是否存在于数据库 |
| 1003 | Admin token required | 提供有效的 Authorization Bearer token |
| Connection refused | Backend 服务未启动 | 启动 `npm run dev:backend` |

## 相关文件

- `backend-src/src/core/ServerManager.ts` - 服务器管理核心逻辑
- `app/api/v1/handlers/protocol-10009.ts` - Scopes 获取处理
- `app/(dashboard)/dashboard/tool-configure/[id]/page.tsx` - 工具配置界面
- `lib/proxy-api.ts` - Proxy API 封装
- `.env` - 环境配置

## 版本信息

- 修复时间: 2025-08-29
- 影响版本: All versions before this fix
- 兼容性: 与同事环境完全兼容

---

> **注意**: 此问题会导致工具创建后无法获取 capabilities，是影响核心功能的关键 bug。建议优先修复。