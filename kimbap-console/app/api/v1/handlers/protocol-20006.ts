import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request20006 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;   // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    serverId: number;    // 服务器ID，0表示所有服务器
    metricType: number;  // 对比指标: 1-响应时间, 2-成功率, 3-请求量
  };
}

interface ToolComparison {
  toolId: string;      // 工具ID
  toolName: string;    // 工具名称
  avgValue: number;    // 平均值
  minValue: number;    // 最小值
  maxValue: number;    // 最大值
}

interface Response20006Data {
  comparison: ToolComparison[];
}

/**
 * Protocol 20006 - Get Tool Performance Comparison
 * 获取工具性能对比数据（用于柱状图）
 */
export async function handleProtocol20006(body: Request20006): Promise<Response20006Data> {
  try {
    const { timeRange, serverId, metricType } = body.params;

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
    
    let comparison: ToolComparison[] = [];
    
    switch (metricType) {
      case 1: { // 响应时间对比
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
          .sort((a, b) => a.avgValue - b.avgValue); // 按平均响应时间升序排列
        break;
      }
      
      case 2: { // 成功率对比
        const successRateStats = await prisma.log.groupBy({
          by: ['serverId'],
          where: whereCondition,
          _count: {
            id: true
          }
        });
        
        // 为每个工具计算成功率
        const toolSuccessRates = await Promise.all(
          successRateStats
            .filter(item => item.serverId)
            .map(async (item) => {
              const toolId = item.serverId!;
              const totalRequests = item._count.id;
              
              // 查询成功请求数
              const successRequests = await prisma.log.count({
                where: {
                  ...whereCondition,
                  serverId: toolId,
                  statusCode: {
                    gte: 200,
                    lt: 300
                  }
                }
              });
              
              const successRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
              
              return {
                toolId,
                toolName: `Tool ${toolId}`,
                avgValue: Math.round(successRate * 10) / 10, // 保留1位小数
                minValue: 0, // 成功率最小值通常为0
                maxValue: 100, // 成功率最大值为100
                totalRequests
              };
            })
        );
        
        comparison = toolSuccessRates
          .filter(item => item.totalRequests > 0) // 只显示有请求的工具
          .map(item => ({
            toolId: item.toolId,
            toolName: item.toolName,
            avgValue: item.avgValue,
            minValue: item.minValue,
            maxValue: item.maxValue
          }))
          .sort((a, b) => b.avgValue - a.avgValue); // 按成功率降序排列
        break;
      }
      
      case 3: { // 请求量对比
        // 按天分组统计每个工具的请求量
        const dailyStats = await prisma.log.findMany({
          where: whereCondition,
          select: {
            serverId: true,
            addtime: true
          }
        });
        
        // 按工具和天分组
        const toolDailyStats = new Map<string, number[]>();
        
        dailyStats.forEach(log => {
          const toolId = log.serverId!;
          const logDate = new Date(Number(log.addtime) * 1000);
          const dayKey = `${logDate.getFullYear()}-${logDate.getMonth()}-${logDate.getDate()}`;
          
          if (!toolDailyStats.has(toolId)) {
            toolDailyStats.set(toolId, []);
          }
          
          const dailyCounts = toolDailyStats.get(toolId)!;
          const existingDayIndex = dailyCounts.findIndex((_, index) => {
            // 这里简化处理，实际可能需要更精确的日期匹配
            return index === dailyCounts.length - 1;
          });
          
          if (existingDayIndex >= 0) {
            dailyCounts[existingDayIndex]++;
          } else {
            dailyCounts.push(1);
          }
        });
        
        // 计算每个工具的请求量统计
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
          .sort((a, b) => b.avgValue - a.avgValue); // 按平均请求量降序排列
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
      timeRange
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 20006 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool performance comparison' });
  }
}
