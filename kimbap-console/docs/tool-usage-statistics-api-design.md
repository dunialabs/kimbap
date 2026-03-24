# Tool Usage Statistics API Design Document

## 概览

本文档详细描述了工具使用统计接口（20001-20010）的数据计算逻辑和设计思路，用于指导未来的接口修改和优化。

## 数据基础

### 数据源表结构

所有统计接口基于 `Log` 表进行数据分析：

```sql
-- Log表关键字段
id                     INT     -- 主键
addtime                BIGINT  -- 添加时间(Unix时间戳)
action                 INT     -- 事件类型(MCPEventLogType枚举)
userid                 VARCHAR -- 用户ID
serverId              VARCHAR -- 工具/服务器ID
duration              INT     -- 响应时间(毫秒)
statusCode            INT     -- HTTP状态码
error                 TEXT    -- 错误信息
requestParams         TEXT    -- 请求参数
responseResult        TEXT    -- 响应结果
sessionId             VARCHAR -- 会话ID
```

### MCPEventLogType 枚举映射

接口主要关注以下事件类型：

```typescript
// 工具相关事件 (用于统计分析)
1001: RequestTool      // Gateway接收工具调用请求
1002: RequestResource  // Gateway接收资源读取请求  
1003: RequestPrompt    // Gateway接收提示获取请求
1004: ResponseTool     // Gateway返回工具调用结果
1005: ResponseResource // Gateway返回资源读取结果
1006: ResponsePrompt   // Gateway返回提示获取结果

// 反向请求 (可选统计)
1201-1206: Reverse*    // 服务器到客户端的反向请求
```

## 接口详细设计

### 20001 - 工具使用汇总统计

**目的**: 提供系统整体的工具使用概览数据

**核心计算逻辑**:

```typescript
// 1. 总工具数 - 历史上出现过的所有工具
totalTools = COUNT(DISTINCT serverId) FROM log WHERE serverId IS NOT NULL

// 2. 活跃工具数 - 时间范围内有请求的工具
activeTools = COUNT(DISTINCT serverId) FROM log 
              WHERE addtime >= startTime AND action IN (1001-1006)

// 3. 请求统计
totalRequests = COUNT(*) FROM log WHERE 时间范围 AND action IN (1001-1006)
successRequests = COUNT(*) FROM log WHERE 时间范围 AND statusCode BETWEEN 200-299
failedRequests = totalRequests - successRequests

// 4. 成功率
avgSuccessRate = (successRequests / totalRequests) * 100

// 5. 平均响应时间
avgResponseTime = AVG(duration) FROM log 
                  WHERE 时间范围 AND duration IS NOT NULL

// 6. 用户总数
totalUsers = COUNT(DISTINCT userid) FROM log WHERE 时间范围
```

**设计考量**:
- 总工具数基于历史数据，反映系统规模
- 活跃工具数基于时间范围，反映当前活跃度
- 成功率计算排除了NULL状态码的记录

### 20002 - 工具详细指标

**目的**: 提供每个工具的详细性能指标，支持分页

**核心计算逻辑**:

```typescript
// 1. 获取工具列表(支持分页)
tools = SELECT DISTINCT serverId FROM log WHERE 时间范围
pagedTools = tools.slice(offset, offset + pageSize)

// 2. 为每个工具计算指标
FOR EACH tool IN pagedTools:
  // 请求统计
  toolRequests = COUNT(*) WHERE serverId = tool AND 时间范围
  toolSuccess = COUNT(*) WHERE serverId = tool AND statusCode 200-299
  toolFailed = toolRequests - toolSuccess
  
  // 响应时间统计  
  avgResponseTime = AVG(duration) WHERE serverId = tool AND duration NOT NULL
  minResponseTime = MIN(duration) WHERE serverId = tool AND duration NOT NULL
  maxResponseTime = MAX(duration) WHERE serverId = tool AND duration NOT NULL
  
  // 最后使用时间
  lastUsed = MAX(addtime) WHERE serverId = tool
  
  // 状态判断
  status = 判断逻辑(最近活动时间, 错误率)
  
  // 独立用户数
  uniqueUsers = COUNT(DISTINCT userid) WHERE serverId = tool AND 时间范围
```

**状态判断逻辑**:

```typescript
// 工具状态判断
if (totalRequests === 0) {
  status = 2; // inactive
} else {
  const recentTime = now - 24小时;
  if (lastUsed > recentTime) {
    status = (failedRequests > successRequests) ? 3 : 1; // error : active
  } else {
    status = 2; // inactive
  }
}
```

### 20003 - 工具使用趋势

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
  
  FOR EACH tool:
    toolLogs = periodLogs WHERE serverId = tool
    requestCount = toolLogs.length
    successCount = COUNT(toolLogs WHERE statusCode 200-299)
    failedCount = requestCount - successCount
```

**设计考量**:
- 按周统计时，取每周的第一天作为起始点
- 支持指定工具ID列表，空则统计所有工具
- 数据结构便于前端绘制多线图表

### 20004 - 工具错误分析

**目的**: 分析工具错误类型和分布，帮助排查问题

**错误分类逻辑**:

```typescript
// 错误类型识别
function categorizeError(log) {
  let errorCode = 'UNKNOWN_ERROR';
  let errorMessage = '未知错误';
  
  // 1. 基于HTTP状态码
  if (log.statusCode) {
    errorCode = `HTTP_${log.statusCode}`;
    if (statusCode >= 400 && statusCode < 500) {
      errorMessage = '客户端错误';
    } else if (statusCode >= 500) {
      errorMessage = '服务器错误';
    }
  }
  
  // 2. 基于错误信息内容
  if (log.error && log.error.trim()) {
    const errorText = log.error.toLowerCase();
    
    // 错误模式匹配
    if (errorText.includes('timeout')) {
      errorCode = 'REQUEST_TIMEOUT';
      errorMessage = '请求超时';
    } else if (errorText.includes('connection')) {
      errorCode = 'CONNECTION_ERROR';
      errorMessage = '连接错误';
    } else if (errorText.includes('permission') || errorText.includes('denied')) {
      errorCode = 'PERMISSION_DENIED';
      errorMessage = '权限被拒绝';
    } else if (errorText.includes('not found')) {
      errorCode = 'NOT_FOUND';
      errorMessage = '资源未找到';
    } else if (errorText.includes('rate limit')) {
      errorCode = 'RATE_LIMIT_EXCEEDED';
      errorMessage = '请求频率超限';
    } else {
      errorCode = 'APPLICATION_ERROR';
      errorMessage = log.error.substring(0, 50);
    }
  }
  
  return { errorCode, errorMessage };
}

// 错误统计
FOR EACH tool WITH errors:
  errorTypes = GROUP BY (errorCode, errorMessage)
  FOR EACH errorType:
    count = COUNT(*)
    percentage = (count / tool.totalErrors) * 100
    lastOccurred = MAX(addtime)
```

**设计考量**:
- 优先识别具体的错误类型而非泛化的HTTP状态码
- 错误信息截取前50个字符避免过长
- 计算错误占比便于识别主要问题

### 20005 - 工具使用分布

**目的**: 提供饼图数据展示工具使用分布

**分布类型设计**:

```typescript
// 1. 按请求数分布 (metricType = 1)
distribution = GROUP BY serverId, COUNT(*) as requestCount
percentage = (requestCount / totalRequests) * 100

// 2. 按用户数分布 (metricType = 2)
FOR EACH tool:
  uniqueUsers = COUNT(DISTINCT userid) WHERE serverId = tool
  percentage = (uniqueUsers / totalUsers) * 100

// 3. 按响应时间分布 (metricType = 3)
FOR EACH tool:
  avgResponseTime = AVG(duration) WHERE serverId = tool AND duration NOT NULL
  // 相对百分比 (基于最大响应时间)
  percentage = (avgResponseTime / maxResponseTime) * 100
```

**设计考量**:
- 响应时间分布使用相对比例而非绝对占比
- 过滤掉无数据的工具避免图表混乱
- 按数值降序排列便于识别主要工具

### 20006 - 工具性能对比

**目的**: 提供柱状图数据展示工具间的性能对比

**对比指标设计**:

```typescript
// 1. 响应时间对比 (metricType = 1)
FOR EACH tool:
  avgValue = AVG(duration)
  minValue = MIN(duration)  
  maxValue = MAX(duration)
  // 按平均响应时间升序排列

// 2. 成功率对比 (metricType = 2)
FOR EACH tool:
  totalRequests = COUNT(*)
  successRequests = COUNT(*) WHERE statusCode 200-299
  avgValue = (successRequests / totalRequests) * 100
  minValue = 0
  maxValue = 100
  // 按成功率降序排列

// 3. 请求量对比 (metricType = 3)
FOR EACH tool:
  // 按天分组统计
  dailyStats = GROUP BY DATE(addtime), COUNT(*)
  avgValue = AVG(dailyStats.count)
  minValue = MIN(dailyStats.count)
  maxValue = MAX(dailyStats.count)
  // 按平均请求量降序排列
```

**设计考量**:
- 提供最小值、最大值、平均值便于理解数据分布
- 不同指标采用不同的排序策略
- 请求量对比基于日均值避免总量差异过大

### 20007 - 用户工具使用情况

**目的**: 从用户维度分析工具使用模式

**用户维度统计**:

```typescript
FOR EACH user:
  // 基础统计
  totalRequests = COUNT(*) WHERE userid = user AND 时间范围
  toolsUsed = COUNT(DISTINCT serverId) WHERE userid = user AND 时间范围
  lastActive = MAX(addtime) WHERE userid = user AND 时间范围
  
  // TOP5工具使用
  topTools = GROUP BY serverId, COUNT(*) as requestCount
             ORDER BY requestCount DESC
             LIMIT 5
  
  // 用户角色信息
  userInfo = SELECT role FROM user WHERE userid = user
```

**设计考量**:
- 支持按用户ID过滤查看特定用户
- TOP5工具便于识别用户的主要使用模式
- 分页支持处理大量用户数据

### 20008 - 工具实时状态

**目的**: 提供工具当前运行状态的实时监控

**状态判断逻辑**:

```typescript
// 时间节点定义
const now = currentTimestamp;
const fiveMinutesAgo = now - 300;  // 5分钟
const oneHourAgo = now - 3600;     // 1小时

FOR EACH tool:
  // 最近活动检查
  recentActivity = COUNT(*) WHERE serverId = tool AND addtime >= fiveMinutesAgo
  lastActivity = MAX(addtime) WHERE serverId = tool
  recentErrors = COUNT(*) WHERE serverId = tool AND addtime >= oneHourAgo AND 错误条件
  
  // 状态判断
  if (lastActivity > fiveMinutesAgo) {
    status = (recentErrors > 0) ? 3 : 0;  // error : online
  } else if (lastActivity > oneHourAgo) {
    status = 2;  // connecting (可能重连中)
  } else {
    status = 1;  // offline
  }
  
  // 活跃连接数 (基于会话ID)
  activeConnections = COUNT(DISTINCT sessionId) 
                      WHERE serverId = tool AND addtime >= fiveMinutesAgo
```

**状态定义**:
- 0: online - 最近5分钟有活动且无错误
- 1: offline - 超过1小时无活动
- 2: connecting - 1小时内有活动但不在最近5分钟
- 3: error - 最近5分钟有活动但存在错误

### 20009 - 使用报告导出

**目的**: 导出详细的使用报告支持离线分析

**导出数据结构**:

```typescript
// 导出记录格式
exportRecord = {
  timestamp: ISO8601时间格式,
  actionType: action数值,
  actionName: action名称映射,
  toolId: serverId,
  userId: userid,
  sessionId: sessionId,
  duration: duration || 0,
  statusCode: statusCode || 0,
  success: (statusCode >= 200 && statusCode < 300) ? 'Yes' : 'No',
  error: error || '',
  requestParams: 截取前500字符,
  responseResult: 截取前500字符
}

// 文件格式支持
formats = {
  1: 'CSV',   // 表格格式，便于Excel分析
  2: 'JSON',  // 结构化格式，便于程序处理
  3: 'PDF'    // 报告格式(当前简化为JSON)
}
```

**设计考量**:
- 导出文件包含元数据（导出时间、记录数量）
- 长文本字段截取避免文件过大
- 文件名包含时间戳和参数便于识别

### 20010 - 工具操作日志

**目的**: 提供详细的操作日志查询，支持审计和问题排查

**日志详情设计**:

```typescript
// 日志记录格式
actionLog = {
  logId: log.id,
  actionType: log.action,
  actionName: getActionName(log.action),  // 映射为可读名称
  toolId: log.serverId,
  toolName: `Tool ${log.serverId}`,
  userId: log.userid,
  userName: log.userid,  // 可扩展为从用户表获取真实姓名
  timestamp: Number(log.addtime),
  responseTime: log.duration || 0,
  status: 根据错误和状态码判断,
  errorMessage: 错误信息截取,
  details: JSON.stringify({
    sessionId: log.sessionId,
    statusCode: log.statusCode,
    requestParams: 截取前500字符,
    responseResult: 截取前500字符
  })
}

// 状态判断
status = 1;  // success (默认)
if (log.error && log.error.trim()) {
  status = 2;  // failed
} else if (log.statusCode && (log.statusCode < 200 || log.statusCode >= 300)) {
  status = 2;  // failed
}
```

**设计考量**:
- 支持按事件类型过滤（actionTypes参数）
- 详情信息以JSON格式存储便于前端解析
- 分页支持处理大量日志数据
- 按时间倒序排列便于查看最新操作

## 数据计算优化建议

### 索引优化

```sql
-- 推荐索引
CREATE INDEX idx_log_addtime ON log(addtime);
CREATE INDEX idx_log_serverid ON log(serverId);
CREATE INDEX idx_log_userid ON log(userid);
CREATE INDEX idx_log_action ON log(action);
CREATE INDEX idx_log_addtime_serverid ON log(addtime, serverId);
CREATE INDEX idx_log_addtime_action ON log(addtime, action);
```

### 查询优化

1. **时间范围过滤**: 所有查询都先按时间范围过滤，利用时间索引
2. **分页处理**: 大数据量接口采用OFFSET/LIMIT分页
3. **并行查询**: 使用Promise.all()并行执行独立查询
4. **数据缓存**: 可考虑对汇总数据进行短期缓存

### 扩展性考虑

1. **数据归档**: 超过一定时间的日志可归档到历史表
2. **分表策略**: 按月分表存储日志数据
3. **实时计算**: 高频访问的统计数据可预计算存储
4. **监控告警**: 对异常的统计结果进行监控告警

## 接口修改指南

### 添加新的统计维度

1. 确定数据来源（Log表或其他表）
2. 设计计算逻辑和聚合方式
3. 考虑时间范围和分页需求
4. 实现接口处理函数
5. 更新proto定义和handler注册

### 修改现有计算逻辑

1. 评估对前端展示的影响
2. 确保数据向后兼容性
3. 更新相关文档和测试用例
4. 考虑数据库查询性能影响

### 性能优化指导

1. 监控慢查询并优化SQL
2. 考虑数据预聚合和缓存
3. 合理使用数据库索引
4. 避免全表扫描操作

---

## 版本历史

- v1.0 (2024-01-15): 初始版本，实现基础统计功能
- 后续版本更新请在此记录...