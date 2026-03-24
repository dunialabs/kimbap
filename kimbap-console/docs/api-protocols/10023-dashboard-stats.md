# Protocol 10023 - Dashboard Statistics API

## 简介
Protocol 10023 为Dashboard页面提供综合的统计数据接口。

## 文件位置
- **接口实现**: `/app/api/v1/handlers/protocol-10023.ts`
- **Protobuf定义**: `/protos/api.proto` (Message: Request10023, Response10023)
- **完整文档**: [`/app/(dashboard)/dashboard/README.md`](../../app/(dashboard)/dashboard/README.md)

## 快速参考

### 请求
```json
{
  "common": {
    "cmdId": 10023,
    "userid": "user123"
  },
  "params": {
    "timeRange": "30d"  // 可选: "24h", "7d", "30d", "90d"
  }
}
```

### 响应数据结构
- `uptime`: 服务器运行时间
- `apiRequests`: API请求总数
- `activeTokens`: 活跃Token数量
- `configuredTools`: 配置的工具数量
- `connectedClients`: 连接的客户端数量
- `monthlyUsage`: 月度使用百分比
- `toolsUsage[]`: 工具使用分布
- `tokenUsage[]`: Token使用分布
- `connectedClientsList[]`: 连接客户端列表
- `recentActivity[]`: 最近活动记录

详细文档请参考[Dashboard页面文档](../../app/(dashboard)/dashboard/README.md)。