import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers, getServersStatus } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { Prisma } from '@prisma/client';

interface Request20001 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number; // : 1-, 7-7, 30-30, 90-90
  };
}

interface Response20001Data {
  totalTools: number;        // 
  activeTools: number;       // （）
  totalRequests: number;     // 
  successRequests: number;   // 
  failedRequests: number;    // 
  avgSuccessRate: number;    // (%)
  avgResponseTime: number;   // (ms)
  totalUsers: number;        // 
}

/**
 * Protocol 20001 - Get Tool Usage Summary
 * （proxyKeyaction 1000-1099）
 */
export async function handleProtocol20001(body: Request20001): Promise<Response20001Data> {
  try {
    const rawToken = body.common?.rawToken;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;
    
    // 1. proxyproxyKey（token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-20001] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-20001] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
    // 2. proxy-apiserver
    let totalToolsCount = 0;
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
      totalToolsCount = validServerIds.size;
      
      console.log('[Protocol-20001] Total tools from proxy-api:', totalToolsCount);
      console.log('[Protocol-20001] Valid server IDs count:', validServerIds.size);
    } catch (error) {
      console.warn('[Protocol-20001] Failed to get servers from proxy-api:', error);
    }

    const validServerIdList = Array.from(validServerIds);

    // 3. where：proxyKey、action 1000-1099serverId
    const whereCondition: any = {
      proxyKey: proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      action: {
        gte: 1000,
        lte: 1099
      },
      serverId: {
        in: validServerIdList,
        not: ''
      }
    };

    // 4. （action 1000-1099）
    const [
      totalRequestsCount,
      uniqueUsers
    ] = await Promise.all([
      
      // 
      prisma.log.count({
        where: whereCondition
      }),
      
      // 
      prisma.log.findMany({
        where: whereCondition,
        select: {
          userid: true
        },
        distinct: ['userid']
      })
    ]);

    let successRequestsCount = 0;
    let avgDuration = 0;

    if (validServerIdList.length > 0) {
      const [successCountRows, avgDurationRows] = await Promise.all([
        prisma.$queryRaw<Array<{ count: bigint | number }>>`
          SELECT COUNT(*) AS count
          FROM log
          WHERE proxy_key = ${proxyKey}
            AND addtime >= ${BigInt(startTime)}
            AND action BETWEEN 1000 AND 1099
            AND server_id <> ''
            AND server_id IN (${Prisma.join(validServerIdList)})
            AND TRIM(COALESCE(error, '')) = ''
            AND (status_code IS NULL OR status_code < 400)
        `,
        prisma.$queryRaw<Array<{ avg_duration: number | null }>>`
          SELECT AVG(duration) AS avg_duration
          FROM log
          WHERE proxy_key = ${proxyKey}
            AND addtime >= ${BigInt(startTime)}
            AND action BETWEEN 1000 AND 1099
            AND server_id <> ''
            AND server_id IN (${Prisma.join(validServerIdList)})
            AND TRIM(COALESCE(error, '')) = ''
            AND (status_code IS NULL OR status_code < 400)
            AND duration > 0
        `,
      ]);

      const successCountValue = successCountRows[0]?.count;
      successRequestsCount = typeof successCountValue === 'bigint' ? Number(successCountValue) : Number(successCountValue || 0);
      avgDuration = Number(avgDurationRows[0]?.avg_duration || 0);
    }
    
    // ：proxy-api 3004
    let activeToolsCount = 0;
    try {
      const serversStatus = await getServersStatus(body.common.userid, rawToken);
      // Online(0)
      const onlineServerIds = Object.keys(serversStatus).filter(serverId => 
        serversStatus[serverId] === 0 // ServerStatus.Online = 0
      );
      
      // （）
      const uniqueActiveToolNames = new Set();
      onlineServerIds.forEach(serverId => {
        if (serversMap[serverId]) {
          // ，serverName
          uniqueActiveToolNames.add(serversMap[serverId].serverName);
        } else {
          // ，Unknown（，）
          uniqueActiveToolNames.add('Unknown');
        }
      });
      
      activeToolsCount = uniqueActiveToolNames.size;
      console.log('[Protocol-20001] Active tools from server status:', activeToolsCount, 'online servers:', onlineServerIds.length);
    } catch (error) {
      console.error('[Protocol-20001] Failed to get servers status:', error);
      // ：0
      activeToolsCount = 0;
      console.warn('[Protocol-20001] Using fallback: activeTools = 0 due to server status query failure');
    }
    
    // 
    const totalRequests = totalRequestsCount;
    const successRequests = successRequestsCount;
    const failedRequests = totalRequests - successRequests;
    
    // 
    const avgSuccessRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
    
    // （）
    const avgResponseTime = Math.round(avgDuration);
    
    // 
    const totalUsers = uniqueUsers.filter(u => u.userid).length;
    
    const response: Response20001Data = {
      totalTools: totalToolsCount,
      activeTools: activeToolsCount,
      totalRequests,
      successRequests,
      failedRequests,
      avgSuccessRate: Math.round(avgSuccessRate * 10) / 10, // 1
      avgResponseTime,
      totalUsers
    };
    
    console.log('[Protocol-20001] Response:', {
      totalTools: response.totalTools,
      activeTools: response.activeTools,
      totalRequests: response.totalRequests,
      avgSuccessRate: response.avgSuccessRate,
      avgResponseTime: response.avgResponseTime,
      proxyKey: proxyKey
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20001 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool usage summary statistics' });
  }
}
