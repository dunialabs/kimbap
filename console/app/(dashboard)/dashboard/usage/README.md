# Usage Overview API Documentation

本文档说明使用概览主页 (`usage/page.tsx`) 中各个UI组件与后端API接口的映射关系，帮助前端开发人员了解数据获取方式。

## 页面结构与API映射

### 1. API Usage Summary Cards (API使用汇总卡片)

**位置**: 页面顶部的4个统计卡片
**接口**: `GET /api/v1?cmdid=22001`

```typescript
// 请求参数
{
  timeRange: 1 // 1-今天(24小时)
}

// 响应数据映射
totalRequests24h: data.totalRequests24h           // "Total Requests (24h)" 卡片主数值
totalRequestsYesterday: data.totalRequestsYesterday // 用于计算增长率
activeTokens: data.activeTokens                    // "Active Tokens" 卡片主数值
tokensUsedLastHour: data.tokensUsedLastHour       // "2 tokens used in last hour" 描述
toolsInUse: data.toolsInUse                       // "Tools in Use" 卡片主数值
mostActiveToolName: data.mostActiveToolName       // "Web Server MCP most active" 描述
avgResponseTime: data.avgResponseTime             // "Avg Response Time" 卡片主数值
avgResponseTimeYesterday: data.avgResponseTimeYesterday // 用于计算响应时间变化

// 计算增长率示例
const requestGrowth = totalRequestsYesterday > 0 ? 
  Math.round(((totalRequests24h - totalRequestsYesterday) / totalRequestsYesterday) * 100) : 0;
const responseTimeChange = avgResponseTime - avgResponseTimeYesterday;
```

### 2. Top Tools by Usage (工具使用排行)

**位置**: 页面中间的工具使用统计卡片
**接口**: `GET /api/v1?cmdid=22002`

```typescript
// 请求参数
{
  timeRange: 1,  // 最近24小时
  limit: 5       // 显示前5个工具
}

// 响应数据映射
topToolsData = data.topTools.map(tool => ({
  name: tool.toolName,           // 工具名称，如 "Web Server MCP"
  requests: tool.requestCount,   // 请求数量，如 1245
  percentage: tool.percentage,   // 占总请求的百分比，如 44
  color: tool.color             // UI显示颜色，如 "#3b82f6"
}))
```

### 3. Active Tokens (活跃令牌)

**位置**: 页面左下角的活跃令牌卡片
**接口**: `GET /api/v1?cmdid=22003`

```typescript
// 请求参数
{
  timeRange: 1,  // 最近24小时
  limit: 3       // 显示前3个活跃令牌
}

// 响应数据映射
activeTokensData = data.activeTokens.map(token => ({
  tokenName: token.tokenName,                    // 令牌名称，如 "Production API"
  tokenId: token.tokenId,                        // 脱敏ID，如 "prod_****...abc123"
  requests: token.requestCount,                  // 请求数量，如 1456
  lastUsed: token.lastUsedMinutesAgo,           // 最后使用时间(分钟前)
  isActive: token.isCurrentlyActive,            // 是否当前活跃
  statusText: token.isCurrentlyActive ? 
    `Active ${token.lastUsedMinutesAgo} min ago` : 
    `Active ${token.lastUsedMinutesAgo} min ago`,
  statusColor: token.isCurrentlyActive ? "text-green-600" : "text-muted-foreground"
}))
```

### 4. Recent Activity (最近活动)

**位置**: 页面右下角的最近活动卡片
**接口**: `GET /api/v1?cmdid=22004`

```typescript
// 请求参数
{
  limit: 5  // 显示最近5个活动
}

// 响应数据映射
recentActivityData = data.recentEvents.map(event => ({
  description: event.description,     // 事件描述，如 "Web Server MCP request completed"
  details: event.details,             // 详细信息，如 "2 minutes ago • 247ms response"
  minutesAgo: event.minutesAgo,       // 多少分钟前
  eventType: event.eventType,         // 事件类型: "tool_request", "token_auth", "rate_limit", "error"
  color: event.color,                 // 状态点颜色: "green", "blue", "yellow", "red"
  statusCode: event.statusCode        // HTTP状态码
}))

// 状态点颜色映射
const colorClasses = {
  green: "bg-green-500",
  blue: "bg-blue-500", 
  purple: "bg-purple-500",
  orange: "bg-orange-500",
  yellow: "bg-yellow-500",
  red: "bg-red-500"
};
```

## 实现建议

### 数据刷新策略

```typescript
// 汇总数据每30秒刷新一次
useEffect(() => {
  const fetchSummary = () => {
    // 调用22001接口获取汇总数据
  };
  
  fetchSummary(); // 立即执行
  const interval = setInterval(fetchSummary, 30000); // 30秒刷新
  
  return () => clearInterval(interval);
}, []);

// 活动数据每15秒刷新一次
useEffect(() => {
  const fetchActivity = () => {
    // 调用22004接口获取最近活动
  };
  
  fetchActivity();
  const interval = setInterval(fetchActivity, 15000); // 15秒刷新
  
  return () => clearInterval(interval);
}, []);
```

### 错误处理

```typescript
// 统一API错误处理
const handleApiError = (error: any, context: string) => {
  console.error(`${context} error:`, error);
  
  // 显示用户友好的错误信息
  showToast({
    title: "数据加载失败",
    description: "请稍后重试，或联系管理员",
    variant: "destructive"
  });
};

// 使用示例
try {
  const response = await fetchUsageSummary();
  if (response.common.code !== 0) {
    throw new Error(response.common.message);
  }
  // 处理成功响应
} catch (error) {
  handleApiError(error, "Usage Summary");
}
```

### 状态管理

```typescript
// 使用React Query进行数据管理
import { useQuery } from '@tanstack/react-query';

const useUsageSummary = () => {
  return useQuery({
    queryKey: ['usageSummary'],
    queryFn: () => fetchUsageSummary({ timeRange: 1 }),
    refetchInterval: 30000,     // 30秒自动刷新
    staleTime: 15000,          // 15秒内不重新请求
    retry: 3,                  // 失败重试3次
  });
};

const useTopTools = () => {
  return useQuery({
    queryKey: ['topTools'],
    queryFn: () => fetchTopTools({ timeRange: 1, limit: 5 }),
    refetchInterval: 60000,    // 1分钟自动刷新
    staleTime: 30000,
  });
};

const useActiveTokens = () => {
  return useQuery({
    queryKey: ['activeTokens'],
    queryFn: () => fetchActiveTokens({ timeRange: 1, limit: 3 }),
    refetchInterval: 30000,
    staleTime: 15000,
  });
};

const useRecentActivity = () => {
  return useQuery({
    queryKey: ['recentActivity'],
    queryFn: () => fetchRecentActivity({ limit: 5 }),
    refetchInterval: 15000,    // 15秒自动刷新
    staleTime: 5000,
  });
};
```

### TypeScript接口定义

```typescript
// types/usage-overview.types.ts
export interface UsageSummary {
  totalRequests24h: number;
  totalRequestsYesterday: number;
  activeTokens: number;
  tokensUsedLastHour: number;
  toolsInUse: number;
  mostActiveToolName: string;
  avgResponseTime: number;
  avgResponseTimeYesterday: number;
}

export interface TopTool {
  toolName: string;
  toolType: string;
  requestCount: number;
  percentage: number;
  color: string;
}

export interface ActiveToken {
  tokenId: string;
  tokenName: string;
  requestCount: number;
  lastUsedMinutesAgo: number;
  isCurrentlyActive: boolean;
}

export interface ActivityEvent {
  eventType: 'tool_request' | 'token_auth' | 'rate_limit' | 'error';
  description: string;
  details: string;
  minutesAgo: number;
  color: 'green' | 'blue' | 'purple' | 'orange' | 'yellow' | 'red';
  statusCode: number;
}
```

### 数据处理工具函数

```typescript
// utils/usage-helpers.ts

// 计算百分比变化
export const calculatePercentageChange = (current: number, previous: number): number => {
  if (previous === 0) return current > 0 ? 100 : 0;
  return Math.round(((current - previous) / previous) * 100);
};

// 格式化时间差
export const formatTimeAgo = (minutesAgo: number): string => {
  if (minutesAgo < 1) return "Just now";
  if (minutesAgo < 60) return `${minutesAgo} min ago`;
  const hoursAgo = Math.floor(minutesAgo / 60);
  if (hoursAgo < 24) return `${hoursAgo} hour${hoursAgo > 1 ? 's' : ''} ago`;
  const daysAgo = Math.floor(hoursAgo / 24);
  return `${daysAgo} day${daysAgo > 1 ? 's' : ''} ago`;
};

// 格式化响应时间
export const formatResponseTime = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
};

// 获取状态变化的显示文本和颜色
export const getChangeIndicator = (change: number) => {
  if (change > 0) {
    return {
      text: `+${change}%`,
      color: "text-green-600"
    };
  } else if (change < 0) {
    return {
      text: `${change}%`,
      color: "text-red-600"
    };
  } else {
    return {
      text: "0%",
      color: "text-muted-foreground"
    };
  }
};
```

### 组件使用示例

```typescript
// 在页面组件中使用
export default function UsagePage() {
  const { data: summary, isLoading: summaryLoading } = useUsageSummary();
  const { data: topTools, isLoading: toolsLoading } = useTopTools();
  const { data: activeTokens, isLoading: tokensLoading } = useActiveTokens();
  const { data: recentActivity, isLoading: activityLoading } = useRecentActivity();

  if (summaryLoading) return <div>Loading...</div>;

  const requestChange = summary ? 
    calculatePercentageChange(summary.totalRequests24h, summary.totalRequestsYesterday) : 0;
    
  const responseTimeChange = summary ? 
    summary.avgResponseTime - summary.avgResponseTimeYesterday : 0;

  return (
    <div className="space-y-4">
      {/* Summary Cards */}
      <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Requests (24h)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.totalRequests24h.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              <span className={getChangeIndicator(requestChange).color}>
                {getChangeIndicator(requestChange).text}
              </span> from yesterday
            </p>
          </CardContent>
        </Card>
        
        {/* 其他卡片... */}
      </div>
      
      {/* 其他组件... */}
    </div>
  );
}
```

## 注意事项

1. **实时性**: 关键指标数据需要定期刷新，建议汇总数据30秒刷新，活动数据15秒刷新
2. **错误处理**: 实现优雅的错误处理和加载状态
3. **性能优化**: 使用适当的缓存策略，避免过度请求
4. **用户体验**: 在数据加载时显示骨架屏或加载指示器
5. **数据验证**: 对API返回的数据进行基本验证，防止显示异常值