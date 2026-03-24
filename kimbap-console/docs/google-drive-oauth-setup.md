# Google Drive OAuth 配置指南

本文档介绍如何在 Kimbap MCP Console 中配置 Google Drive OAuth 授权。

## 简化流程

用户选择 Google Drive 模板 → 跳转 Google 授权 → Google 重定向回 `/dashboard/tool-configure?code=xxx&state=xxx` → 前端自动交换 token 并添加工具

## 1. 在 Google Cloud Console 创建 OAuth 凭据

### 步骤 1: 创建项目
1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目

### 步骤 2: 启用 Google Drive API
1. 在左侧菜单中选择 "API 和服务" > "库"
2. 搜索 "Google Drive API"
3. 点击启用

### 步骤 3: 创建 OAuth 2.0 凭据
1. 在左侧菜单中选择 "API 和服务" > "凭据"
2. 点击 "创建凭据" > "OAuth 客户端 ID"
3. 如果是首次创建，需要先配置 OAuth 同意屏幕
4. 选择应用类型: "Web 应用"
5. 配置以下信息:
   - 名称: Kimbap MCP Console
   - 已获授权的重定向 URI (需要添加所有可能的访问地址):
     - `http://localhost:3000/dashboard/tool-configure`
     - `https://yourdomain.com/dashboard/tool-configure` (替换为实际域名)
6. 点击创建，记录下 `Client ID` 和 `Client Secret`

## 2. 配置环境变量

在项目根目录的 `.env.local` 文件中添加:

```bash
# Google Drive OAuth 配置
GOOGLE_CLIENT_ID="your-client-id.apps.googleusercontent.com"
GOOGLE_CLIENT_SECRET="your-client-secret"
```

**注意**:
- `GOOGLE_REDIRECT_URI` 已改为动态配置，系统会自动使用当前请求的 host + 端口
- 在 Google Cloud Console 配置重定向 URI 时，需要添加所有可能的访问地址:
  - 开发环境: `http://localhost:3000/dashboard/tool-configure`
  - 生产环境: `https://yourdomain.com/dashboard/tool-configure`

## 3. 在代码中使用

### 在工具模板选择时启动授权

```typescript
import { startGoogleDriveAuth } from '@/lib/google-oauth'

// 用户点击 Google Drive 模板时
function handleSelectGoogleDrive() {
  const toolTmplId = 'google-drive-template-id' // 从模板列表获取
  const proxyId = '1' // 当前 proxy ID

  // 直接跳转到 Google 授权页面
  startGoogleDriveAuth(toolTmplId, proxyId)
}
```

### 在工具列表页面自动添加工具

```typescript
'use client'

import { useEffect } from 'react'
import { useSearchParams } from 'next/navigation'
import { parseGoogleDriveAuthCallback } from '@/lib/google-oauth'
import { api } from '@/lib/api-client'

export default function ToolsPage() {
  const searchParams = useSearchParams()

  useEffect(() => {
    // 检查是否是 Google Drive 授权回调
    if (searchParams.get('google_auth') === 'success') {
      const toolData = parseGoogleDriveAuthCallback(searchParams)

      if (toolData) {
        // 自动调用添加工具接口
        addGoogleDriveTool(toolData)
      }
    }
  }, [searchParams])

  async function addGoogleDriveTool(toolData: any) {
    try {
      // 调用协议 10005 添加工具
      const response = await api.tools.operateTool({
        handleType: 1, // add
        proxyId: toolData.proxyId,
        toolTmplId: toolData.toolTmplId,
        toolType: 1, // Google Drive
        authConf: [
          {
            key: 'access_token',
            value: toolData.accessToken
          },
          {
            key: 'refresh_token',
            value: toolData.refreshToken
          },
          {
            key: 'expires_in',
            value: toolData.expiresIn.toString()
          }
        ],
        masterPwd: getMasterPassword() // 从本地获取
      })

      if (response.data?.success) {
        console.log('Google Drive tool added successfully')
        // 刷新工具列表
        // TODO: 重新获取工具列表
      }
    } catch (error) {
      console.error('Failed to add Google Drive tool:', error)
    }
  }

  return (
    // Your component JSX
  )
}
```

## 4. API 路由说明

项目已自动创建以下 API 路由:

- `GET /api/auth/google` - 启动 OAuth 授权流程
- `GET /api/auth/google/callback` - OAuth 回调处理
- `POST /api/auth/google/refresh` - 刷新 access token

## 5. OAuth Scopes

当前配置的 Google Drive 权限范围:
- `https://www.googleapis.com/auth/drive.file` - 仅访问由应用创建或打开的文件

如需更多权限，可以在 `/app/api/auth/google/route.ts` 中修改 `scope` 参数:

```typescript
// 完整 Drive 访问权限
authUrl.searchParams.set('scope', 'https://www.googleapis.com/auth/drive')

// 只读权限
authUrl.searchParams.set('scope', 'https://www.googleapis.com/auth/drive.readonly')
```

## 6. 安全注意事项

1. **不要提交敏感信息**:
   - `.env.local` 文件不应该提交到 Git
   - 使用 `.env.example` 作为配置模板

2. **生产环境配置**:
   - 使用 HTTPS
   - 配置正确的重定向 URI
   - 定期轮换 Client Secret

3. **Token 存储**:
   - Access Token 和 Refresh Token 应加密存储
   - 不要在客户端 localStorage 中存储敏感 token
   - 建议存储在服务器端或使用加密的数据库

## 7. 常见问题

### Q: 回调失败，显示 redirect_uri_mismatch
A: 检查 Google Cloud Console 中配置的重定向 URI 是否与代码中的完全一致（包括协议、域名、端口、路径）

### Q: 如何获取用户信息？
A: 可以添加 `openid email profile` scope，并调用 Google UserInfo API

### Q: Token 过期后如何自动刷新？
A: 使用 `refreshGoogleToken` 函数，建议在 token 过期前 5 分钟自动刷新

## 8. 相关资源

- [Google OAuth 2.0 文档](https://developers.google.com/identity/protocols/oauth2)
- [Google Drive API 文档](https://developers.google.com/drive/api/guides/about-sdk)
- [OAuth 2.0 Scopes](https://developers.google.com/identity/protocols/oauth2/scopes#drive)
