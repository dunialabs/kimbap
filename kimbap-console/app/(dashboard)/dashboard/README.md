# Dashboard Page Documentation

## 页面说明
此目录包含KIMBAP Console的Dashboard主页面，展示服务器运行状态、使用统计、工具和Token使用情况等关键信息。

## Protocol 10023 - Dashboard Overview Statistics API 实现文档

### 概述
Protocol 10023 是一个综合的Dashboard统计接口，为前端Dashboard页面提供所需的全部统计数据。该接口通过一次调用返回所有必要的统计信息，减少前后端交互次数，提高页面加载性能。

## 接口定义

### 请求参数 (Request10023)
```protobuf
message Request10023 {
  Request common = 1;  // 包含cmdId和userid
  Params params = 2;
  message Params {
    string timeRange = 1; // 时间范围: "24h", "7d", "30d", "90d"
  }
}
```

### 响应数据 (Response10023)
```protobuf
message Response10023 {
  Response common = 1;
  Data data = 2;
  message Data {
    // 基础统计
    string uptime = 1;              // 服务器运行时间
    int32 apiRequests = 2;          // API请求总数
    int32 activeTokens = 3;         // 活跃token数量
    int32 configuredTools = 4;      // 配置的工具数量
    int32 connectedClients = 5;     // 连接的客户端数量
    int32 monthlyUsage = 6;         // 月度使用百分比
    
    // 工具使用分布
    repeated ToolUsage toolsUsage = 7;
    
    // Token使用分布
    repeated TokenUsage tokenUsage = 8;
    
    // 连接的客户端详情
    repeated ConnectedClient connectedClientsList = 9;
    
    // 最近活动
    repeated RecentActivity recentActivity = 10;
  }
}
```

## 实现思路

### 1. 数据源分析
该接口需要从多个数据源聚合数据：
- **本地数据库 (Prisma)**：Log表、User表、Config表
- **Proxy API**：服务器信息、用户信息、服务器状态

### 2. 核心统计逻辑

#### 2.1 服务器运行时间 (uptime)
- 从Log表中查找最早的日志记录
- 计算从该时间点到当前时间的差值
- 格式化为 "X days, Y hours" 格式

#### 2.2 API请求总数 (apiRequests)
- 根据timeRange参数计算起始时间
- 统计Log表中在时间范围内的记录总数

#### 2.3 活跃Token数量 (activeTokens)
- 在指定时间范围内，从Log表中获取唯一的userid
- 统计去重后的userid数量

#### 2.4 配置的工具数量 (configuredTools)
- 调用proxy-api的getServers接口
- 返回服务器列表的长度

#### 2.5 连接的客户端数量 (connectedClients)
- 调用proxy-api的getServersStatus接口
- 统计状态为Online (0) 的服务器数量

#### 2.6 月度使用百分比 (monthlyUsage)
- 计算当月第一天的时间戳
- 统计本月的总请求数
- 假设月度限额为100,000，计算使用百分比

#### 2.7 工具使用分布 (toolsUsage)
- 按serverId分组统计Log表
- 计算每个工具的请求次数和占比
- 返回Top 10的工具

#### 2.8 Token使用分布 (tokenUsage)
- 按userid分组统计Log表
- 从本地User表获取token信息
- 对token进行脱敏处理（只显示后4位）
- 返回Top 10的token使用情况

#### 2.9 连接的客户端列表 (connectedClientsList)
- 获取最近24小时内的唯一客户端（userid + ip组合）
- 统计每个客户端的请求次数
- 格式化最后活跃时间

#### 2.10 最近活动 (recentActivity)
- 获取最新的10条日志记录
- 根据action码转换为可读的描述
- 根据是否有错误设置状态（success/warning）
- 格式化时间为相对时间（如"2 minutes ago"）

### 3. 性能优化策略

1. **并行查询**：尽可能并行执行独立的数据库查询和API调用
2. **缓存机制**：对于不经常变化的数据（如工具配置），可以考虑添加缓存
3. **分页限制**：对于列表类数据，限制返回数量（如Top 10）
4. **索引优化**：确保Log表的关键字段（addtime, userid, serverId）有适当的索引

### 4. 错误处理

- 每个统计模块独立处理错误，单个模块失败不影响其他数据
- 如果关键数据源（如proxy-api）不可用，返回默认值而不是抛出错误
- 记录详细的错误日志便于调试

### 5. 扩展性考虑

1. **可配置的限额**：月度使用限额应该可以配置
2. **时间范围扩展**：支持自定义时间范围
3. **数据过滤**：支持按特定条件过滤统计数据
4. **实时更新**：考虑使用WebSocket推送实时数据更新

## 使用示例

### 请求示例
```json
{
  "common": {
    "cmdId": 10023,
    "userid": "user123"
  },
  "params": {
    "timeRange": "30d"
  }
}
```

### 响应示例
```json
{
  "common": {
    "code": 0,
    "message": "Success",
    "cmdId": 10023
  },
  "data": {
    "uptime": "15 days, 8 hours",
    "apiRequests": 45678,
    "activeTokens": 12,
    "configuredTools": 8,
    "connectedClients": 5,
    "monthlyUsage": 46,
    "toolsUsage": [
      {
        "name": "github-copilot",
        "requests": 15234,
        "percentage": 33
      }
    ],
    "tokenUsage": [
      {
        "name": "user123",
        "token": "...a3b4",
        "requests": 8912,
        "percentage": 19
      }
    ],
    "connectedClientsList": [
      {
        "id": "user123-192.168.1.1",
        "name": "user123",
        "token": "...a3b4",
        "ip": "192.168.1.1",
        "location": "Unknown",
        "lastActive": "5 minutes ago",
        "requests": 234
      }
    ],
    "recentActivity": [
      {
        "action": "user123 - Tool configuration on github-copilot",
        "status": "success",
        "time": "2 minutes ago"
      }
    ]
  }
}
```

## 注意事项

1. **数据安全**：Token信息需要脱敏处理，只显示最后几位
2. **权限控制**：根据用户角色返回不同范围的数据
3. **性能监控**：监控接口响应时间，确保在可接受范围内
4. **数据准确性**：定期验证统计数据的准确性

## 后续优化建议

1. 添加数据缓存层，减少数据库查询
2. 实现增量更新机制，支持实时数据推送
3. 添加更多维度的统计分析（如按时间段、按用户角色等）
4. 支持导出统计报表功能