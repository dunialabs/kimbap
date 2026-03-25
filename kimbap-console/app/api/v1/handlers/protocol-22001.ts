import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers, getUsers } from '@/lib/proxy-api';
import { TOOL_USAGE_ACTION_RANGE } from '@/lib/log-utils';



interface Request22001 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange?: number;
  };
}

interface Response22001Data {
  totalRequests24h: number;           // 24小时总请求数
  requestsChangePercent: number;      // 相比昨天的增长率百分比
  activeTokens: number;               // 活跃令牌数
  tokensUsedLastHour: number;         // 最近1小时使用的令牌数
  toolsInUse: number;                 // 使用中的工具数
  mostActiveToolName: string;         // 最活跃的工具名称
  avgResponseTime: number;            // 平均响应时间(毫秒)
  responseTimeChange: number;         // 相比昨天的响应时间变化(毫秒)
}

/**
 * Protocol 22001 - Get Usage Overview Summary
 * 获取使用概览汇总统计（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol22001(body: Request22001): Promise<Response22001Data> {
  try {
    const rawToken = body.common?.rawToken;
    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : 1;
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-22001] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-22001] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const oneHourSeconds = 60 * 60;
    const rangeSeconds = timeRange * 24 * 60 * 60;

    const currentRangeStart = now - rangeSeconds;
    const previousRangeStart = now - (rangeSeconds * 2);
    const oneHourAgo = now - oneHourSeconds;
    
    const logWhereCondition = {
      proxyKey: proxyKey,
      action: {
        gte: TOOL_USAGE_ACTION_RANGE.gte,
        lte: TOOL_USAGE_ACTION_RANGE.lte
      },
      serverId: {
        notIn: ['', 'Unknown', 'unknown', 'null', 'undefined', '0']
      }
    };
    
    // 查询24小时内的总请求数
    const totalRequests24hCount = await prisma.log.count({
      where: {
        ...logWhereCondition,
        addtime: {
          gte: BigInt(currentRangeStart)
        }
      }
    });
    
    // 查询昨天的总请求数（用于计算增长率）
    const totalRequestsYesterdayCount = await prisma.log.count({
      where: {
        ...logWhereCondition,
        addtime: {
          gte: BigInt(previousRangeStart),
          lt: BigInt(currentRangeStart)
        }
      }
    });
    
    // 2. 从proxy-api获取用户列表（过滤删除的用户）
    let validUserIds: Set<string> | null = null;
    try {
      const usersResult = await getUsers({}, body.common.userid, rawToken);
      const usersList = usersResult.users || [];
      validUserIds = new Set<string>();
      usersList.forEach((user: any) => {
        if (user.userId) {
          validUserIds!.add(user.userId);
        }
      });
      console.log('[Protocol-22001] Got valid users count:', validUserIds.size);
    } catch (error) {
      console.warn('[Protocol-22001] Failed to get users from proxy-api:', error);
    }
    
    const validUserIdList = validUserIds && validUserIds.size > 0 ? Array.from(validUserIds) : null;
    const useridFilter = validUserIdList ? { in: validUserIdList, not: '' } : { not: '' };

    const activeTokensResult = await prisma.log.findMany({
      where: {
        ...logWhereCondition,
        addtime: { gte: BigInt(currentRangeStart) },
        userid: useridFilter,
      },
      select: { userid: true },
      distinct: ['userid'],
    });
    
    const activeTokensCount = activeTokensResult.length;
    
    const tokensUsedLastHourResult = await prisma.log.findMany({
      where: {
        ...logWhereCondition,
        addtime: { gte: BigInt(oneHourAgo) },
        userid: useridFilter,
      },
      select: { userid: true },
      distinct: ['userid'],
    });
    
    const tokensUsedLastHourCount = tokensUsedLastHourResult.length;
    
    // 3. 获取Top Tools（最近24小时）并计算toolsInUse
    let toolsInUseCount = 0;
    let mostActiveToolName = 'No Active Tools';
    
    try {
      // 获取24小时内的所有日志（基于action 1000-1099）
      const toolLogs = await prisma.log.findMany({
        where: {
          ...logWhereCondition,
          addtime: { gte: BigInt(currentRangeStart) }
        },
        select: {
          id: true,
          serverId: true
        }
      });
      
      console.log('[Protocol-22001] Found', toolLogs.length, 'logs in last 24h');
      
      // 获取工具列表用于名称映射
      let serversMap: { [serverId: string]: any } = {};
      try {
        const serversResult = await getServers({}, body.common.userid, rawToken);
        const serversList = serversResult.servers || [];
        serversList.forEach((server: any) => {
          serversMap[server.serverId] = server;
        });
      } catch (error) {
        console.warn('[Protocol-22001] Failed to get servers from proxy-api:', error);
      }
      
      // 按工具分类统计（使用新的分组规则）
      const toolCounts: { [toolName: string]: number } = {};
      
      toolLogs.forEach(log => {
        if (!log.serverId) return;
        const server = serversMap[log.serverId];
        if (!server || !server.serverName) return;
        const toolName = server.serverName;
        toolCounts[toolName] = (toolCounts[toolName] || 0) + 1;
      });
      
      // 统计工具数量和获取最活跃工具
      toolsInUseCount = Object.keys(toolCounts).length;
      
      // 获取最活跃的工具名称
      const sortedTools = Object.entries(toolCounts).sort(([,a], [,b]) => b - a);
      if (sortedTools.length > 0) {
        mostActiveToolName = sortedTools[0][0];
      }
      
      console.log('[Protocol-22001] Tools in use:', toolsInUseCount, 'Most active:', mostActiveToolName);
    } catch (error) {
      console.error('[Protocol-22001] Error getting tool usage stats:', error);
    }
    
    // 4. 计算平均响应时间（action 1000-1099，error为空，duration>0）
    let avgResponseTime = 0;
    let avgResponseTimeYesterday = 0;
    
    try {
      // 今天24小时内的平均响应时间
      const responseTimeToday = await prisma.log.aggregate({
        where: {
          ...logWhereCondition,
          addtime: { gte: BigInt(currentRangeStart) },
          error: '',
          duration: { gt: 0 }
        },
        _avg: { duration: true }
      });
      
      avgResponseTime = responseTimeToday._avg.duration ? Math.round(responseTimeToday._avg.duration) : 0;
      
      // 昨天24小时的平均响应时间
      const responseTimeYesterday = await prisma.log.aggregate({
        where: {
          ...logWhereCondition,
          addtime: { 
            gte: BigInt(previousRangeStart),
            lt: BigInt(currentRangeStart)
          },
          error: '',
          duration: { gt: 0 }
        },
        _avg: { duration: true }
      });
      
      avgResponseTimeYesterday = responseTimeYesterday._avg.duration ? Math.round(responseTimeYesterday._avg.duration) : 0;
    } catch (error) {
      console.error('[Protocol-22001] Error calculating response time:', error);
    }
    
    // 计算增长率
    const requestsChangePercent = totalRequestsYesterdayCount > 0 
      ? ((totalRequests24hCount - totalRequestsYesterdayCount) / totalRequestsYesterdayCount) * 100 
      : 0;
    
    const responseTimeChange = avgResponseTime - avgResponseTimeYesterday;
    
    const response: Response22001Data = {
      totalRequests24h: totalRequests24hCount,
      requestsChangePercent: Math.round(requestsChangePercent * 10) / 10,
      activeTokens: activeTokensCount,
      tokensUsedLastHour: tokensUsedLastHourCount,
      toolsInUse: toolsInUseCount,
      mostActiveToolName,
      avgResponseTime,
      responseTimeChange
    };
    
    console.log('[Protocol-22001] Response:', {
      totalRequests24h: response.totalRequests24h,
      activeTokens: response.activeTokens,
      toolsInUse: response.toolsInUse,
      mostActiveToolName: response.mostActiveToolName,
      avgResponseTime: response.avgResponseTime,
      proxyKey: proxyKey
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 22001 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
      details: 'Failed to get usage overview summary statistics' 
    });
  }
}
