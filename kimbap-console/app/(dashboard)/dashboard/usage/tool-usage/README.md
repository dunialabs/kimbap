# Tool Usage Statistics API Documentation

本文档说明工具使用统计页面 (`tool-usage/page.tsx`) 中各个UI组件与后端API接口的映射关系，帮助前端开发人员了解数据获取方式。

## 页面结构与API映射

### 1. Overview Tab (概览标签页)

#### 1.1 Summary Cards (汇总卡片)
**位置**: 页面顶部的4个统计卡片
**接口**: `GET /api/v1?cmdid=20001`
```typescript
// 请求参数
{
  timeRange: 30, // 最近30天
  serverId: 0    // 所有服务器
}

// 响应数据映射
totalTools: data.totalTools          // "Total Tools"卡片
totalRequests: data.totalRequests    // "Total Requests"卡片
avgSuccessRate: data.avgSuccessRate  // "Success Rate"卡片
avgResponseTime: data.avgResponseTime // "Avg Response Time"卡片
```

#### 1.2 Tool Usage Distribution (工具使用分布饼图)
**位置**: Overview中间的饼图
**接口**: `GET /api/v1?cmdid=20005`
```typescript
// 请求参数
{
  timeRange: 30,    // 最近30天
  serverId: 0,      // 所有服务器
  metricType: 1     // 按请求数统计
}

// 响应数据格式
pieData = data.distribution.map(item => ({
  name: item.toolName,
  value: item.value,
  percentage: item.percentage
}))
```

#### 1.3 Detailed Tool List (工具详细列表)
**位置**: Overview底部的工具详情卡片列表
**接口**: `GET /api/v1?cmdid=20002`
```typescript
// 请求参数
{
  timeRange: 30,
  serverId: 0,
  toolId: "",       // 空表示所有工具
  page: 1,
  pageSize: 50
}

// 响应数据映射到ToolUsageData接口
toolUsageData = data.toolMetrics.map(tool => ({
  toolName: tool.toolName,
  totalRequests: tool.totalRequests,
  successfulRequests: tool.successRequests,
  failedRequests: tool.failedRequests,
  averageResponseTime: tool.avgResponseTime,
  successRate: tool.successRate,
  lastUsed: new Date(tool.lastUsed * 1000).toISOString(),
  status: tool.status === 1 ? "active" : 
          tool.status === 2 ? "inactive" : "error"
}))
```

### 2. Performance Tab (性能标签页)

#### 2.1 Response Time Comparison (响应时间对比柱状图)
**位置**: Performance第一个图表
**接口**: `GET /api/v1?cmdid=20006`
```typescript
// 请求参数
{
  timeRange: 30,
  serverId: 0,
  metricType: 1    // 响应时间对比
}

// 数据格式
responseTimeData = data.comparison.map(item => ({
  toolName: item.toolName,
  averageResponseTime: item.avgValue
}))
```

#### 2.2 Success vs Failed Requests (成功失败请求对比)
**位置**: Performance第二个图表
**接口**: `GET /api/v1?cmdid=20002` (复用详细指标接口)
```typescript
// 使用工具详细指标接口的数据
successVsFailedData = data.toolMetrics.map(tool => ({
  toolName: tool.toolName,
  successfulRequests: tool.successRequests,
  failedRequests: tool.failedRequests
}))
```

### 3. Error Analysis Tab (错误分析标签页)

**位置**: 错误分析页面的错误卡片列表
**接口**: `GET /api/v1?cmdid=20004`
```typescript
// 请求参数
{
  timeRange: 30,
  serverId: 0,
  toolId: ""       // 空表示所有工具
}

// 响应数据映射
errorData = data.toolErrors.map(toolError => ({
  toolName: toolError.toolName,
  failedRequests: toolError.totalErrors,
  totalRequests: 0, // 需要从20002接口补充
  errorTypes: toolError.errorTypes.map(error => ({
    type: error.errorMessage,
    count: error.count
  }))
}))
```

### 4. Usage Trends Tab (使用趋势标签页)

**位置**: 趋势页面的折线图
**接口**: `GET /api/v1?cmdid=20003`
```typescript
// 请求参数
{
  timeRange: 7,     // 最近7天
  serverId: 0,
  toolIds: [],      // 空表示所有工具
  granularity: 2    // 按天统计
}

// 响应数据格式转换
trendData = data.trendData.map(point => {
  const dataPoint = { date: point.date };
  point.toolCounts.forEach(tool => {
    dataPoint[tool.toolName] = tool.requestCount;
  });
  return dataPoint;
})
```

## 额外功能接口

### 5. Real-time Status (实时状态) - 可用于状态指示器
**接口**: `GET /api/v1?cmdid=20008`
```typescript
// 获取工具实时状态，用于状态徽章显示
// 可以定时轮询(如每30秒)更新工具状态
```

### 6. Export Report (导出报告) - 可添加导出功能
**接口**: `GET /api/v1?cmdid=20009`
```typescript
// 请求参数
{
  timeRange: 30,
  serverId: 0,
  format: 1,        // 1-CSV, 2-JSON, 3-PDF
  toolIds: []       // 空表示所有工具
}
```

### 7. Action Logs (操作日志) - 可用于详细日志查看
**接口**: `GET /api/v1?cmdid=20010`
```typescript
// 获取基于MCPEventLogType的详细操作日志
// 可作为钻取功能，点击工具查看详细日志
```

### 8. User Usage Analysis (用户使用分析) - 可扩展用户维度
**接口**: `GET /api/v1?cmdid=20007`
```typescript
// 按用户维度分析工具使用情况
// 可用于管理员查看用户使用统计
```

## 实现建议

### 数据刷新策略
- **Summary Cards**: 页面加载时获取，5分钟自动刷新
- **Charts**: 切换Tab时获取，支持手动刷新
- **Real-time Status**: 30秒轮询更新

### 错误处理
```typescript
// 统一错误处理
if (response.common.code !== 0) {
  console.error('API Error:', response.common.message);
  // 显示错误提示
}
```

### 分页处理
```typescript
// 对于支持分页的接口(20002, 20007, 20010)
const [page, setPage] = useState(1);
const pageSize = 20;

// 请求时带上分页参数
const params = {
  // ...其他参数
  page: page,
  pageSize: pageSize
};
```

### 时间范围控制
建议在页面顶部添加时间范围选择器：
```typescript
const timeRangeOptions = [
  { label: "今天", value: 1 },
  { label: "最近7天", value: 7 },
  { label: "最近30天", value: 30 },
  { label: "最近90天", value: 90 }
];
```

### TypeScript接口定义
```typescript
// 建议在types目录下创建tool-usage-api.types.ts
export interface ToolUsageSummary {
  totalTools: number;
  activeTools: number;
  totalRequests: number;
  successRequests: number;
  failedRequests: number;
  avgSuccessRate: number;
  avgResponseTime: number;
  totalUsers: number;
}

export interface ToolMetrics {
  toolId: string;
  toolName: string;
  toolType: number;
  totalRequests: number;
  successRequests: number;
  failedRequests: number;
  successRate: number;
  avgResponseTime: number;
  minResponseTime: number;
  maxResponseTime: number;
  lastUsed: number;
  status: number;
  uniqueUsers: number;
}
// ... 其他接口定义
```