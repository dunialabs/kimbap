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
    timeRange: number;  // : 1-, 7-7, 30-30, 90-90
    toolIds?: string[]; // ID，
    page?: number;      // -
    pageSize?: number;  // -
  };
}

interface ToolMetrics {
  toolId: string;
  toolName: string;         //  (matches frontend)
  totalRequests: number;    // 
  successfulRequests: number;  //  (matches frontend)
  failedRequests: number;   // 
  averageResponseTime: number;  // (ms) (matches frontend)
  successRate: number;      // (%)
  lastUsed: string;         // () (matches frontend)
  status: "active" | "inactive" | "error";  //  (matches frontend)
  errorTypes: Array<{       //  (matches frontend)
    type: string;
    count: number;
  }>;
}

interface Response20002Data {
  tools: ToolMetrics[];  // Changed from toolMetrics to tools to match frontend expectation
  totalCount: number; // （）
}

/**
 * Protocol 20002 - Get Tool Detailed Metrics
 * （proxyKeyaction 1000-1099）
 */
export async function handleProtocol20002(body: Request20002): Promise<Response20002Data> {
  try {
    const { timeRange, toolIds, page = 1, pageSize = 50 } = body.params;
    const rawToken = body.common?.rawToken;
    const safePage = Math.max(1, Math.floor(Number(page) || 1));
    const safePageSize = Math.min(1000, Math.max(1, Math.floor(Number(pageSize) || 50)));
    
    // 1. proxyproxyKey（token）
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
    
    // 
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 2. proxy-apiserver
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
    
    // 3. where：proxyKey、action 1000-1099serverId
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
        notIn: ['', 'Unknown', 'unknown', 'null', 'undefined', '0'] // serverId
      }
    };

    const filteredToolIds = Array.isArray(toolIds)
      ? toolIds.filter((id): id is string => typeof id === 'string' && id.trim().length > 0)
      : [];

    if (filteredToolIds.length > 0) {
      logWhereCondition.serverId = {
        ...logWhereCondition.serverId,
        in: filteredToolIds
      };
    }
    
    const groupedServers = await prisma.log.groupBy({
      by: ['serverId'],
      where: logWhereCondition,
      _count: {
        id: true
      },
      orderBy: {
        _count: {
          id: 'desc'
        }
      }
    });

    const totalCount = groupedServers.length;
    
    // 
    const offset = (safePage - 1) * safePageSize;
    const pagedGroups = groupedServers.slice(offset, offset + safePageSize);
    const pagedServerIds = pagedGroups
      .map((group) => group.serverId)
      .filter((id): id is string => !!id);

    const pagedLogs = pagedServerIds.length === 0
      ? []
      : await prisma.log.findMany({
          where: {
            ...logWhereCondition,
            serverId: {
              in: pagedServerIds
            }
          },
          select: {
            id: true,
            serverId: true,
            error: true,
            statusCode: true,
            duration: true,
            addtime: true
          }
        });

    const logsByServerId = new Map<string, typeof pagedLogs>();
    pagedLogs.forEach((log) => {
      if (!log.serverId) return;
      const existing = logsByServerId.get(log.serverId) || [];
      existing.push(log);
      logsByServerId.set(log.serverId, existing);
    });
    
    // 6. 
    const toolMetrics: ToolMetrics[] = [];
    
    for (const group of pagedGroups) {
      const currentServerId = group.serverId;
      if (!currentServerId) continue;
      const server = serversMap[currentServerId];
      const toolName = server ? server.serverName : `${currentServerId} (Deleted)`;
      const logs = logsByServerId.get(currentServerId) || [];
      
      // logs
      const totalRequests = logs.length;
      const successfulRequests = logs.filter((log) => isSuccessfulRequestLog(log)).length;
      const failedRequests = totalRequests - successfulRequests;
      
      const validDurations = logs
        .filter((log) => isSuccessfulRequestLog(log) && log.duration && log.duration > 0)
        .map(log => log.duration!);
      const averageResponseTime = validDurations.length > 0 
        ? Math.round(validDurations.reduce((sum, d) => sum + d, 0) / validDurations.length)
        : 0;
      
      // 
      const addtimes = logs.map(log => Number(log.addtime));
      const lastUsedTimestamp = addtimes.length > 0 ? Math.max(...addtimes) : 0;
      const lastUsed = lastUsedTimestamp > 0 ? new Date(lastUsedTimestamp * 1000).toISOString() : new Date().toISOString();
      
      // 
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
      
      // 
      const successRate = totalRequests > 0 ? (successfulRequests / totalRequests) * 100 : 0;
      
      // 
      let status: "active" | "inactive" | "error" = "active";
      if (successRate < 70) {
        status = "error";
      } else if (totalRequests < 10) {
        status = "inactive";
      }
      
      toolMetrics.push({
        toolId: currentServerId,
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
