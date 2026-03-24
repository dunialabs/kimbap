# Protocol Buffers 接口文档

使用 Protocol Buffers 定义 API 接口，仅作为文档使用。使用协议号代替路由名称。

## 文件说明

- `common.proto` - 公共类型定义
- `api.proto` - API 接口定义（使用协议号）

## 协议号规则

- 10001-10099: 认证相关
- 10100-10199: 用户管理
- 10200-10299: 服务器管理
- 10300-10399: 仪表板
- 10400-10499: 日志管理

## 使用示例

### 1. 路由映射

传统路由：`POST /api/login`  
协议号路由：`POST /api/10001`

### 2. Proto 定义

```protobuf
// 10001 - 用户登录 [POST /api/10001]
message Request10001 {
  string email = 1;
  string password = 2;
}

message Response10001 {
  string token = 1;
  common.UserInfo user = 2;
}
```

### 3. TypeScript 实现

```typescript
// 请求接口
interface Request10001 {
  email: string;
  password: string;
}

// 响应接口
interface Response10001 {
  token: string;
  user: UserInfo;
}

// API 调用
const response = await fetch('/api/10001', {
  method: 'POST',
  body: JSON.stringify({ email, password })
});
```

### 4. 通用响应格式

所有接口返回统一的响应格式：
```typescript
interface ApiResponse {
  code: number;    // 0:成功 其他:错误码
  message: string; // 响应消息
  data?: any;      // 具体的响应数据 (Response10001等)
}
```