import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers, getServersStatus } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { Prisma } from '@prisma/client';

interface Request20001 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
  };
}

interface Response20001Data {
  totalTools: number;        // 工具总数
  activeTools: number;       // 活跃工具数（有请求的）
  totalRequests: number;     // 总请求数
  successRequests: number;   // 成功请求数
  failedRequests: number;    // 失败请求数
  avgSuccessRate: number;    // 平均成功率(%)
  avgResponseTime: number;   // 平均响应时间(ms)
  totalUsers: number;        // 使用工具的用户总数
}

/**
 * Protocol 20001 - Get Tool Usage Summary
 * 获取工具使用情况汇总统计（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol20001(body: Request20001): Promise<Response20001Data> {
  try {
    const { timeRange } = body.params;
    
    // 1. 获取当前proxy的proxyKey（不用token）
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
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60; // 转换为秒
    const startTime = now - timeRangeSeconds;
    
    // 2. 从proxy-api获取有效的server列表
    let totalToolsCount = 0;
    let serversMap: { [serverId: string]: any } = {};
    let validServerIds = new Set<string>();
    try {
      const serversResult = await getServers({ enabled: true }, body.common.userid);
      const serversList = serversResult.servers || [];
      
      // 建立serverId到server的映射，同时计算工具总数
      const uniqueToolNames = new Set();
      serversList.forEach((server: any) => {
        if (server.serverId) {
          serversMap[server.serverId] = server;
          validServerIds.add(server.serverId);
        }
        if (server.serverName) {
          uniqueToolNames.add(server.serverName);
        }
      });
      totalToolsCount = uniqueToolNames.size;
      
      console.log('[Protocol-20001] Total tools from proxy-api:', totalToolsCount);
      console.log('[Protocol-20001] Valid server IDs count:', validServerIds.size);
    } catch (error) {
      console.warn('[Protocol-20001] Failed to get servers from proxy-api:', error);
    }

    // 3. 构建where条件：基于proxyKey、action 1000-1099和有效serverId
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
        in: Array.from(validServerIds),
        not: ''
      }
    };

    // 4. 并行查询统计数据（基于action 1000-1099）
    const [
      totalRequestsCount,
      uniqueUsers
    ] = await Promise.all([
      
      // 总请求数
      prisma.log.count({
        where: whereCondition
      }),
      
      // 使用工具的用户总数
      prisma.log.findMany({
        where: whereCondition,
        select: {
          userid: true
        },
        distinct: ['userid']
      })
    ]);

    const validServerIdList = Array.from(validServerIds);
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
            AND BTRIM(COALESCE(error, '')) = ''
            AND (status_code IS NULL OR status_code < 400)
        `,
        prisma.$queryRaw<Array<{ avg_duration: number | null }>>`
          SELECT AVG(duration)::float8 AS avg_duration
          FROM log
          WHERE proxy_key = ${proxyKey}
            AND addtime >= ${BigInt(startTime)}
            AND action BETWEEN 1000 AND 1099
            AND server_id <> ''
            AND server_id IN (${Prisma.join(validServerIdList)})
            AND BTRIM(COALESCE(error, '')) = ''
            AND (status_code IS NULL OR status_code < 400)
            AND duration > 0
        `,
      ]);

      const successCountValue = successCountRows[0]?.count;
      successRequestsCount = typeof successCountValue === 'bigint' ? Number(successCountValue) : Number(successCountValue || 0);
      avgDuration = Number(avgDurationRows[0]?.avg_duration || 0);
    }
    
    // 5. 计算统计结果
    const totalTools = totalToolsCount;
    
    // 活跃工具数：基于proxy-api 3004获取真正启动的服务器数量
    let activeToolsCount = 0;
    try {
      const serversStatus = await getServersStatus(body.common.userid);
      // 过滤出状态为Online(0)的服务器
      const onlineServerIds = Object.keys(serversStatus).filter(serverId => 
        serversStatus[serverId] === 0 // ServerStatus.Online = 0
      );
      
      // 通过工具名称去重计算活跃工具数（使用统一的分组规则）
      const uniqueActiveToolNames = new Set();
      onlineServerIds.forEach(serverId => {
        if (serversMap[serverId]) {
          // 存在的工具，使用serverName
          uniqueActiveToolNames.add(serversMap[serverId].serverName);
        } else {
          // 已删除的工具，归类为Unknown（但在线服务器状态下，这种情况应该很少见）
          uniqueActiveToolNames.add('Unknown');
        }
      });
      
      activeToolsCount = uniqueActiveToolNames.size;
      console.log('[Protocol-20001] Active tools from server status:', activeToolsCount, 'online servers:', onlineServerIds.length);
    } catch (error) {
      console.error('[Protocol-20001] Failed to get servers status:', error);
      // 回退：活跃工具数设为0或基于已配置工具数
      activeToolsCount = 0;
      console.warn('[Protocol-20001] Using fallback: activeTools = 0 due to server status query failure');
    }
    
    // 请求统计
    const totalRequests = totalRequestsCount;
    const successRequests = successRequestsCount;
    const failedRequests = totalRequests - successRequests;
    
    // 计算成功率
    const avgSuccessRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
    
    // 计算平均响应时间（毫秒）
    const avgResponseTime = Math.round(avgDuration);
    
    // 用户总数
    const totalUsers = uniqueUsers.filter(u => u.userid).length;
    
    const response: Response20001Data = {
      totalTools,
      activeTools: activeToolsCount,
      totalRequests,
      successRequests,
      failedRequests,
      avgSuccessRate: Math.round(avgSuccessRate * 10) / 10, // 保留1位小数
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
    console.error('Protocol 20001 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool usage summary statistics' });
  }
}
