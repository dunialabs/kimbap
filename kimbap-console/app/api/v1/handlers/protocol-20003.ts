import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';

interface Request20003 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;      // 时间范围: 7-最近7天, 30-最近30天, 90-最近90天
    toolIds?: string[];     // 要查看趋势的工具ID列表，空表示所有工具
    granularity?: number;   // 数据粒度: 1-按小时, 2-按天, 3-按周
  };
}

interface Response20003Data {
  trends: Array<{
    date: string;
    [toolName: string]: string | number;
  }>;
}

/**
 * Protocol 20003 - Get Tool Usage Trends
 * 获取工具使用趋势数据（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol20003(body: Request20003): Promise<Response20003Data> {
  try {
    const { timeRange, toolIds, granularity = 2 } = body.params;
    const rawToken = body.common?.rawToken;
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-20003] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-20003] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
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
      console.log('[Protocol-20003] Got servers from proxy-api:', serversList.length);
      console.log('[Protocol-20003] Valid server IDs count:', validServerIds.size);
    } catch (error) {
      console.warn('[Protocol-20003] Failed to get servers from proxy-api:', error);
    }
    
    const overallStartTime = Math.floor(Date.now() / 1000) - (timeRange * 24 * 60 * 60);
    const INVALID_SERVER_IDS = new Set(['', 'Unknown', 'unknown', 'null', 'undefined', '0']);

    const serverIdFilter: any = { not: '', notIn: ['Unknown', 'unknown', 'null', 'undefined', '0'] };
    if (toolIds && toolIds.length > 0) {
      serverIdFilter.in = toolIds;
    }

    const allLogs = await prisma.log.findMany({
      where: {
        proxyKey,
        action: { gte: 1000, lte: 1099 },
        addtime: { gte: BigInt(overallStartTime) },
        serverId: serverIdFilter,
      },
      select: { addtime: true, serverId: true },
    });

    const timePoints = generateTimePoints(timeRange, granularity);

    const bucketMap = new Map<string, Record<string, number>>();
    for (const tp of timePoints) {
      bucketMap.set(tp, {});
    }

    for (const log of allLogs) {
      const serverId = log.serverId;
      if (!serverId || INVALID_SERVER_IDS.has(serverId)) continue;

      const tp = timePoints.find(p => {
        const { startTime: s, endTime: e } = getTimeRangeForPoint(p, granularity);
        return Number(log.addtime) >= s && Number(log.addtime) < e;
      });
      if (!tp) continue;

      const toolName = serversMap[serverId]?.serverName ?? `${serverId} (Deleted)`;
      const bucket = bucketMap.get(tp)!;
      bucket[toolName] = (bucket[toolName] ?? 0) + 1;
    }

    const trends: Array<{date: string; [toolName: string]: string | number}> = timePoints.map(tp => ({
      date: tp,
      ...bucketMap.get(tp),
    }));
    
    const response: Response20003Data = {
      trends
    };
    
    console.log(`[Protocol-20003] Response:`, {
      trendsCount: trends.length,
      timePointsCount: timePoints.length,
      sampleTrendData: trends.length > 0 ? trends[0] : null,
      validServerIdsCount: validServerIds.size,
      proxyKey: proxyKey.substring(0, 8) + '...'
    });
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20003 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool usage trends' });
  }
}

/**
 * 根据时间点和粒度获取时间范围
 */
function getTimeRangeForPoint(timePoint: string, granularity: number): { startTime: number, endTime: number } {
  const date = new Date(timePoint);
  
  let startTime: number;
  let endTime: number;
  
  switch (granularity) {
    case 1: { // 按小时
      startTime = Math.floor(date.getTime() / 1000);
      endTime = startTime + 60 * 60; // +1小时
      break;
    }
    case 3: { // 按周
      const startOfWeek = new Date(date);
      startOfWeek.setDate(date.getDate() - date.getDay());
      startOfWeek.setHours(0, 0, 0, 0);
      startTime = Math.floor(startOfWeek.getTime() / 1000);
      endTime = startTime + 7 * 24 * 60 * 60; // +7天
      break;
    }
    case 2: // 按天（默认）
    default: {
      const startOfDay = new Date(date);
      startOfDay.setHours(0, 0, 0, 0);
      startTime = Math.floor(startOfDay.getTime() / 1000);
      endTime = startTime + 24 * 60 * 60; // +1天
      break;
    }
  }
  
  return { startTime, endTime };
}

/**
 * 生成时间点列表
 */
function generateTimePoints(timeRange: number, granularity: number): string[] {
  const now = new Date();
  const points: string[] = [];
  
  let intervalMs: number;
  let dateFormat: (date: Date) => string;
  
  switch (granularity) {
    case 1: // 按小时
      intervalMs = 60 * 60 * 1000;
      dateFormat = (date: Date) => 
        `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`;
      break;
    case 3: // 按周
      intervalMs = 7 * 24 * 60 * 60 * 1000;
      dateFormat = (date: Date) => {
        const startOfWeek = new Date(date);
        startOfWeek.setDate(date.getDate() - date.getDay());
        return `${startOfWeek.getFullYear()}-${String(startOfWeek.getMonth() + 1).padStart(2, '0')}-${String(startOfWeek.getDate()).padStart(2, '0')}`;
      };
      break;
    case 2: // 按天（默认）
    default:
      intervalMs = 24 * 60 * 60 * 1000;
      dateFormat = (date: Date) => 
        `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
      break;
  }
  
  const startTime = now.getTime() - (timeRange * 24 * 60 * 60 * 1000);
  
  for (let time = startTime; time <= now.getTime(); time += intervalMs) {
    points.push(dateFormat(new Date(time)));
  }
  
  return points;
}
