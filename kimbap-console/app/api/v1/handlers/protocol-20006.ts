import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request20006 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;   // : 1-, 7-7, 30-30, 90-90
    serverId: number;    // ID，0
    metricType: number;  // : 1-, 2-, 3-
  };
}

interface ToolComparison {
  toolId: string;      // ID
  toolName: string;    // 
  avgValue: number;    // 
  minValue: number;    // 
  maxValue: number;    // 
}

interface Response20006Data {
  comparison: ToolComparison[];
}

/**
 * Protocol 20006 - Get Tool Performance Comparison
 * （）
 */
export async function handleProtocol20006(body: Request20006): Promise<Response20006Data> {
  try {
    const { serverId, metricType } = body.params;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-20006] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
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
    
    let comparison: ToolComparison[] = [];
    
    switch (metricType) {
      case 1: { // 
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
          _min: {
            duration: true
          },
          _max: {
            duration: true
          }
        });
        
        comparison = responseTimeStats
          .filter(item => item.serverId && item._avg.duration)
          .map(item => ({
            toolId: item.serverId!,
            toolName: `Tool ${item.serverId}`,
            avgValue: Math.round(item._avg.duration!),
            minValue: Math.round(item._min.duration!),
            maxValue: Math.round(item._max.duration!)
          }))
          .sort((a, b) => a.avgValue - b.avgValue); // 
        break;
      }
      
      case 2: { // 
        const successRateStats = await prisma.log.groupBy({
          by: ['serverId'],
          where: whereCondition,
          _count: {
            id: true
          }
        });
        
        // 
        const toolSuccessRates = await Promise.all(
          successRateStats
            .filter(item => item.serverId)
            .map(async (item) => {
              const toolId = item.serverId!;
              const totalRequests = item._count.id;
              
              const successRequests = await prisma.log.count({
                where: {
                  ...whereCondition,
                  serverId: toolId,
                  error: { in: ['', null] },
                  OR: [
                    { statusCode: null },
                    { statusCode: { gte: 200, lt: 400 } },
                  ],
                }
              });
              
              const successRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
              
              return {
                toolId,
                toolName: `Tool ${toolId}`,
                avgValue: Math.round(successRate * 10) / 10, // 1
                minValue: 0, // 0
                maxValue: 100, // 100
                totalRequests
              };
            })
        );
        
        comparison = toolSuccessRates
          .filter(item => item.totalRequests > 0) // 
          .map(item => ({
            toolId: item.toolId,
            toolName: item.toolName,
            avgValue: item.avgValue,
            minValue: item.minValue,
            maxValue: item.maxValue
          }))
          .sort((a, b) => b.avgValue - a.avgValue); // 
        break;
      }
      
      case 3: { // 
        // 
        const dailyStats = await prisma.log.findMany({
          where: whereCondition,
          select: {
            serverId: true,
            addtime: true
          }
        });
        
        // 
        const toolDayMap = new Map<string, Map<string, number>>();

        dailyStats.forEach(log => {
          const toolId = log.serverId!;
          const d = new Date(Number(log.addtime) * 1000);
          const dayKey = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
          if (!toolDayMap.has(toolId)) toolDayMap.set(toolId, new Map());
          const dayBuckets = toolDayMap.get(toolId)!;
          dayBuckets.set(dayKey, (dayBuckets.get(dayKey) ?? 0) + 1);
        });

        const toolDailyStats = new Map<string, number[]>(
          Array.from(toolDayMap.entries()).map(([toolId, dayBuckets]) => [toolId, Array.from(dayBuckets.values())])
        );
        
        // 
        comparison = Array.from(toolDailyStats.entries())
          .map(([toolId, dailyCounts]) => {
            const totalRequests = dailyCounts.reduce((sum, count) => sum + count, 0);
            const avgRequests = dailyCounts.length > 0 ? totalRequests / dailyCounts.length : 0;
            const minRequests = dailyCounts.length > 0 ? Math.min(...dailyCounts) : 0;
            const maxRequests = dailyCounts.length > 0 ? Math.max(...dailyCounts) : 0;
            
            return {
              toolId,
              toolName: `Tool ${toolId}`,
              avgValue: Math.round(avgRequests),
              minValue: minRequests,
              maxValue: maxRequests
            };
          })
          .sort((a, b) => b.avgValue - a.avgValue); // 
        break;
      }
      
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported comparison metric type' });
    }
    
    const response: Response20006Data = {
      comparison
    };
    
    console.log('Protocol 20006 response:', {
      metricType,
      comparisonCount: comparison.length,
      timeRange: normalizedTimeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20006 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool performance comparison' });
  }
}
