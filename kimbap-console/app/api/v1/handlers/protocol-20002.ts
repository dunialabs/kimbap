import { ApiError, ErrorCode } from '@/lib/error-codes';
import { isSuccessfulRequestLog } from '@/lib/log-utils';
import { getProxy, getServers } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';

interface Request20002 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;  // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    toolIds?: string[]; // 特定工具ID列表，空表示所有工具
    page?: number;      // 分页-页码
    pageSize?: number;  // 分页-每页数量
  };
}

interface ToolMetrics {
  toolId: string;
  toolName: string;         // 工具名称 (matches frontend)
  totalRequests: number;    // 总请求数
  successfulRequests: number;  // 成功请求数 (matches frontend)
  failedRequests: number;   // 失败请求数
  averageResponseTime: number;  // 平均响应时间(ms) (matches frontend)
  successRate: number;      // 成功率(%)
  lastUsed: string;         // 最后使用时间(字符串) (matches frontend)
  status: "active" | "inactive" | "error";  // 工具状态 (matches frontend)
  errorTypes: Array<{       // 错误类型统计 (matches frontend)
    type: string;
    count: number;
  }>;
}

interface Response20002Data {
  tools: ToolMetrics[];  // Changed from toolMetrics to tools to match frontend expectation
  totalCount: number; // 总数量（用于分页）
}

/**
 * Protocol 20002 - Get Tool Detailed Metrics
 * 获取各工具详细指标数据（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol20002(body: Request20002): Promise<Response20002Data> {
  try {
    const { timeRange, toolIds, page = 1, pageSize = 50 } = body.params;
    const rawToken = body.common?.rawToken;
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-20002] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-20002] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 2. 从proxy-api获取有效的server列表
    let serversMap: { [serverId: string]: any } = {};
    let validServerIds = new Set<string>();
    try {
      const serversResult = await getServers({ enabled: true }, body.common.userid, rawToken);
      const serversList = serversResult.servers || [];
      serversList.forEach((server: any) => {
        if (server.serverId) {
          serversMap[server.serverId] = server;
          validServerIds.add(server.serverId);
        }
      });
      console.log('[Protocol-20002] Got servers from proxy-api:', serversList.length);
      console.log('[Protocol-20002] Valid server IDs count:', validServerIds.size);
    } catch (error) {
      console.warn('[Protocol-20002] Failed to get servers from proxy-api:', error);
    }
    
    // 3. 构建where条件：基于proxyKey、action 1000-1099和非空serverId
    const logWhereCondition: any = {
      proxyKey: proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      action: {
        gte: 1000,
        lte: 1099
      },
      serverId: {
        not: '',
        notIn: ['Unknown', 'unknown', 'null', 'undefined', '0'] // 排除明显无效的serverId
      }
    };
    
    // 4. 获取所有日志数据并分类工具
    const allLogs = await prisma.log.findMany({
      where: logWhereCondition,
      select: {
        id: true,
        serverId: true,
        error: true,
        statusCode: true,
        duration: true,
        addtime: true
      }
    });
    
    console.log('[Protocol-20002] Found', allLogs.length, 'total logs');
    
    // 5. 按工具分类统计（现在所有log都有有效的serverId）
    const INVALID_IDS = new Set(['Unknown', 'unknown', 'null', 'undefined', '0']);
    const toolGroups: { [serverId: string]: { toolName: string; logs: any[] } } = {};

    allLogs.forEach(log => {
      const serverId = log.serverId!;
      if (!serverId || INVALID_IDS.has(serverId)) return;

      const server = serversMap[serverId];
      const toolName = server ? server.serverName : `${serverId} (Deleted)`;

      if (!toolGroups[serverId]) {
        toolGroups[serverId] = { toolName, logs: [] };
      }
      toolGroups[serverId].logs.push(log);
    });
    
    // 按请求数量排序
    const toolGroupsArray = Object.entries(toolGroups)
      .map(([serverId, group]) => ({
        toolName: group.toolName,
        serverId,
        requestCount: group.logs.length,
        logs: group.logs,
      }))
      .sort((a, b) => b.requestCount - a.requestCount);
    
    console.log('[Protocol-20002] Grouped into', toolGroupsArray.length, 'tools:', 
      toolGroupsArray.map(g => `${g.toolName}(${g.requestCount})`).join(', '));
    
    // 过滤特定工具（如果指定）
    let filteredGroups = toolGroupsArray;
    if (toolIds && toolIds.length > 0) {
      filteredGroups = toolGroupsArray.filter(group => 
        group.serverId && toolIds.includes(group.serverId)
      );
    }
    
    const totalCount = filteredGroups.length;
    
    // 分页处理
    const offset = (page - 1) * pageSize;
    const pagedGroups = filteredGroups.slice(offset, offset + pageSize);
    
    // 6. 为每个工具计算详细指标
    const toolMetrics: ToolMetrics[] = [];
    
    for (const group of pagedGroups) {
      const { toolName, logs } = group;
      
      // 基于内存中的logs计算指标
      const totalRequests = logs.length;
      const successfulRequests = logs.filter((log) => isSuccessfulRequestLog(log)).length;
      const failedRequests = totalRequests - successfulRequests;
      
      const validDurations = logs
        .filter((log) => isSuccessfulRequestLog(log) && log.duration && log.duration > 0)
        .map(log => log.duration!);
      const averageResponseTime = validDurations.length > 0 
        ? Math.round(validDurations.reduce((sum, d) => sum + d, 0) / validDurations.length)
        : 0;
      
      // 最后使用时间
      const addtimes = logs.map(log => Number(log.addtime));
      const lastUsedTimestamp = addtimes.length > 0 ? Math.max(...addtimes) : 0;
      const lastUsed = lastUsedTimestamp > 0 ? new Date(lastUsedTimestamp * 1000).toISOString() : new Date().toISOString();
      
      // 错误统计
      const errorGroups: { [error: string]: number } = {};
      logs.forEach(log => {
        if (log.error && log.error !== '') {
          errorGroups[log.error] = (errorGroups[log.error] || 0) + 1;
        }
      });
      
      const errorTypes = Object.entries(errorGroups)
        .sort(([,a], [,b]) => b - a)
        .slice(0, 5)
        .map(([error, count]) => ({ type: error, count }));
      
      // 计算成功率
      const successRate = totalRequests > 0 ? (successfulRequests / totalRequests) * 100 : 0;
      
      // 工具状态
      let status: "active" | "inactive" | "error" = "active";
      if (successRate < 70) {
        status = "error";
      } else if (totalRequests < 10) {
        status = "inactive";
      }
      
      toolMetrics.push({
        toolId: group.serverId,
        toolName,
        totalRequests,
        successfulRequests,
        failedRequests,
        averageResponseTime,
        successRate: Math.round(successRate * 10) / 10,
        lastUsed,
        status,
        errorTypes
      });
    }
    
    const response: Response20002Data = {
      tools: toolMetrics,
      totalCount
    };
    
    console.log(`[Protocol-20002] Returning ${toolMetrics.length} tools based on log statistics (action 1000-1099, proxyKey: ${proxyKey.substring(0, 8)}...)`);
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20002 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool detailed metrics' });
  }
}
