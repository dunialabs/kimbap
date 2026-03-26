import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers } from '@/lib/proxy-api';



interface Request20005 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;
    serverId: number;
    metricType: number;
  };
}

interface Distribution {
  toolId: string;      // ID
  toolName: string;    // 
  value: number;       // （//）
  percentage: number;  // (%)
}

interface Response20005Data {
  distribution: Distribution[];
}

/**
 * Protocol 20005 - Get Tool Usage Distribution
 * （）
 */
export async function handleProtocol20005(body: Request20005): Promise<Response20005Data> {
  try {
    const rawToken = body.common?.rawToken;
    const { timeRange, serverId, metricType } = body.params;

    let proxyKey = '';
    let serversMap: Record<string, string> = {};
    try {
      const [proxy, serversResult] = await Promise.all([
        getProxy(),
        getServers({}, body.common.userid, rawToken).catch(() => ({ servers: [] })),
      ]);
      proxyKey = proxy.proxyKey;
      (serversResult.servers || []).forEach((s: any) => {
        if (s.serverId) serversMap[s.serverId] = s.serverName || s.serverId;
      });
    } catch (error) {
      console.error('[Protocol-20005] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    // 
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // where
    const whereCondition: any = {
      proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      action: {
        in: [1001, 1002, 1003, 1004, 1005, 1006]
      },
      serverId: {
        not: null
      }
    };
    
    // serverId，
    if (serverId > 0) {
      whereCondition.serverId = serverId.toString();
    }
    
    let distribution: Distribution[] = [];
    
    switch (metricType) {
      case 1: { // 
        const requestCounts = await prisma.log.groupBy({
          by: ['serverId'],
          where: whereCondition,
          _count: {
            id: true
          }
        });
        
        const totalRequests = requestCounts.reduce((sum, item) => sum + item._count.id, 0);
        
        distribution = requestCounts
          .filter(item => item.serverId)
          .map(item => ({
            toolId: item.serverId!,
            toolName: serversMap[item.serverId!] ?? `Tool ${item.serverId}`,
            value: item._count.id,
            percentage: totalRequests > 0 ? Math.round((item._count.id / totalRequests) * 1000) / 10 : 0
          }))
          .sort((a, b) => b.value - a.value);
        break;
      }
      
      case 2: { // 
        // ID
        const allTools = await prisma.log.findMany({
          where: whereCondition,
          select: {
            serverId: true
          },
          distinct: ['serverId']
        });
        
        // 
        const toolUserCounts = await Promise.all(
          allTools
            .filter(tool => tool.serverId)
            .map(async (tool) => {
              const uniqueUsers = await prisma.log.findMany({
                where: {
                  ...whereCondition,
                  serverId: tool.serverId,
                  userid: {
                    not: ''
                  }
                },
                select: {
                  userid: true
                },
                distinct: ['userid']
              });
              
              return {
                toolId: tool.serverId!,
                userCount: uniqueUsers.length
              };
            })
        );
        
        const totalUsers = toolUserCounts.reduce((sum, item) => sum + item.userCount, 0);
        
        distribution = toolUserCounts
          .filter(item => item.userCount > 0)
          .map(item => ({
            toolId: item.toolId,
            toolName: serversMap[item.toolId] ?? `Tool ${item.toolId}`,
            value: item.userCount,
            percentage: totalUsers > 0 ? Math.round((item.userCount / totalUsers) * 1000) / 10 : 0
          }))
          .sort((a, b) => b.value - a.value);
        break;
      }
      
      case 3: { // 
        const responseTimeStats = await prisma.log.groupBy({
          by: ['serverId'],
          where: {
            ...whereCondition,
            duration: {
              not: null
            }
          },
          _avg: {
            duration: true
          },
          _count: {
            id: true
          }
        });
        
        distribution = responseTimeStats
          .filter(item => item.serverId && item._avg.duration)
          .map(item => {
            const avgResponseTime = Math.round(item._avg.duration!);
            return {
              toolId: item.serverId!,
              toolName: serversMap[item.serverId!] ?? `Tool ${item.serverId}`,
              value: avgResponseTime,
              percentage: 0 // ，
            };
          })
          .sort((a, b) => b.value - a.value);
        
        // （）
        if (distribution.length > 0) {
          const maxResponseTime = distribution[0].value;
          distribution = distribution.map(item => ({
            ...item,
            percentage: maxResponseTime > 0 ? Math.round((item.value / maxResponseTime) * 1000) / 10 : 0
          }));
        }
        break;
      }
      
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported metric type' });
    }
    
    const response: Response20005Data = {
      distribution
    };
    
    console.log('Protocol 20005 response:', {
      metricType,
      distributionCount: distribution.length,
      timeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20005 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool usage distribution' });
  }
}
