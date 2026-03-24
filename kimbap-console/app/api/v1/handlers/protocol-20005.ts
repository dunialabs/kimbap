import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request20005 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;   // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    serverId: number;    // 服务器ID，0表示所有服务器
    metricType: number;  // 指标类型: 1-按请求数, 2-按用户数, 3-按响应时间
  };
}

interface Distribution {
  toolId: string;      // 工具ID
  toolName: string;    // 工具名称
  value: number;       // 数值（请求数/用户数/响应时间）
  percentage: number;  // 占比(%)
}

interface Response20005Data {
  distribution: Distribution[];
}

/**
 * Protocol 20005 - Get Tool Usage Distribution
 * 获取工具使用分布数据（用于饼图）
 */
export async function handleProtocol20005(body: Request20005): Promise<Response20005Data> {
  try {
    const { timeRange, serverId, metricType } = body.params;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-20005] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 构建where条件
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
    
    // 如果指定了serverId，添加过滤条件
    if (serverId > 0) {
      whereCondition.serverId = serverId.toString();
    }
    
    let distribution: Distribution[] = [];
    
    switch (metricType) {
      case 1: { // 按请求数分布
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
            toolName: `Tool ${item.serverId}`,
            value: item._count.id,
            percentage: totalRequests > 0 ? Math.round((item._count.id / totalRequests) * 1000) / 10 : 0
          }))
          .sort((a, b) => b.value - a.value);
        break;
      }
      
      case 2: { // 按用户数分布
        // 先获取所有工具ID
        const allTools = await prisma.log.findMany({
          where: whereCondition,
          select: {
            serverId: true
          },
          distinct: ['serverId']
        });
        
        // 为每个工具计算独立用户数
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
            toolName: `Tool ${item.toolId}`,
            value: item.userCount,
            percentage: totalUsers > 0 ? Math.round((item.userCount / totalUsers) * 1000) / 10 : 0
          }))
          .sort((a, b) => b.value - a.value);
        break;
      }
      
      case 3: { // 按平均响应时间分布
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
              toolName: `Tool ${item.serverId}`,
              value: avgResponseTime,
              percentage: 0 // 响应时间不计算百分比，而是显示相对比值
            };
          })
          .sort((a, b) => b.value - a.value);
        
        // 为响应时间计算相对百分比（基于最大值）
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
    console.error('Protocol 20005 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool usage distribution' });
  }
}
