# Token Usage Statistics API Documentation

本文档说明访问令牌使用统计页面 (`token-usage/page.tsx`) 中各个UI组件与后端API接口的映射关系，帮助前端开发人员了解数据获取方式。

## 页面结构与API映射

### 1. Overview Tab (概览标签页)

#### 1.1 Summary Cards (汇总卡片)
**位置**: 页面顶部的4个统计卡片
**接口**: `GET /api/v1?cmdid=21001`
```typescript
// 请求参数
{
  timeRange: 30 // 最近30天
}

// 响应数据映射
totalTokens: data.totalTokens        // "Total Tokens"卡片
totalRequests: data.totalRequests    // "Total Requests"卡片
avgSuccessRate: data.avgSuccessRate  // "Success Rate"卡片
totalClients: data.totalClients      // "Connected Clients"卡片
```

#### 1.2 Token Usage Distribution (令牌使用分布饼图)
**位置**: Overview中间的饼图
**接口**: `GET /api/v1?cmdid=21006`
```typescript
// 请求参数
{
  timeRange: 30,    // 最近30天
  metricType: 1     // 按请求数统计
}

// 响应数据格式
pieData = data.distribution.map(item => ({
  name: item.tokenName,
  value: item.value,
  percentage: item.percentage
}))
```

#### 1.3 Detailed Token List (令牌详细列表)
**位置**: Overview底部的令牌详情卡片列表
**接口**: `GET /api/v1?cmdid=21002`
```typescript
// 请求参数
{
  timeRange: 30,
  tokenId: "",      // 空表示所有令牌
  page: 1,
  pageSize: 50
}

// 响应数据映射到TokenUsageData接口
tokenUsageData = data.tokenMetrics.map(token => ({
  tokenName: token.tokenName,
  tokenId: token.tokenId,
  totalRequests: token.totalRequests,
  successfulRequests: token.successRequests,
  failedRequests: token.failedRequests,
  rateLimit: token.rateLimit,
  lastUsed: new Date(token.lastUsed * 1000).toISOString(),
  status: token.status === 1 ? "active" : 
          token.status === 2 ? "inactive" : 
          token.status === 3 ? "expired" : "limited",
  createdDate: new Date(token.createdAt * 1000).toISOString().split('T')[0],
  expiryDate: token.expiresAt > 0 ? new Date(token.expiresAt * 1000).toISOString().split('T')[0] : null,
  clientCount: token.clientCount,
  topLocations: token.topLocations.map(loc => ({
    country: loc.country,
    city: loc.city,
    requests: loc.requests
  }))
}))
```

### 2. Geographic Usage Tab (地理位置使用标签页)

**位置**: Geographic页面的地理分布卡片
**接口**: `GET /api/v1?cmdid=21004`
```typescript
// 请求参数
{
  timeRange: 30,
  tokenId: ""       // 空表示所有令牌
}

// 响应数据映射
geographicData = data.geoUsage.map(tokenGeo => ({
  tokenName: tokenGeo.tokenName,
  locations: tokenGeo.locations.map(loc => ({
    country: loc.country,
    city: loc.city,
    requests: loc.requests,
    percentage: loc.percentage
  }))
}))
```

### 3. Usage Patterns Tab (使用模式标签页)

#### 3.1 Usage Trends (使用趋势折线图)
**位置**: 趋势页面的第一个折线图
**接口**: `GET /api/v1?cmdid=21003`
```typescript
// 请求参数
{
  timeRange: 7,     // 最近7天
  tokenIds: [],     // 空表示所有令牌
  granularity: 2    // 按天统计
}

// 响应数据格式转换
trendData = data.trendData.map(point => {
  const dataPoint = { date: point.date };
  point.tokenCounts.forEach(token => {
    dataPoint[token.tokenName] = token.requestCount;
  });
  return dataPoint;
})
```

#### 3.2 Minute Usage Pattern (分钟级使用模式)
**位置**: 每个活跃令牌的分钟使用模式图表
**接口**: `GET /api/v1?cmdid=21005`
```typescript
// 请求参数
{
  tokenId: "specific-token-id", // 特定令牌ID
  patternType: 1                // 1-最近60分钟
}

// 响应数据格式
minuteUsage = data.patterns.map(point => ({
  minute: point.timeLabel,
  requests: point.requests
}))
```

## 额外功能接口

### 4. Token Rate Limit Analysis (速率限制分析) - 可用于监控告警
**接口**: `GET /api/v1?cmdid=21007`
```typescript
// 获取令牌速率限制分析，用于监控达到限制的情况
// 可以定时查询，发现频繁触发限制的令牌
```

### 5. Token Client Analysis (客户端分析) - 可用于安全监控
**接口**: `GET /api/v1?cmdid=21008`
```typescript
// 分析使用各令牌的客户端信息
// 可用于检测异常的IP访问模式
```

### 6. Export Token Report (导出令牌报告) - 可添加导出功能
**接口**: `GET /api/v1?cmdid=21009`
```typescript
// 请求参数
{
  timeRange: 30,
  format: 1,        // 1-CSV, 2-JSON, 3-PDF
  tokenIds: [],     // 空表示所有令牌
  includeGeoData: true,     // 包含地理位置数据
  includeClientData: true   // 包含客户端数据
}
```

### 7. Token Audit Logs (令牌审计日志) - 可用于详细日志查看
**接口**: `GET /api/v1?cmdid=21010`
```typescript
// 获取令牌相关的审计日志
// 可作为钻取功能，点击令牌查看详细操作记录
```

## 实现建议

### 数据刷新策略
- **Summary Cards**: 页面加载时获取，5分钟自动刷新
- **Geographic Data**: 切换Tab时获取，支持手动刷新
- **Usage Patterns**: 活跃令牌的分钟模式每30秒刷新
- **Token Details**: 分页数据按需加载

### 错误处理
```typescript
// 统一错误处理
if (response.common.code !== 0) {
  console.error('API Error:', response.common.message);
  // 显示错误提示
  showToast({
    title: "数据加载失败",
    description: response.common.message,
    variant: "destructive"
  });
}
```

### 分页处理
```typescript
// 对于支持分页的接口(21002, 21010)
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

### 实时更新策略
```typescript
// 分钟级数据的实时更新
useEffect(() => {
  const interval = setInterval(() => {
    // 只更新活跃令牌的分钟使用模式
    activeTokens.forEach(token => {
      fetchTokenMinutePattern(token.tokenId);
    });
  }, 30000); // 30秒更新一次
  
  return () => clearInterval(interval);
}, [activeTokens]);
```

### TypeScript接口定义
```typescript
// 建议在types目录下创建token-usage-api.types.ts
export interface TokenUsageSummary {
  totalTokens: number;
  activeTokens: number;
  totalRequests: number;
  successRequests: number;
  failedRequests: number;
  avgSuccessRate: number;
  totalClients: number;
  expiredTokens: number;
  limitedTokens: number;
}

export interface TokenMetrics {
  tokenId: string;
  tokenName: string;
  userId: string;
  role: number;
  totalRequests: number;
  successRequests: number;
  failedRequests: number;
  successRate: number;
  rateLimit: number;
  rateLimitHits: number;
  lastUsed: number;
  createdAt: number;
  expiresAt: number;
  status: number;
  clientCount: number;
  topLocations: Location[];
}

export interface Location {
  country: string;
  city: string;
  requests: number;
  percentage: number;
}

export interface UsagePoint {
  timeLabel: string;
  requests: number;
  successRequests: number;
  failedRequests: number;
  rateLimitHits: number;
}
// ... 其他接口定义
```

### 状态管理建议
```typescript
// 使用React Query或SWR进行数据缓存和同步
import { useQuery } from '@tanstack/react-query';

const useTokenSummary = (timeRange: number) => {
  return useQuery({
    queryKey: ['tokenSummary', timeRange],
    queryFn: () => fetchTokenSummary(timeRange),
    refetchInterval: 5 * 60 * 1000, // 5分钟刷新
    staleTime: 2 * 60 * 1000,       // 2分钟内不重新请求
  });
};
```

### 性能优化
1. **虚拟化长列表**: 如果令牌数量很多，考虑使用虚拟滚动
2. **按需加载**: 地理位置数据和分钟模式数据按需加载
3. **缓存策略**: 利用浏览器缓存减少重复请求
4. **防抖处理**: 搜索和筛选功能添加防抖

### 安全考虑
1. **令牌脱敏**: 确保令牌ID在前端显示时已脱敏
2. **权限控制**: 根据用户角色显示不同的数据范围
3. **审计日志**: 记录敏感操作的访问日志