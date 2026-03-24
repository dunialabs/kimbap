# Usage Overview Statistics API Design Document

## 概览

本文档详细描述了使用概览统计接口（22001-22010）的数据计算逻辑和设计思路，用于指导未来的接口修改和优化。这些接口为系统使用概览主页提供核心数据支持。

## 数据基础

### 数据源表结构

所有统计接口基于以下表进行数据分析：

```sql
-- Log表关键字段（主要数据源）
id                     INT     -- 主键
addtime                BIGINT  -- 添加时间(Unix时间戳)
userid                 VARCHAR -- 用户ID
tokenMask              VARCHAR -- 令牌掩码（用于识别令牌使用）
action                 VARCHAR -- 操作类型（用于识别工具类型）
ip                     VARCHAR -- 客户端IP
ua                     VARCHAR -- 用户代理
statusCode            INT     -- HTTP状态码
error                 TEXT    -- 错误信息
duration              INT     -- 响应时间(毫秒)
sessionId             VARCHAR -- 会话ID

-- User表关键字段（用于令牌信息）
userid                 VARCHAR -- 用户ID（作为令牌标识）
accessToken           VARCHAR -- 访问令牌
role                  INT     -- 用户角色 (1-owner, 2-admin, 3-member)
```

### 时间范围计算标准

```typescript
// 统一的时间计算逻辑
const now = Math.floor(Date.now() / 1000);
const timeRangeSeconds = timeRange * 24 * 60 * 60;
const startTime = now - timeRangeSeconds;

// 特殊时间点
const oneDayAgo = now - (24 * 60 * 60);        // 24小时前
const oneHourAgo = now - (60 * 60);             // 1小时前  
const fiveMinutesAgo = now - (5 * 60);          // 5分钟前
const yesterday = now - (24 * 60 * 60);         // 昨天同一时间
const dayBeforeYesterday = now - (2 * 24 * 60 * 60); // 前天同一时间
```

## 接口详细设计

### 22001 - 使用概览汇总统计

**目的**: 提供系统使用概览主页的核心指标数据

**核心计算逻辑**:

```typescript
// 1. 24小时总请求数
totalRequests24h = COUNT(*) FROM log WHERE addtime >= yesterday

// 2. 昨天总请求数（用于计算增长率）
totalRequestsYesterday = COUNT(*) FROM log 
  WHERE addtime >= dayBeforeYesterday AND addtime < yesterday

// 3. 活跃令牌数（24小时内有活动的令牌）
activeTokens = COUNT(DISTINCT tokenMask) FROM log 
  WHERE addtime >= yesterday AND tokenMask != ''

// 4. 最近1小时使用的令牌数
tokensUsedLastHour = COUNT(DISTINCT tokenMask) FROM log 
  WHERE addtime >= oneHourAgo AND tokenMask != ''

// 5. 使用中的工具数（基于action字段）
toolsInUse = COUNT(DISTINCT action) FROM log 
  WHERE addtime >= yesterday AND action != ''

// 6. 最活跃的工具（24小时内请求最多的action）
mostActiveToolName = SELECT action, COUNT(*) as cnt FROM log 
  WHERE addtime >= yesterday AND action != ''
  GROUP BY action ORDER BY cnt DESC LIMIT 1

// 7. 24小时平均响应时间
avgResponseTime = AVG(duration) FROM log 
  WHERE addtime >= yesterday AND duration > 0

// 8. 昨天平均响应时间（用于计算变化）
avgResponseTimeYesterday = AVG(duration) FROM log 
  WHERE addtime >= dayBeforeYesterday AND addtime < yesterday AND duration > 0
```

**设计考量**:
- 提供昨天的对比数据，便于计算增长率和变化趋势
- 区分不同时间粒度的令牌活跃度（24小时 vs 1小时）
- 基于action字段推断工具使用情况
- 过滤异常的响应时间数据（duration > 0）

### 22002 - 使用量最高的工具

**目的**: 识别系统中最常用的MCP工具，提供使用分布概览

**核心计算逻辑**:

```typescript
// 1. 按action分组统计
toolUsageStats = SELECT action, COUNT(*) as request_count FROM log 
  WHERE addtime >= startTime AND action != ''
  GROUP BY action ORDER BY request_count DESC LIMIT {limit}

// 2. 总请求数（用于计算百分比）
totalRequests = COUNT(*) FROM log WHERE addtime >= startTime

// 3. 工具名称和类型推断
function inferToolInfo(action: string): { name: string; type: string } {
  if (action.includes('web') || action.includes('http')) {
    return { name: 'Web Server MCP', type: 'Web Service' };
  } else if (action.includes('notion')) {
    return { name: 'Notion MCP', type: 'Productivity' };
  } else if (action.includes('github')) {
    return { name: 'GitHub MCP', type: 'Development' };
  } else if (action.includes('postgres') || action.includes('sql')) {
    return { name: 'PostgreSQL MCP', type: 'Database' };
  } else {
    return { name: formatActionName(action), type: 'Custom' };
  }
}

// 4. 百分比计算
percentage = (requestCount / totalRequests) * 100
```

**工具识别策略**:
- 基于action字段的关键词匹配
- 预定义常见MCP工具的映射规则
- 未知工具自动格式化名称
- 为每个工具分配UI显示颜色

### 22003 - 活跃令牌概览

**目的**: 显示最活跃的访问令牌及其使用状态

**核心计算逻辑**:

```typescript
// 1. 按tokenMask分组统计活跃令牌
tokenStats = SELECT tokenMask, COUNT(*) as request_count, MAX(addtime) as last_used
  FROM log WHERE addtime >= startTime AND tokenMask != ''
  GROUP BY tokenMask ORDER BY request_count DESC LIMIT {limit}

// 2. 令牌名称生成（基于tokenMask推断环境）
function generateTokenName(tokenMask: string): string {
  const prefix = tokenMask.substring(0, 4).toLowerCase();
  
  if (prefix.includes('prod')) return 'Production API';
  if (prefix.includes('dev')) return 'Development';
  if (prefix.includes('test')) return 'Testing';
  if (prefix.includes('stag')) return 'Staging';
  
  // 使用哈希分配环境名称确保一致性
  const environments = ['Production API', 'Development', 'Testing', 'Staging'];
  const hash = tokenMask.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return environments[hash % environments.length];
}

// 3. 活跃状态判断
isCurrentlyActive = (lastUsedTimestamp > fiveMinutesAgo)
lastUsedMinutesAgo = Math.floor((now - lastUsedTimestamp) / 60)

// 4. 排序逻辑：当前活跃的优先，然后按请求数排序
sort((a, b) => {
  if (a.isCurrentlyActive && !b.isCurrentlyActive) return -1;
  if (!a.isCurrentlyActive && b.isCurrentlyActive) return 1;
  return b.requestCount - a.requestCount;
})
```

**令牌脱敏策略**:
- 只显示tokenMask前8位 + "..."
- 基于tokenMask特征推断环境类型
- 使用哈希确保同一令牌始终显示相同名称

### 22004 - 最近活动

**目的**: 提供系统最近活动的实时动态展示

**核心计算逻辑**:

```typescript
// 1. 获取最近的日志记录
recentLogs = SELECT * FROM log 
  WHERE addtime >= (now - 24*60*60)  -- 最近24小时
  ORDER BY addtime DESC LIMIT {limit * 2}

// 2. 事件类型判断
function determineEventType(log): string {
  if (log.statusCode === 429) return 'rate_limit';
  if (log.statusCode >= 400) return 'error';  
  if (log.tokenMask && log.tokenMask.trim()) return 'token_auth';
  if (log.action && log.action.trim()) return 'tool_request';
  return 'tool_request';
}

// 3. 事件描述生成
function generateEventDescription(log, eventType): string {
  switch (eventType) {
    case 'tool_request':
      if (log.action.includes('web')) return 'Web Server MCP request completed';
      if (log.action.includes('notion')) return 'New Notion page created';
      if (log.action.includes('github')) return 'GitHub repository accessed';
      return `${formatActionName(log.action)} request completed`;
    case 'rate_limit': return 'Rate limit warning';
    case 'error': return `Request failed - ${log.statusCode}`;
    default: return 'System activity';
  }
}

// 4. 详细信息生成
function generateEventDetails(log, eventType): string {
  const minutesAgo = Math.floor((now - log.addtime) / 60);
  const timeStr = formatTimeAgo(minutesAgo);
  
  switch (eventType) {
    case 'tool_request': return `${timeStr} • ${log.duration}ms response`;
    case 'token_auth': return `${timeStr} • ${maskToken(log.tokenMask)}`;
    case 'rate_limit': return `${timeStr} • ${maskToken(log.tokenMask)}`;
    case 'error': return `${timeStr} • ${truncateError(log.error)}`;
  }
}

// 5. 多样性控制（避免同类型事件过多）
eventTypeCounter = new Map();
for (log of recentLogs) {
  const eventType = determineEventType(log);
  const eventCount = eventTypeCounter.get(eventType) || 0;
  
  // 限制同类型事件数量
  if (eventCount >= Math.ceil(limit / 3)) continue;
  
  events.push(createEvent(log, eventType));
  eventTypeCounter.set(eventType, eventCount + 1);
}
```

**活动事件优化**:
- 确保事件类型多样性，避免单一类型占据列表
- 智能的事件描述生成，提高可读性
- 适当的数据脱敏和截断
- 模拟事件补充（当实际数据不足时）

## 数据优化策略

### 查询性能优化

```sql
-- 推荐索引
CREATE INDEX idx_log_addtime_action ON log(addtime, action);
CREATE INDEX idx_log_addtime_tokenMask ON log(addtime, tokenMask);
CREATE INDEX idx_log_addtime_statusCode ON log(addtime, statusCode);
CREATE INDEX idx_log_tokenMask_addtime ON log(tokenMask, addtime);
```

### 缓存策略

```typescript
// 缓存配置建议
interface CacheConfig {
  summary: { ttl: 30, key: 'usage:summary:{timeRange}' },      // 30秒
  topTools: { ttl: 60, key: 'usage:tools:{timeRange}' },      // 1分钟
  activeTokens: { ttl: 30, key: 'usage:tokens:{timeRange}' }, // 30秒
  activity: { ttl: 15, key: 'usage:activity' }                // 15秒
}
```

### 数据聚合优化

```typescript
// 并行查询优化
async function handleProtocol22001(body) {
  const [
    totalRequests24h,
    totalRequestsYesterday,
    activeTokensResult,
    tokensLastHourResult,
    toolsInUseResult,
    topToolResult,
    avgResponseTime24h,
    avgResponseTimeYesterday
  ] = await Promise.all([
    // 8个独立的查询并行执行
    prisma.log.count({ where: { addtime: { gte: yesterday } } }),
    prisma.log.count({ where: { addtime: { gte: dayBeforeYesterday, lt: yesterday } } }),
    // ... 其他查询
  ]);
}
```

## 扩展功能设计

### 22005 - 使用趋势

```typescript
// 趋势数据生成
function generateTrendData(timeRange: number, metricType: string) {
  const interval = timeRange === 1 ? 3600 : 86400; // 1天用小时，7天用天
  const points = timeRange === 1 ? 24 : timeRange;
  
  for (let i = 0; i < points; i++) {
    const pointTime = startTime + (i * interval);
    const pointEndTime = pointTime + interval;
    
    const value = await calculateMetricValue(metricType, pointTime, pointEndTime);
    trendData.push({
      timeLabel: formatTimeLabel(pointTime, timeRange),
      value,
      timestamp: pointTime
    });
  }
}
```

### 22006 - 系统健康状态

```typescript
// 系统健康检查
interface SystemHealth {
  overallStatus: 'healthy' | 'warning' | 'critical';
  services: Array<{
    serviceName: string;
    status: 'online' | 'offline' | 'degraded';
    responseTime: number;
    lastCheck: string;
  }>;
  metrics: {
    cpuUsage: number;    // 需要系统监控集成
    memoryUsage: number; // 需要系统监控集成
    diskUsage: number;   // 需要系统监控集成
    activeConnections: number;
  };
}
```

## 错误处理和监控

### 错误分类

```typescript
// 错误类型定义
enum ApiErrorType {
  DATABASE_CONNECTION = 'database_connection',
  INVALID_PARAMETERS = 'invalid_parameters', 
  PERMISSION_DENIED = 'permission_denied',
  RATE_LIMIT_EXCEEDED = 'rate_limit_exceeded',
  INTERNAL_ERROR = 'internal_error'
}

// 错误处理策略
function handleApiError(error: Error, context: string) {
  const errorType = classifyError(error);
  
  // 记录错误日志
  logger.error('Usage Overview API Error', {
    context,
    errorType,
    message: error.message,
    stack: error.stack,
    timestamp: Date.now()
  });
  
  // 返回用户友好的错误信息
  switch (errorType) {
    case ApiErrorType.DATABASE_CONNECTION:
      throw new Error('数据库连接异常，请稍后重试');
    case ApiErrorType.INVALID_PARAMETERS:
      throw new Error('请求参数无效');
    default:
      throw new Error('系统暂时不可用，请稍后重试');
  }
}
```

### 性能监控

```typescript
// 性能监控装饰器
function monitorPerformance(target: any, propertyName: string, descriptor: PropertyDescriptor) {
  const method = descriptor.value;
  
  descriptor.value = async function (...args: any[]) {
    const startTime = Date.now();
    const apiName = `${target.constructor.name}.${propertyName}`;
    
    try {
      const result = await method.apply(this, args);
      const duration = Date.now() - startTime;
      
      // 记录性能指标
      metrics.recordApiCall(apiName, duration, 'success');
      
      if (duration > 5000) {
        logger.warn('Slow API Call', { apiName, duration });
      }
      
      return result;
    } catch (error) {
      const duration = Date.now() - startTime;
      metrics.recordApiCall(apiName, duration, 'error');
      throw error;
    }
  };
}
```

## 数据质量保证

### 数据验证

```typescript
// 响应数据验证
function validateResponse(data: any, schema: any): boolean {
  // 基本类型检查
  if (typeof data.totalRequests24h !== 'number' || data.totalRequests24h < 0) {
    logger.warn('Invalid totalRequests24h value', { value: data.totalRequests24h });
    data.totalRequests24h = 0;
  }
  
  // 数据合理性检查
  if (data.avgResponseTime > 60000) { // 超过1分钟响应时间
    logger.warn('Unusually high response time', { value: data.avgResponseTime });
  }
  
  // 数据一致性检查
  if (data.activeTokens > data.totalRequests24h) {
    logger.warn('Data consistency issue: activeTokens > totalRequests');
  }
  
  return true;
}
```

### 数据降级策略

```typescript
// 数据降级处理
function applyDataFallback(apiResponse: any, fallbackData: any) {
  return {
    ...fallbackData,
    ...apiResponse,
    // 确保关键字段有默认值
    totalRequests24h: apiResponse.totalRequests24h ?? 0,
    activeTokens: apiResponse.activeTokens ?? 0,
    avgResponseTime: apiResponse.avgResponseTime ?? 0
  };
}
```

## 接口修改指南

### 添加新指标

1. 确定数据来源和计算逻辑
2. 考虑性能影响和缓存策略
3. 实现数据验证和错误处理
4. 更新proto定义和handler注册
5. 添加相应的性能监控

### 优化现有接口

1. 分析查询性能瓶颈
2. 优化SQL查询和索引
3. 实现适当的缓存机制
4. 考虑数据预聚合
5. 监控改进效果

---

## 版本历史

- v1.0 (2024-01-15): 初始版本，实现基础使用概览功能
- 后续版本更新请在此记录...