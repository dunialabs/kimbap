# Token Usage Statistics API Design Document

## 概览

本文档详细描述了访问令牌使用统计接口（21001-21010）的数据计算逻辑和设计思路，用于指导未来的接口修改和优化。

## 数据基础

### 数据源表结构

所有统计接口基于以下表进行数据分析：

```sql
-- User表关键字段（访问令牌信息）
userid                 VARCHAR -- 用户ID（作为令牌标识）
accessToken           VARCHAR -- 访问令牌
role                  INT     -- 用户角色 (1-owner, 2-admin, 3-member)

-- Log表关键字段（令牌使用日志）
id                     INT     -- 主键
addtime                BIGINT  -- 添加时间(Unix时间戳)
userid                 VARCHAR -- 用户ID
tokenMask              VARCHAR -- 令牌掩码（用于识别令牌使用）
ip                     VARCHAR -- 客户端IP
ua                     VARCHAR -- 用户代理
statusCode            INT     -- HTTP状态码
error                 TEXT    -- 错误信息
duration              INT     -- 响应时间(毫秒)
sessionId             VARCHAR -- 会话ID
```

### 令牌识别机制

由于当前数据结构中没有直接的令牌使用记录表，我们通过以下方式识别令牌使用：

```typescript
// 令牌识别逻辑
1. 通过Log表的tokenMask字段识别令牌使用
2. 通过User表的accessToken字段获取令牌信息
3. 使用tokenMask前缀匹配关联令牌和日志记录
4. 对tokenId进行脱敏处理（只显示前8位+...）
```

## 接口详细设计

### 21001 - 访问令牌使用汇总统计

**目的**: 提供系统整体的令牌使用概览数据

**核心计算逻辑**:

```typescript
// 1. 总令牌数 - 从user表统计
totalTokens = COUNT(*) FROM user

// 2. 活跃令牌数 - 时间范围内有请求的令牌
activeTokens = COUNT(DISTINCT tokenMask) FROM log 
               WHERE addtime >= startTime AND tokenMask != ''

// 3. 请求统计
totalRequests = COUNT(*) FROM log WHERE 时间范围 AND tokenMask != ''
successRequests = COUNT(*) FROM log WHERE 时间范围 AND statusCode BETWEEN 200-299
failedRequests = totalRequests - successRequests

// 4. 成功率
avgSuccessRate = (successRequests / totalRequests) * 100

// 5. 独立客户端数
totalClients = COUNT(DISTINCT ip) FROM log WHERE 时间范围 AND tokenMask != ''

// 6. 过期令牌数和限制令牌数（需要扩展数据结构）
expiredTokens = 0  // 需要在user表中添加expireAt字段
limitedTokens = 0  // 需要统计速率限制记录
```

**设计考量**:
- 通过tokenMask字段识别令牌使用，避免直接使用敏感的accessToken
- 客户端数基于IP去重，提供连接规模概览
- 过期和限制统计需要后续扩展数据结构

### 21002 - 令牌详细指标

**目的**: 提供每个令牌的详细使用指标，支持分页

**核心计算逻辑**:

```typescript
// 1. 获取所有用户作为令牌列表
users = SELECT userid, accessToken, role FROM user

// 2. 为每个令牌计算指标
FOR EACH user IN users:
  tokenMask = user.accessToken.substring(0, 16)  // 用于日志匹配
  
  // 请求统计
  tokenRequests = COUNT(*) WHERE tokenMask = tokenMask AND 时间范围
  tokenSuccess = COUNT(*) WHERE tokenMask = tokenMask AND statusCode 200-299
  tokenFailed = tokenRequests - tokenSuccess
  
  // 客户端数量
  clientCount = COUNT(DISTINCT ip) WHERE tokenMask = tokenMask AND 时间范围
  
  // 最后使用时间
  lastUsed = MAX(addtime) WHERE tokenMask = tokenMask
  
  // 状态判断
  status = calculateTokenStatus(tokenRequests, lastUsed)
  
  // 地理位置分布（模拟）
  topLocations = generateMockGeoData(tokenRequests)
```

**令牌状态判断逻辑**:

```typescript
function calculateTokenStatus(totalRequests: number, lastUsed: number): number {
  const now = currentTimestamp;
  const oneDayAgo = now - (24 * 60 * 60);
  
  if (totalRequests === 0) {
    return 2; // inactive
  }
  
  if (lastUsed > oneDayAgo) {
    return 1; // active
  } else {
    return 2; // inactive
  }
  
  // 需要根据实际业务规则判断 expired(3) 和 limited(4) 状态
}
```

**地理位置数据模拟**:

```typescript
// 由于缺少IP地理位置库，目前使用模拟数据
function generateMockGeoData(totalRequests: number): Location[] {
  const geoTemplates = [
    { country: 'US', city: 'New York', weight: 0.4 },
    { country: 'UK', city: 'London', weight: 0.3 },
    { country: 'DE', city: 'Berlin', weight: 0.2 },
    { country: 'JP', city: 'Tokyo', weight: 0.1 }
  ];
  
  return geoTemplates.map(geo => ({
    country: geo.country,
    city: geo.city,
    requests: Math.floor(totalRequests * geo.weight),
    percentage: geo.weight * 100
  })).filter(loc => loc.requests > 0);
}
```

### 21003 - 令牌使用趋势

**目的**: 提供时间序列数据用于趋势分析

**时间粒度设计**:

```typescript
// 粒度配置
granularity: {
  1: { interval: 3600, format: "YYYY-MM-DD HH:00" },     // 按小时
  2: { interval: 86400, format: "YYYY-MM-DD" },          // 按天(默认)
  3: { interval: 604800, format: "YYYY-MM-DD" }          // 按周
}

// 时间点生成
timePoints = []
for (time = startTime; time <= now; time += interval) {
  timePoints.push(time)
}

// 数据聚合
FOR EACH timePoint:
  periodLogs = logs WHERE addtime BETWEEN timePoint AND timePoint+interval
  
  FOR EACH tokenMask:
    tokenLogs = periodLogs WHERE tokenMask = tokenMask
    requestCount = tokenLogs.length
    successCount = COUNT(tokenLogs WHERE statusCode 200-299)
    failedCount = requestCount - successCount
    rateLimitCount = COUNT(tokenLogs WHERE statusCode = 429)
```

**设计考量**:
- 支持指定令牌列表，空则统计所有令牌
- 数据结构便于前端绘制多线图表
- 包含速率限制统计便于监控

### 21004 - 令牌地理位置使用分布

**目的**: 分析令牌使用的地理分布，帮助了解访问模式

**地理位置分析逻辑**:

```typescript
// 1. 获取令牌相关的IP数据
tokenIPs = SELECT tokenMask, ip FROM log 
           WHERE 时间范围 AND tokenMask != '' AND ip != ''

// 2. 按令牌分组IP统计
FOR EACH token:
  ipCounts = GROUP BY ip, COUNT(*) WHERE tokenMask = token
  
  // 3. IP地理位置映射（当前为模拟数据）
  locations = []
  FOR EACH ip IN ipCounts:
    geoInfo = lookupIPGeolocation(ip)  // 需要IP地理库
    locations.add({
      country: geoInfo.country,
      city: geoInfo.city,
      requests: ipCounts[ip],
      uniqueIPs: 1
    })
  
  // 4. 按地理位置聚合
  geoStats = GROUP BY (country, city), SUM(requests), COUNT(DISTINCT ip)
```

**地理位置数据来源**:
- **当前实现**: 使用权重分布的模拟数据
- **未来扩展**: 集成IP地理位置库（如MaxMind GeoIP2）
- **隐私考虑**: IP地址脱敏处理，只显示地理位置统计

### 21005 - 令牌使用模式

**目的**: 提供特定令牌的时间使用模式分析

**模式类型设计**:

```typescript
// 1. 最近60分钟模式 (patternType = 1)
startTime = now - 3600
interval = 60      // 1分钟间隔
points = 60

// 2. 最近24小时模式 (patternType = 2)  
startTime = now - 86400
interval = 3600    // 1小时间隔
points = 24

// 3. 最近7天每小时模式 (patternType = 3)
startTime = now - (7 * 86400)
interval = 3600    // 1小时间隔
points = 7 * 24

// 使用模式数据生成
FOR i = 0 TO points:
  timePoint = startTime + (i * interval)
  timePointEnd = timePoint + interval
  
  periodLogs = logs WHERE tokenMask = tokenMask 
                   AND addtime BETWEEN timePoint AND timePointEnd
  
  patterns[i] = {
    timeLabel: formatTime(timePoint),
    requests: periodLogs.length,
    successRequests: COUNT(periodLogs WHERE statusCode 200-299),
    failedRequests: COUNT(periodLogs WHERE statusCode NOT BETWEEN 200-299),
    rateLimitHits: COUNT(periodLogs WHERE statusCode = 429)
  }
```

**时间格式化**:

```typescript
function formatTime(timestamp: number, patternType: number): string {
  const date = new Date(timestamp * 1000);
  
  switch (patternType) {
    case 1: // 分钟级
      return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
    case 2: // 小时级
      return `${date.getHours().toString().padStart(2, '0')}:00`;
    case 3: // 7天小时级
      return `${date.getMonth() + 1}-${date.getDate().toString().padStart(2, '0')} ${date.getHours().toString().padStart(2, '0')}:00`;
  }
}
```

### 21006 - 令牌使用分布

**目的**: 提供饼图数据展示令牌使用分布

**分布类型设计**:

```typescript
// 1. 按请求数分布 (metricType = 1)
distribution = GROUP BY tokenMask, COUNT(*) as requestCount
percentage = (requestCount / totalRequests) * 100

// 2. 按客户端数分布 (metricType = 2)
FOR EACH token:
  uniqueClients = COUNT(DISTINCT ip) WHERE tokenMask = token
  percentage = (uniqueClients / totalClients) * 100

// 3. 按成功率分布 (metricType = 3)
FOR EACH token:
  totalRequests = COUNT(*) WHERE tokenMask = token
  successRequests = COUNT(*) WHERE tokenMask = token AND statusCode 200-299
  successRate = (successRequests / totalRequests) * 100
  // 使用相对百分比显示分布
  percentage = (successRate / maxSuccessRate) * 100
```

**设计考量**:
- 请求数分布反映令牌活跃度
- 客户端数分布反映令牌使用广度
- 成功率分布使用相对比例便于比较

## 数据扩展建议

### 令牌管理表扩展

```sql
-- 建议添加令牌管理表
CREATE TABLE access_tokens (
  id SERIAL PRIMARY KEY,
  token_id VARCHAR(64) UNIQUE NOT NULL,
  user_id VARCHAR(256) NOT NULL,
  token_name VARCHAR(128),
  rate_limit INT DEFAULT 100,
  created_at BIGINT NOT NULL,
  expires_at BIGINT DEFAULT 0,  -- 0表示永不过期
  status INT DEFAULT 1,         -- 1-active, 2-inactive, 3-expired
  last_used_at BIGINT DEFAULT 0,
  notes TEXT
);

-- 令牌使用记录表
CREATE TABLE token_usage_logs (
  id SERIAL PRIMARY KEY,
  token_id VARCHAR(64) NOT NULL,
  client_ip VARCHAR(45),
  user_agent TEXT,
  country_code VARCHAR(2),
  city VARCHAR(100),
  request_count INT DEFAULT 1,
  success_count INT DEFAULT 0,
  failed_count INT DEFAULT 0,
  rate_limit_hits INT DEFAULT 0,
  logged_at BIGINT NOT NULL,
  
  INDEX idx_token_time (token_id, logged_at),
  INDEX idx_time_country (logged_at, country_code)
);
```

### 地理位置服务集成

```typescript
// IP地理位置查询服务
interface GeoLocationService {
  lookupIP(ip: string): Promise<{
    country: string;
    countryName: string;
    city: string;
    latitude: number;
    longitude: number;
  }>;
}

// 实现示例
class MaxMindGeoService implements GeoLocationService {
  async lookupIP(ip: string) {
    // 集成MaxMind GeoIP2库
    // 处理IP脱敏和缓存
  }
}
```

### 速率限制监控

```typescript
// 速率限制记录表
CREATE TABLE rate_limit_events (
  id SERIAL PRIMARY KEY,
  token_id VARCHAR(64) NOT NULL,
  client_ip VARCHAR(45),
  requests_in_window INT,
  window_start BIGINT,
  blocked_requests INT,
  created_at BIGINT NOT NULL
);

// 速率限制分析接口
interface RateLimitAnalysis {
  tokenId: string;
  configuredLimit: number;
  totalHits: number;
  peakHitsPerMinute: number;
  recentEvents: RateLimitEvent[];
}
```

## 性能优化策略

### 数据库索引优化

```sql
-- 推荐索引
CREATE INDEX idx_log_tokenMask_addtime ON log(tokenMask, addtime);
CREATE INDEX idx_log_addtime_status ON log(addtime, statusCode);
CREATE INDEX idx_log_tokenMask_ip ON log(tokenMask, ip);
CREATE INDEX idx_user_role ON user(role);
```

### 查询优化

1. **时间范围预过滤**: 所有查询都先按时间范围过滤
2. **令牌掩码索引**: 使用tokenMask字段索引提高查询效率
3. **分页优化**: 大数据量接口采用游标分页
4. **并行查询**: 使用Promise.all()并行执行独立查询

### 缓存策略

```typescript
// 缓存配置
interface CacheConfig {
  summary: { ttl: 300, key: 'token:summary:{timeRange}' },      // 5分钟
  trends: { ttl: 600, key: 'token:trends:{timeRange}' },        // 10分钟
  geoData: { ttl: 1800, key: 'token:geo:{timeRange}' },        // 30分钟
  patterns: { ttl: 60, key: 'token:pattern:{tokenId}:{type}' }  // 1分钟
}

// Redis缓存实现
class TokenStatsCache {
  async get(key: string) {
    return await redis.get(key);
  }
  
  async set(key: string, data: any, ttl: number) {
    await redis.setex(key, ttl, JSON.stringify(data));
  }
}
```

### 实时数据处理

```typescript
// 实时统计更新
interface RealtimeProcessor {
  // 处理新的日志记录
  processLogEntry(logEntry: LogEntry): void;
  
  // 更新令牌使用统计
  updateTokenStats(tokenId: string): void;
  
  // 检查速率限制
  checkRateLimit(tokenId: string, clientIP: string): boolean;
}
```

## 安全和隐私考虑

### 数据脱敏

```typescript
// 令牌ID脱敏
function maskTokenId(tokenId: string): string {
  if (tokenId.length <= 8) return tokenId;
  return tokenId.substring(0, 8) + '...';
}

// IP地址脱敏
function maskIPAddress(ip: string): string {
  const parts = ip.split('.');
  if (parts.length === 4) {
    return `${parts[0]}.${parts[1]}.xxx.xxx`;
  }
  return 'xxx.xxx.xxx.xxx';
}

// 用户代理脱敏
function maskUserAgent(ua: string): string {
  return ua.substring(0, 50) + (ua.length > 50 ? '...' : '');
}
```

### 权限控制

```typescript
// 基于角色的数据访问控制
interface AccessControl {
  // Owner: 可查看所有令牌统计
  // Admin: 可查看所有令牌统计  
  // Member: 只能查看自己的令牌统计
  
  canViewToken(userRole: number, tokenUserId: string, requestUserId: string): boolean {
    if (userRole <= 2) return true; // Owner/Admin
    return tokenUserId === requestUserId; // Member只能看自己的
  }
  
  canViewGeoData(userRole: number): boolean {
    return userRole <= 2; // 只有Owner/Admin可以查看地理位置数据
  }
}
```

### 审计日志

```typescript
// 访问审计记录
interface AuditLog {
  logId: string;
  tokenId: string;
  eventType: number;  // 1-查看统计, 2-导出数据, 3-查看详情
  eventDescription: string;
  accessorUserId: string;
  accessorRole: number;
  clientIP: string;
  timestamp: number;
  dataScope: string; // 访问的数据范围
}
```

## 接口修改指南

### 添加新的统计维度

1. 确定数据来源（Log表、User表或新增表）
2. 设计计算逻辑和聚合方式
3. 考虑权限控制和数据脱敏
4. 实现接口处理函数
5. 更新proto定义和handler注册
6. 添加相应的缓存策略

### 扩展地理位置功能

1. 集成IP地理位置库
2. 实现IP地址脱敏机制
3. 添加地理位置数据缓存
4. 考虑GDPR等隐私法规合规性

### 性能优化指导

1. 监控慢查询并优化SQL
2. 实现分层缓存策略
3. 考虑数据预聚合
4. 优化大数据量分页查询

---

## 版本历史

- v1.0 (2024-01-15): 初始版本，实现基础令牌统计功能
- 后续版本更新请在此记录...