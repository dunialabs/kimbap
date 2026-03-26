import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers } from '@/lib/proxy-api';
import { TOOL_USAGE_ACTION_RANGE } from '@/lib/log-utils';



interface Request22002 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number; // : 1-, 7-7, 30-30
    limit: number;     // ，10
  };
}

interface ToolUsageStat {
  toolName: string;      // 
  toolType: string;      // 
  requestCount: number;  // 
  percentage: number;    // 
  color: string;         // （UI）
}

interface Response22002Data {
  tools: ToolUsageStat[];  // Changed from topTools to tools to match frontend expectation
  totalRequests: number; // （）
}

/**
 * Protocol 22002 - Get Top Tools by Usage
 * （proxyKeyaction 1000-1099）
 */
export async function handleProtocol22002(body: Request22002): Promise<Response22002Data> {
  try {
    const rawToken = body.common?.rawToken;
    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : 1;
    const parsedLimit = Number(body.params?.limit);
    const normalizedLimit = Math.floor(parsedLimit);
    const limit = Number.isFinite(normalizedLimit) && normalizedLimit >= 1 ? normalizedLimit : 10;
    
    // ，UI
    const colors = [
      '#3b82f6', // blue-500
      '#10b981', // emerald-500  
      '#8b5cf6', // violet-500
      '#f59e0b', // amber-500
      '#ef4444', // red-500
      '#06b6d4', // cyan-500
      '#84cc16', // lime-500
      '#f97316', // orange-500
      '#ec4899', // pink-500
      '#6366f1'  // indigo-500
    ];
    
    // 1. proxyproxyKey（token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-22002] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-22002] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 2. server
    let serversMap: Record<string, any> = {};
    try {
      const serversResult = await getServers({}, body.common.userid, rawToken);
      const serversList = serversResult.servers || [];
      serversList.forEach((server: any) => {
        if (server.serverId) serversMap[server.serverId] = server;
      });
      console.log('[Protocol-22002] Got valid servers count:', Object.keys(serversMap).length);
    } catch (error) {
      console.warn('[Protocol-22002] Failed to get servers from proxy-api:', error);
    }
    
    // 3. proxyKey、action 1000-1099serverId
    const logWhereCondition = {
      proxyKey: proxyKey,
      action: {
        gte: TOOL_USAGE_ACTION_RANGE.gte,
        lte: TOOL_USAGE_ACTION_RANGE.lte
      },
      addtime: { gte: BigInt(startTime) },
      serverId: {
        not: '',
        notIn: ['Unknown', 'unknown', 'null', 'undefined', '0'] // serverId
      }
    };
    
    let topTools: ToolUsageStat[] = [];
    let totalRequestsCount = 0;
    
    try {
      totalRequestsCount = await prisma.log.count({ where: logWhereCondition });

      const groupedLogs = await prisma.log.groupBy({
        by: ['serverId'],
        where: logWhereCondition,
        _count: { serverId: true },
        orderBy: { _count: { serverId: 'desc' } },
        take: limit
      });

      console.log('[Protocol-22002] Found', totalRequestsCount, 'total logs');

      if (groupedLogs.length > 0) {
        topTools = groupedLogs.map(({ serverId, _count }, index) => {
          const foundServer = serverId ? serversMap[serverId] : null;
          const toolName = foundServer ? foundServer.serverName : `${serverId} (Unavailable)`;
          const requestCount = _count.serverId;
          const percentage = totalRequestsCount > 0 ? (requestCount / totalRequestsCount) * 100 : 0;
          
          return {
            toolName,
            toolType: foundServer ? (foundServer.type || 'Custom') : 'Unknown',
            requestCount,
            percentage: Math.round(percentage * 10) / 10,
            color: colors[index % colors.length]
          };
        });
      } else {
        console.log('[Protocol-22002] No tool usage found in logs for action 1000-1099 and proxyKey:', proxyKey);
      }

      console.log('[Protocol-22002] Top tools with new grouping rules:', topTools.map(t => `${t.toolName}: ${t.requestCount}`));
    } catch (error) {
      console.error('[Protocol-22002] Error querying tool usage stats:', error);
    }

    const response: Response22002Data = {
      tools: topTools,
      totalRequests: totalRequestsCount
    };
    
    console.log('[Protocol-22002] Response:', {
      topToolsCount: topTools.length,
      totalRequests: totalRequestsCount,
      topTool: topTools.length > 0 ? topTools[0].toolName : 'None',
      timeRange,
      toolNames: topTools.map(t => t.toolName),
      proxyKey: proxyKey
    });
    
    return response;
    
  } catch (error) {
    console.error('[Protocol-22002] error:', error);
    if (error instanceof ApiError) throw error;
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
      details: 'Failed to get top usage tools' 
    });
  }
}
