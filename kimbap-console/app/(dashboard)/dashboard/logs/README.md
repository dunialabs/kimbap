# Logs Management API Documentation

本文档说明日志管理页面 (`logs/page.tsx`) 中各个UI组件与后端API接口的映射关系，帮助前端开发人员了解数据获取方式。

## 页面结构与API映射

### 1. Log Filtering System (日志过滤系统)

**位置**: 页面顶部的过滤器卡片
**接口**: `GET /api/v1?cmdid=23001`

```typescript
// 请求参数
{
  page: 1,                    // 页码
  pageSize: 50,               // 每页大小
  timeRange: "24h",           // 时间范围: "1h", "6h", "24h", "7d", "all"
  level: "all",               // 日志级别: "all", "INFO", "WARN", "ERROR", "DEBUG"
  source: "all",              // 日志来源: "all", "api-gateway", "tool-manager", etc.
  search: "",                 // 搜索关键词
  requestId: "",              // 请求ID过滤
  userId: ""                  // 用户ID过滤
}

// 响应数据映射
logsData = data.logs.map(log => ({
  id: log.id,
  timestamp: log.timestamp,           // "2024-01-23 14:35:42.123"
  level: log.level,                   // "INFO", "WARN", "ERROR", "DEBUG"
  message: log.message,               // 日志消息
  source: log.source,                 // "api-gateway", "tool-manager"
  requestId: log.requestId,           // 请求ID（可选）
  userId: log.userId,                 // 用户ID（可选）
  rawData: log.rawData,               // 原始日志数据
  details: {
    method: log.details.method,       // HTTP方法
    url: log.details.url,             // 请求URL
    statusCode: log.details.statusCode, // 状态码
    responseTime: log.details.responseTime, // 响应时间
    userAgent: log.details.userAgent, // 用户代理
    ip: log.details.ip,               // 客户端IP
    tokenId: log.details.tokenId,     // 令牌ID（脱敏）
    toolName: log.details.toolName,   // 工具名称
    errorType: log.details.errorType, // 错误类型
    stackTrace: log.details.stackTrace // 堆栈跟踪
  }
}))

// 分页信息
totalCount = data.totalCount
totalPages = data.totalPages
availableSources = data.availableSources  // 用于动态生成来源过滤器选项
```

### 2. Log Statistics (日志统计信息)

**位置**: 可用于显示统计概览或图表
**接口**: `GET /api/v1?cmdid=23002`

```typescript
// 请求参数
{
  timeRange: "24h"  // 时间范围: "1h", "6h", "24h", "7d"
}

// 响应数据映射
statistics = data.statistics
logStats = {
  totalLogs: statistics.totalLogs,      // 总日志数
  errorLogs: statistics.errorLogs,      // 错误日志数
  warnLogs: statistics.warnLogs,        // 警告日志数
  infoLogs: statistics.infoLogs,        // 信息日志数
  debugLogs: statistics.debugLogs,      // 调试日志数
  errorRate: statistics.errorRate       // 错误率（百分比）
}

// 按来源统计（用于图表）
sourceStats = statistics.sourceStats.map(source => ({
  source: source.source,               // 来源名称
  logCount: source.logCount,           // 日志数量
  errorCount: source.errorCount,       // 错误数量
  percentage: source.percentage        // 占总日志的百分比
}))

// 按小时统计（用于趋势图）
hourlyStats = statistics.hourlyStats.map(hour => ({
  hour: hour.hour,                     // 小时标签，如 "14:00"
  totalCount: hour.totalCount,         // 该小时总日志数
  errorCount: hour.errorCount,         // 该小时错误数
  timestamp: hour.timestamp            // 时间戳
}))
```

### 3. Real-time Log Updates (实时日志更新)

**位置**: 页面实时更新功能
**接口**: `GET /api/v1?cmdid=23005`

```typescript
// 请求参数
{
  lastLogId: 12345,    // 最后接收到的日志ID（用于增量获取）
  level: "all",        // 日志级别过滤
  source: "all",       // 日志来源过滤
  limit: 50           // 返回数量限制
}

// 响应数据映射
realtimeUpdate = {
  newLogs: data.newLogs.map(log => ({
    // 与23001相同的LogEntry结构
    id: log.id,
    timestamp: log.timestamp,
    level: log.level,
    message: log.message,
    source: log.source,
    // ... 其他字段
  })),
  latestLogId: data.latestLogId,       // 最新日志ID
  hasMore: data.hasMore                // 是否还有更多日志
}
```

### 4. Log Export (日志导出)

**位置**: 页面顶部的下载按钮
**接口**: `GET /api/v1?cmdid=23004`

```typescript
// 请求参数
{
  timeRange: "24h",    // 时间范围
  level: "all",        // 日志级别过滤
  source: "all",       // 日志来源过滤
  search: "",          // 搜索关键词
  format: 1,           // 导出格式: 1-TXT, 2-JSON, 3-CSV
  maxRecords: 10000    // 最大记录数限制
}

// 响应数据映射
exportInfo = {
  downloadUrl: data.downloadUrl,       // 下载链接
  fileName: data.fileName,             // 文件名
  fileSize: data.fileSize,             // 文件大小（字节）
  recordCount: data.recordCount,       // 实际导出的记录数
  expiresAt: data.expiresAt           // 链接过期时间
}
```

## 实现建议

### 数据刷新策略

```typescript
// 1. 初始数据加载
const [logs, setLogs] = useState([]);
const [filters, setFilters] = useState({
  page: 1,
  pageSize: 50,
  timeRange: "24h",
  level: "all",
  source: "all",
  search: "",
  requestId: "",
  userId: ""
});

// 2. 实时更新机制
const [lastLogId, setLastLogId] = useState(0);

useEffect(() => {
  const fetchRealtimeLogs = async () => {
    try {
      const response = await fetchRealTimeLogs({
        lastLogId,
        level: filters.level,
        source: filters.source,
        limit: 50
      });
      
      if (response.common.code === 0) {
        const { newLogs, latestLogId } = response.data;
        
        if (newLogs.length > 0) {
          setLogs(prevLogs => [...newLogs, ...prevLogs]);
          setLastLogId(latestLogId);
        }
      }
    } catch (error) {
      console.error('Real-time update failed:', error);
    }
  };
  
  // 每10秒检查新日志
  const interval = setInterval(fetchRealtimeLogs, 10000);
  
  return () => clearInterval(interval);
}, [lastLogId, filters.level, filters.source]);

// 3. 过滤器变化时重新加载
useEffect(() => {
  const fetchLogs = async () => {
    try {
      const response = await fetchLogsWithFilters(filters);
      if (response.common.code === 0) {
        setLogs(response.data.logs);
        // 更新lastLogId以便实时更新
        if (response.data.logs.length > 0) {
          setLastLogId(Math.max(...response.data.logs.map(log => parseInt(log.id))));
        }
      }
    } catch (error) {
      console.error('Failed to fetch logs:', error);
    }
  };
  
  fetchLogs();
}, [filters]);
```

### 日志级别样式映射

```typescript
// 日志级别对应的UI样式
const getLevelIcon = (level: string) => {
  switch (level) {
    case "ERROR": return <XCircle className="h-4 w-4" />;
    case "WARN": return <AlertTriangle className="h-4 w-4" />;
    case "INFO": return <Info className="h-4 w-4" />;
    case "DEBUG": return <Activity className="h-4 w-4" />;
    default: return <Info className="h-4 w-4" />;
  }
};

const getLevelColor = (level: string) => {
  switch (level) {
    case "ERROR": return "destructive";
    case "WARN": return "secondary";
    case "INFO": return "default";
    case "DEBUG": return "outline";
    default: return "outline";
  }
};

const getLevelBgColor = (level: string) => {
  switch (level) {
    case "ERROR": return "bg-red-50 border-red-200";
    case "WARN": return "bg-yellow-50 border-yellow-200";
    case "INFO": return "bg-blue-50 border-blue-200";
    case "DEBUG": return "bg-gray-50 border-gray-200";
    default: return "bg-gray-50 border-gray-200";
  }
};
```

### 搜索和过滤实现

```typescript
// 防抖搜索
import { useDebouncedCallback } from 'use-debounce';

const debouncedSearch = useDebouncedCallback(
  (searchTerm: string) => {
    setFilters(prev => ({ 
      ...prev, 
      search: searchTerm, 
      page: 1  // 重置到第一页
    }));
  },
  500  // 500ms防抖
);

// 搜索输入处理
const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
  const value = e.target.value;
  setSearchTerm(value);
  debouncedSearch(value);
};

// 过滤器重置
const handleClearFilters = () => {
  setFilters({
    page: 1,
    pageSize: 50,
    timeRange: "24h",
    level: "all",
    source: "all",
    search: "",
    requestId: "",
    userId: ""
  });
  setSearchTerm("");
};
```

### 日志详情模态框

```typescript
// 日志详情状态
const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null);
const [showLogDetail, setShowLogDetail] = useState(false);

// 打开日志详情
const handleViewLogDetail = (log: LogEntry) => {
  setSelectedLog(log);
  setShowLogDetail(true);
};

// 复制原始日志数据
const handleCopyRawData = async (rawData: string) => {
  try {
    await navigator.clipboard.writeText(rawData);
    toast.success("Raw log data copied to clipboard");
  } catch (error) {
    console.error('Failed to copy to clipboard:', error);
    toast.error("Failed to copy log data");
  }
};
```

### 导出功能实现

```typescript
// 导出日志
const handleExportLogs = async (format: number) => {
  try {
    const response = await exportLogs({
      timeRange: filters.timeRange,
      level: filters.level,
      source: filters.source,
      search: filters.search,
      format,
      maxRecords: 10000
    });
    
    if (response.common.code === 0) {
      const { downloadUrl, fileName, recordCount } = response.data;
      
      // 创建下载链接
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = fileName;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      
      toast.success(`Successfully exported ${recordCount} log records`);
    }
  } catch (error) {
    console.error('Export failed:', error);
    toast.error("Failed to export logs");
  }
};

// 导出格式选择
const ExportDropdown = () => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="outline" size="sm">
        <Download className="mr-2 h-4 w-4" />
        Export Logs
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem onClick={() => handleExportLogs(1)}>
        Export as TXT
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => handleExportLogs(2)}>
        Export as JSON
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => handleExportLogs(3)}>
        Export as CSV
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
);
```

### 分页实现

```typescript
// 分页状态
const [pagination, setPagination] = useState({
  page: 1,
  pageSize: 50,
  totalCount: 0,
  totalPages: 0
});

// 分页处理
const handlePageChange = (newPage: number) => {
  setFilters(prev => ({ ...prev, page: newPage }));
  setPagination(prev => ({ ...prev, page: newPage }));
};

// 分页组件
const LogsPagination = () => (
  <div className="flex items-center justify-between px-2">
    <div className="text-sm text-muted-foreground">
      Showing {((pagination.page - 1) * pagination.pageSize) + 1} to{' '}
      {Math.min(pagination.page * pagination.pageSize, pagination.totalCount)} of{' '}
      {pagination.totalCount} results
    </div>
    
    <div className="flex items-center space-x-2">
      <Button
        variant="outline"
        size="sm"
        onClick={() => handlePageChange(pagination.page - 1)}
        disabled={pagination.page <= 1}
      >
        Previous
      </Button>
      
      <div className="text-sm">
        Page {pagination.page} of {pagination.totalPages}
      </div>
      
      <Button
        variant="outline"
        size="sm"
        onClick={() => handlePageChange(pagination.page + 1)}
        disabled={pagination.page >= pagination.totalPages}
      >
        Next
      </Button>
    </div>
  </div>
);
```

### TypeScript 接口定义

```typescript
// types/logs.types.ts
export interface LogEntry {
  id: string;
  timestamp: string;
  level: "INFO" | "WARN" | "ERROR" | "DEBUG";
  message: string;
  source: string;
  requestId?: string;
  userId?: string;
  rawData: string;
  details: LogDetails;
}

export interface LogDetails {
  method?: string;
  url?: string;
  statusCode?: number;
  responseTime?: number;
  userAgent?: string;
  ip?: string;
  tokenId?: string;
  toolName?: string;
  errorType?: string;
  stackTrace?: string;
}

export interface LogFilters {
  page: number;
  pageSize: number;
  timeRange: "1h" | "6h" | "24h" | "7d" | "all";
  level: "all" | "INFO" | "WARN" | "ERROR" | "DEBUG";
  source: string;
  search: string;
  requestId: string;
  userId: string;
}

export interface LogStatistics {
  totalLogs: number;
  errorLogs: number;
  warnLogs: number;
  infoLogs: number;
  debugLogs: number;
  errorRate: number;
  sourceStats: SourceStats[];
  hourlyStats: HourlyStats[];
}

export interface SourceStats {
  source: string;
  logCount: number;
  errorCount: number;
  percentage: number;
}

export interface HourlyStats {
  hour: string;
  totalCount: number;
  errorCount: number;
  timestamp: number;
}
```

### 错误处理

```typescript
// 统一错误处理
const handleApiError = (error: any, context: string) => {
  console.error(`${context} error:`, error);
  
  if (error.response?.status === 429) {
    toast.error("Too many requests. Please try again later.");
  } else if (error.response?.status === 403) {
    toast.error("Access denied. Please check your permissions.");
  } else {
    toast.error("An error occurred while loading logs. Please try again.");
  }
};

// 使用示例
try {
  const response = await fetchLogsWithFilters(filters);
  // 处理成功响应
} catch (error) {
  handleApiError(error, "Logs Loading");
}
```

## 注意事项

1. **实时更新**: 使用基于lastLogId的增量更新机制，避免重复获取已有日志
2. **性能优化**: 大量日志时考虑虚拟滚动，避免DOM性能问题
3. **搜索防抖**: 搜索功能使用防抖机制，减少API调用频率
4. **错误重试**: 网络错误时实现自动重试机制
5. **数据安全**: 敏感信息（如IP、Token）在前端显示时进行适当脱敏
6. **导出限制**: 导出功能应有合理的数量限制，避免性能问题