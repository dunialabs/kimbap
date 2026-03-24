# Logs Management API Design Document

## 概览

本文档详细描述了日志管理接口（23001-23010）的数据计算逻辑和设计思路，用于指导未来的接口修改和优化。这些接口为日志查看、过滤、导出和监控提供全面的功能支持。

## 数据基础

### 数据源表结构

所有日志管理接口基于以下表进行数据分析：

```sql
-- Log表关键字段（主要数据源）
id                     INT     -- 主键，用于分页和增量更新
addtime                BIGINT  -- 添加时间(Unix时间戳)
userid                 VARCHAR -- 用户ID
tokenMask              VARCHAR -- 令牌掩码（用于识别令牌使用）
action                 VARCHAR -- 操作类型（用于识别工具类型和来源）
ip                     VARCHAR -- 客户端IP
ua                     VARCHAR -- 用户代理
statusCode            INT     -- HTTP状态码（用于推断日志级别）
error                 TEXT    -- 错误信息
duration              INT     -- 响应时间(毫秒)
sessionId             VARCHAR -- 会话ID/请求ID
```

### 日志级别推断规则

由于当前Log表结构没有直接的level字段，我们基于statusCode推断日志级别：

```typescript
function inferLogLevel(statusCode: number): string {
  if (statusCode >= 500) return "ERROR";    // 服务器错误
  if (statusCode >= 400) return "WARN";     // 客户端错误/警告
  if (statusCode >= 200) return "INFO";     // 成功请求
  return "DEBUG";                           // 其他状态
}
```

### 日志来源推断规则

基于action字段内容推断日志来源：

```typescript
function inferSource(action: string): string {
  const actionLower = action.toLowerCase();
  
  if (actionLower.includes('web') || actionLower.includes('http')) {
    return 'api-gateway';
  } else if (actionLower.includes('tool') || actionLower.includes('mcp')) {
    return 'tool-manager';
  } else if (actionLower.includes('auth') || actionLower.includes('token')) {
    return 'auth-service';
  } else if (actionLower.includes('database') || actionLower.includes('db')) {
    return 'database';
  } else if (actionLower.includes('cache')) {
    return 'cache';
  } else if (actionLower.includes('monitor')) {
    return 'monitor';
  } else {
    return 'system';
  }
}
```

## 接口详细设计

### 23001 - 获取日志列表（支持多种过滤条件）

**目的**: 提供功能完整的日志查询接口，支持分页、过滤和搜索

**核心查询逻辑**:

```typescript
// 1. 时间范围过滤
const timeRangeMap = {
  "1h": now - (60 * 60),
  "6h": now - (6 * 60 * 60),
  "24h": now - (24 * 60 * 60),
  "7d": now - (7 * 24 * 60 * 60),
  "all": 0
};

whereCondition.addtime = { gte: BigInt(timeRangeMap[timeRange]) };

// 2. 级别过滤（基于statusCode）
switch (level) {
  case "ERROR": whereCondition.statusCode = { gte: 500 }; break;
  case "WARN": whereCondition.statusCode = { gte: 400, lt: 500 }; break;
  case "INFO": whereCondition.statusCode = { gte: 200, lt: 400 }; break;
  case "DEBUG": whereCondition.statusCode = { lt: 200 }; break;
}

// 3. 来源过滤（基于action字段）
if (source !== "all") {
  whereCondition.action = { contains: source };
}

// 4. 多字段搜索
if (search) {
  whereCondition.OR = [
    { action: { contains: search, mode: 'insensitive' } },
    { error: { contains: search, mode: 'insensitive' } },
    { ua: { contains: search, mode: 'insensitive' } }
  ];
}

// 5. 分页查询
const logs = await prisma.log.findMany({
  where: whereCondition,
  orderBy: { addtime: 'desc' },
  skip: (page - 1) * pageSize,
  take: pageSize
});
```

**日志消息生成策略**:

```typescript
function generateLogMessage(log: LogEntry): string {
  const level = inferLogLevel(log.statusCode);
  const action = log.action || 'unknown_action';
  
  switch (level) {
    case "ERROR":
      return `${action} request failed - ${log.error || 'Unknown error'}`;
    case "WARN":
      return `${action} completed with warnings - Status ${log.statusCode}`;
    case "INFO":
      return `${action} request processed successfully`;
    case "DEBUG":
      return `${action} debug information logged`;
    default:
      return `${action} system activity`;
  }
}
```

**原始日志数据格式化**:

```typescript
function generateRawData(log: LogEntry): string {
  const timestamp = new Date(Number(log.addtime) * 1000).toISOString().replace('T', ' ').slice(0, -5);
  const level = inferLogLevel(log.statusCode);
  const source = inferSource(log.action || '');
  const message = generateLogMessage(log);
  
  let rawData = `[${timestamp}] [${level}] [${source}] ${message}\n`;
  
  // 添加详细信息
  if (log.sessionId) rawData += `Request ID: ${log.sessionId}\n`;
  if (log.userid) rawData += `User ID: ${log.userid}\n`;
  if (log.action) rawData += `Action: ${log.action}\n`;
  if (log.statusCode) rawData += `Status: ${log.statusCode}\n`;
  if (log.duration) rawData += `Response Time: ${log.duration}ms\n`;
  if (log.ua) rawData += `User Agent: ${log.ua}\n`;
  if (log.ip) rawData += `IP: ${log.ip}\n`;
  if (log.tokenMask) rawData += `Token: ${log.tokenMask.substring(0, 8)}...\n`;
  if (log.error) rawData += `Error: ${log.error}\n`;
  
  return rawData;
}
```

### 23002 - 获取日志统计信息

**目的**: 提供日志的统计概览，用于监控和报表

**核心统计逻辑**:

```typescript
// 1. 级别统计
let errorLogs = 0, warnLogs = 0, infoLogs = 0, debugLogs = 0;

allLogs.forEach(log => {
  const level = inferLogLevel(log.statusCode);
  switch (level) {
    case "ERROR": errorLogs++; break;
    case "WARN": warnLogs++; break;
    case "INFO": infoLogs++; break;
    case "DEBUG": debugLogs++; break;
  }
});

// 2. 错误率计算
const errorRate = totalLogs > 0 ? (errorLogs / totalLogs) * 100 : 0;

// 3. 按来源统计
const sourceMap = new Map<string, { total: number; errors: number }>();

allLogs.forEach(log => {
  const source = inferSource(log.action || '');
  const isError = log.statusCode >= 500;
  
  if (!sourceMap.has(source)) {
    sourceMap.set(source, { total: 0, errors: 0 });
  }
  
  const stats = sourceMap.get(source)!;
  stats.total++;
  if (isError) stats.errors++;
});

// 4. 按小时统计（最近24小时）
const hourlyMap = new Map<number, { total: number; errors: number }>();

// 初始化24小时数据结构
for (let i = 0; i < 24; i++) {
  const hourTimestamp = twentyFourHoursAgo + (i * 60 * 60);
  const hourKey = Math.floor(hourTimestamp / 3600) * 3600;
  hourlyMap.set(hourKey, { total: 0, errors: 0 });
}

// 统计实际数据
allLogs.forEach(log => {
  const logTime = Number(log.addtime);
  if (logTime >= twentyFourHoursAgo) {
    const hourKey = Math.floor(logTime / 3600) * 3600;
    const isError = log.statusCode >= 500;
    
    if (hourlyMap.has(hourKey)) {
      const stats = hourlyMap.get(hourKey)!;
      stats.total++;
      if (isError) stats.errors++;
    }
  }
});
```

### 23004 - 导出日志

**目的**: 支持多种格式的日志导出功能

**导出格式处理**:

```typescript
// 1. TXT格式导出
case 1: {
  fileContent = logs.map(log => generateRawData(log)).join('\n\n');
  formatName = 'TXT';
  fileExtension = 'txt';
  break;
}

// 2. JSON格式导出
case 2: {
  const jsonData = logs.map(log => ({
    id: log.id.toString(),
    timestamp: new Date(Number(log.addtime) * 1000).toISOString(),
    level: inferLogLevel(log.statusCode),
    message: generateLogMessage(log),
    source: inferSource(log.action || ''),
    details: {
      action: log.action,
      statusCode: log.statusCode,
      responseTime: log.duration,
      userAgent: log.ua,
      ip: log.ip,
      tokenId: log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : null,
      error: log.error
    },
    rawData: generateRawData(log)
  }));
  
  fileContent = JSON.stringify({
    exportInfo: {
      exportTime: new Date().toISOString(),
      timeRange,
      filters: { level, source, search },
      totalRecords: logs.length
    },
    logs: jsonData
  }, null, 2);
  
  formatName = 'JSON';
  fileExtension = 'json';
  break;
}

// 3. CSV格式导出
case 3: {
  let csvContent = 'ID,Timestamp,Level,Source,Message,Status Code,Response Time,User ID,Request ID,IP,User Agent,Token ID,Error\n';
  
  logs.forEach(log => {
    const timestamp = new Date(Number(log.addtime) * 1000).toISOString();
    const level = inferLogLevel(log.statusCode);
    const source = inferSource(log.action || '');
    const message = generateLogMessage(log).replace(/"/g, '""'); // Escape quotes
    
    const row = [
      log.id.toString(),
      timestamp,
      level,
      source,
      `"${message}"`,
      log.statusCode.toString(),
      (log.duration || 0).toString(),
      log.userid || '',
      log.sessionId || '',
      log.ip || '',
      `"${log.ua || ''}"`,
      log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : '',
      `"${log.error || ''}"`
    ].join(',');
    
    csvContent += row + '\n';
  });
  
  fileContent = csvContent;
  formatName = 'CSV';
  fileExtension = 'csv';
  break;
}
```

### 23005 - 获取实时日志

**目的**: 支持实时日志流更新，减少客户端轮询开销

**增量获取策略**:

```typescript
// 1. 基于ID的增量查询
const whereCondition = {
  id: { gt: lastLogId }  // 只获取ID大于lastLogId的新日志
};

// 2. 按ID升序获取，确保时间顺序
const logs = await prisma.log.findMany({
  where: whereCondition,
  orderBy: { id: 'asc' },  // 注意：升序获取
  take: limit
});

// 3. 检查是否还有更多数据
const hasMore = logs.length === limit;
const latestLogId = logs.length > 0 ? Math.max(...logs.map(log => log.id)) : lastLogId;

// 4. 返回增量数据
return {
  newLogs: transformedLogs,
  latestLogId,
  hasMore
};
```

**实时更新优化**:
- 使用ID而非时间戳避免重复数据
- 支持级别和来源过滤减少传输量
- 限制单次返回数量避免大数据传输
- 客户端维护lastLogId状态

## 性能优化策略

### 数据库索引优化

```sql
-- 核心查询索引
CREATE INDEX idx_log_addtime_statuscode ON log(addtime, statusCode);
CREATE INDEX idx_log_action_addtime ON log(action, addtime);
CREATE INDEX idx_log_sessionid ON log(sessionId);
CREATE INDEX idx_log_userid_addtime ON log(userid, addtime);

-- 实时更新索引
CREATE INDEX idx_log_id_statuscode ON log(id, statusCode);
CREATE INDEX idx_log_id_action ON log(id, action);

-- 搜索优化索引
CREATE INDEX idx_log_error_gin ON log USING gin(to_tsvector('english', error));
CREATE INDEX idx_log_ua_gin ON log USING gin(to_tsvector('english', ua));
```

### 查询优化策略

```typescript
// 1. 分页查询优化
interface OptimizedPagination {
  // 使用游标分页替代OFFSET
  cursorId?: number;
  pageSize: number;
  
  // 构建查询条件
  buildWhereCondition(): any {
    const where: any = { /* 基础条件 */ };
    
    if (this.cursorId) {
      where.id = { lt: this.cursorId }; // 游标分页
    }
    
    return where;
  }
}

// 2. 批量数据处理
async function fetchLogsBatch(whereCondition: any, batchSize = 1000) {
  const batches = [];
  let lastId = 0;
  
  while (true) {
    const batch = await prisma.log.findMany({
      where: { ...whereCondition, id: { gt: lastId } },
      orderBy: { id: 'asc' },
      take: batchSize
    });
    
    if (batch.length === 0) break;
    
    batches.push(batch);
    lastId = batch[batch.length - 1].id;
  }
  
  return batches.flat();
}

// 3. 并行统计查询
async function getLogStatisticsParallel(whereCondition: any) {
  const [
    totalCount,
    errorCount,
    warnCount,
    infoCount,
    debugCount,
    sourceStats
  ] = await Promise.all([
    prisma.log.count({ where: whereCondition }),
    prisma.log.count({ where: { ...whereCondition, statusCode: { gte: 500 } } }),
    prisma.log.count({ where: { ...whereCondition, statusCode: { gte: 400, lt: 500 } } }),
    prisma.log.count({ where: { ...whereCondition, statusCode: { gte: 200, lt: 400 } } }),
    prisma.log.count({ where: { ...whereCondition, statusCode: { lt: 200 } } }),
    getSourceStatistics(whereCondition)
  ]);
  
  return { totalCount, errorCount, warnCount, infoCount, debugCount, sourceStats };
}
```

### 缓存策略

```typescript
// 1. 统计数据缓存
interface LogsCacheConfig {
  statistics: { ttl: 300, key: 'logs:stats:{timeRange}' },     // 5分钟
  sources: { ttl: 600, key: 'logs:sources:{timeRange}' },      // 10分钟
  hourlyStats: { ttl: 900, key: 'logs:hourly:{date}' },       // 15分钟
}

// 2. Redis缓存实现
class LogsStatsCache {
  async getStatistics(timeRange: string): Promise<LogStatistics | null> {
    const key = `logs:stats:${timeRange}`;
    const cached = await redis.get(key);
    return cached ? JSON.parse(cached) : null;
  }
  
  async setStatistics(timeRange: string, stats: LogStatistics): Promise<void> {
    const key = `logs:stats:${timeRange}`;
    await redis.setex(key, 300, JSON.stringify(stats));
  }
}

// 3. 应用层缓存
const statisticsCache = new Map<string, { data: LogStatistics; expiry: number }>();

function getCachedStatistics(timeRange: string): LogStatistics | null {
  const cached = statisticsCache.get(timeRange);
  if (cached && cached.expiry > Date.now()) {
    return cached.data;
  }
  return null;
}
```

## 扩展功能设计

### 高级搜索 (23009)

```typescript
// 1. 全文搜索支持
interface AdvancedSearch {
  query: string;                    // 搜索查询语句
  fields: string[];                 // 搜索字段
  timeStart: string;                // 精确时间范围
  timeEnd: string;
  levels: string[];                 // 多级别选择
  sources: string[];                // 多来源选择
  sortBy: 'timestamp' | 'level' | 'source';
  sortOrder: 'asc' | 'desc';
}

// 2. 搜索查询构建
function buildAdvancedSearchQuery(params: AdvancedSearch) {
  const whereConditions = [];
  
  // 时间范围
  if (params.timeStart && params.timeEnd) {
    whereConditions.push({
      addtime: {
        gte: BigInt(new Date(params.timeStart).getTime() / 1000),
        lte: BigInt(new Date(params.timeEnd).getTime() / 1000)
      }
    });
  }
  
  // 多字段搜索
  if (params.query && params.fields.length > 0) {
    const searchConditions = params.fields.map(field => ({
      [field]: { contains: params.query, mode: 'insensitive' }
    }));
    whereConditions.push({ OR: searchConditions });
  }
  
  // 多级别过滤
  if (params.levels.length > 0) {
    const levelConditions = params.levels.map(level => {
      switch (level) {
        case "ERROR": return { statusCode: { gte: 500 } };
        case "WARN": return { statusCode: { gte: 400, lt: 500 } };
        case "INFO": return { statusCode: { gte: 200, lt: 400 } };
        case "DEBUG": return { statusCode: { lt: 200 } };
      }
    }).filter(Boolean);
    
    if (levelConditions.length > 0) {
      whereConditions.push({ OR: levelConditions });
    }
  }
  
  return { AND: whereConditions };
}
```

### 审计跟踪 (23010)

```typescript
// 1. 实体审计跟踪
interface AuditTrail {
  entityType: 'user' | 'token' | 'request' | 'session';
  entityId: string;
  
  // 生成审计事件
  generateAuditEvents(logs: LogEntry[]): AuditEvent[] {
    return logs.map(log => ({
      timestamp: new Date(Number(log.addtime) * 1000).toISOString(),
      eventType: this.inferEventType(log),
      action: log.action || 'unknown',
      source: inferSource(log.action || ''),
      level: inferLogLevel(log.statusCode),
      details: this.generateEventDetails(log),
      relatedLogId: log.id.toString()
    }));
  }
  
  // 推断事件类型
  inferEventType(log: LogEntry): string {
    if (log.action?.includes('auth')) return 'authentication';
    if (log.action?.includes('access')) return 'data_access';
    if (log.statusCode >= 500) return 'system_error';
    if (log.statusCode >= 400) return 'client_error';
    return 'normal_operation';
  }
}

// 2. 审计摘要生成
function generateAuditSummary(events: AuditEvent[]): AuditSummary {
  const errorEvents = events.filter(e => e.level === 'ERROR').length;
  const timestamps = events.map(e => new Date(e.timestamp).getTime());
  const involvedSources = Array.from(new Set(events.map(e => e.source)));
  
  return {
    totalEvents: events.length,
    errorEvents,
    firstActivity: new Date(Math.min(...timestamps)).toISOString(),
    lastActivity: new Date(Math.max(...timestamps)).toISOString(),
    involvedSources
  };
}
```

## 错误处理和监控

### 错误分类和处理

```typescript
// 1. 错误类型定义
enum LogsApiErrorType {
  INVALID_TIME_RANGE = 'invalid_time_range',
  INVALID_PAGINATION = 'invalid_pagination',
  SEARCH_QUERY_TOO_COMPLEX = 'search_query_too_complex',
  EXPORT_SIZE_EXCEEDED = 'export_size_exceeded',
  DATABASE_TIMEOUT = 'database_timeout'
}

// 2. 错误处理策略
class LogsApiErrorHandler {
  static handle(error: Error, context: string): never {
    const errorType = this.classifyError(error);
    
    switch (errorType) {
      case LogsApiErrorType.INVALID_TIME_RANGE:
        throw new Error('时间范围参数无效，请检查timeRange值');
      case LogsApiErrorType.INVALID_PAGINATION:
        throw new Error('分页参数无效，页码必须大于0');
      case LogsApiErrorType.SEARCH_QUERY_TOO_COMPLEX:
        throw new Error('搜索查询过于复杂，请简化搜索条件');
      case LogsApiErrorType.EXPORT_SIZE_EXCEEDED:
        throw new Error('导出数据量超过限制，请缩小时间范围或添加过滤条件');
      case LogsApiErrorType.DATABASE_TIMEOUT:
        throw new Error('数据库查询超时，请稍后重试或联系管理员');
      default:
        throw new Error('系统暂时不可用，请稍后重试');
    }
  }
  
  private static classifyError(error: Error): LogsApiErrorType {
    if (error.message.includes('timeout')) {
      return LogsApiErrorType.DATABASE_TIMEOUT;
    }
    if (error.message.includes('invalid time')) {
      return LogsApiErrorType.INVALID_TIME_RANGE;
    }
    // ... 其他错误分类逻辑
    return LogsApiErrorType.DATABASE_TIMEOUT; // 默认
  }
}
```

### 性能监控

```typescript
// 1. 查询性能监控
class LogsQueryMonitor {
  static async monitorQuery<T>(
    queryName: string,
    queryFn: () => Promise<T>
  ): Promise<T> {
    const startTime = Date.now();
    
    try {
      const result = await queryFn();
      const duration = Date.now() - startTime;
      
      // 记录查询性能
      this.recordQueryMetrics(queryName, duration, 'success');
      
      // 慢查询告警
      if (duration > 5000) {
        console.warn(`Slow logs query detected: ${queryName} took ${duration}ms`);
      }
      
      return result;
    } catch (error) {
      const duration = Date.now() - startTime;
      this.recordQueryMetrics(queryName, duration, 'error');
      throw error;
    }
  }
  
  private static recordQueryMetrics(queryName: string, duration: number, status: string) {
    // 实现指标记录逻辑（如发送到监控系统）
    console.log(`Query: ${queryName}, Duration: ${duration}ms, Status: ${status}`);
  }
}
```

## 数据安全和隐私

### 敏感数据处理

```typescript
// 1. IP地址脱敏
function maskIPAddress(ip: string): string {
  if (!ip) return '';
  
  const parts = ip.split('.');
  if (parts.length === 4) {
    return `${parts[0]}.${parts[1]}.xxx.xxx`;
  }
  
  // IPv6处理
  if (ip.includes(':')) {
    const parts = ip.split(':');
    return parts.slice(0, 4).join(':') + '::xxxx';
  }
  
  return 'xxx.xxx.xxx.xxx';
}

// 2. 用户代理脱敏
function maskUserAgent(ua: string): string {
  if (!ua) return '';
  
  // 移除可能的敏感信息（如特定版本号、设备标识）
  return ua.substring(0, 100) + (ua.length > 100 ? '...' : '');
}

// 3. Token脱敏
function maskToken(token: string): string {
  if (!token || token.length <= 8) return token;
  return token.substring(0, 8) + '...';
}

// 4. 错误信息脱敏
function sanitizeErrorMessage(error: string): string {
  if (!error) return '';
  
  // 移除可能包含敏感信息的路径
  const sanitized = error
    .replace(/\/home\/[^\/]+/g, '/home/***')
    .replace(/\/Users\/[^\/]+/g, '/Users/***')
    .replace(/password=[^&\s]+/gi, 'password=***')
    .replace(/token=[^&\s]+/gi, 'token=***');
  
  return sanitized;
}
```

### 访问控制

```typescript
// 1. 基于角色的日志访问控制
interface LogsAccessControl {
  canViewLogs(userRole: number, logUserId?: string, requestUserId?: string): boolean {
    // Owner和Admin可以查看所有日志
    if (userRole <= 2) return true;
    
    // Member只能查看自己相关的日志
    if (userRole === 3 && logUserId) {
      return logUserId === requestUserId;
    }
    
    // Member可以查看无用户关联的系统日志
    if (userRole === 3 && !logUserId) return true;
    
    return false;
  }
  
  canExportLogs(userRole: number): boolean {
    // 只有Owner和Admin可以导出日志
    return userRole <= 2;
  }
  
  getMaxExportRecords(userRole: number): number {
    switch (userRole) {
      case 1: return 100000; // Owner
      case 2: return 50000;  // Admin
      case 3: return 10000;  // Member
      default: return 1000;
    }
  }
}
```

## 接口修改指南

### 添加新的日志字段

1. **扩展Log表结构**：添加新字段并创建相应索引
2. **更新推断逻辑**：修改级别和来源推断函数
3. **调整响应格式**：更新protobuf定义和响应结构
4. **兼容性处理**：确保新字段为可选，保持向后兼容

### 优化查询性能

1. **索引优化**：分析慢查询并添加合适的索引
2. **查询重写**：优化复杂查询的SQL结构
3. **缓存策略**：为常用查询添加缓存
4. **分片策略**：考虑按时间分片存储历史日志

### 扩展搜索功能

1. **全文搜索**：集成Elasticsearch或PostgreSQL全文搜索
2. **搜索建议**：实现搜索自动完成和建议
3. **查询优化**：添加查询分析和优化
4. **结果高亮**：在搜索结果中高亮匹配文本

---

## 版本历史

- v1.0 (2024-01-15): 初始版本，实现基础日志管理功能
- 后续版本更新请在此记录...