import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21006 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;   // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    metricType: number;  // 指标类型: 1-按请求数, 2-按客户端数, 3-按成功率
  };
}

interface TokenDistribution {
  tokenId: string;      // 令牌ID
  tokenName: string;    // 令牌名称
  value: number;        // 数值（请求数/客户端数/成功率*100）
  percentage: number;   // 占比(%)
}

interface Response21006Data {
  distribution: TokenDistribution[];
}

/**
 * Protocol 21006 - Get Token Distribution
 * 获取令牌使用分布数据（用于饼图）
 */
export async function handleProtocol21006(body: Request21006): Promise<Response21006Data> {
  try {
    const { timeRange, metricType } = body.params;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-21006] Failed to get proxy info:', error);
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
      tokenMask: {
        not: ''
      }
    };
    
    let distribution: TokenDistribution[] = [];
    
    switch (metricType) {
      case 1: { // 按请求数分布
        const requestCounts = await prisma.log.groupBy({
          by: ['tokenMask'],
          where: whereCondition,
          _count: {
            id: true
          }
        });
        
        const totalRequests = requestCounts.reduce((sum, item) => sum + item._count.id, 0);
        
        distribution = requestCounts
          .filter(item => item.tokenMask)
          .map(item => ({
            tokenId: item.tokenMask!.substring(0, 8) + '...',
            tokenName: `Token ${item.tokenMask!.substring(0, 8)}...`,
            value: item._count.id,
            percentage: totalRequests > 0 ? Math.round((item._count.id / totalRequests) * 1000) / 10 : 0
          }))
          .sort((a, b) => b.value - a.value);
        break;
      }
      
      case 2: { // 按客户端数分布
        // 先获取所有token
        const allTokens = await prisma.log.findMany({
          where: whereCondition,
          select: {
            tokenMask: true
          },
          distinct: ['tokenMask']
        });
        
        // 为每个token计算独立客户端数
        const tokenClientCounts = await Promise.all(
          allTokens
            .filter(token => token.tokenMask)
            .map(async (token) => {
              const uniqueClients = await prisma.log.findMany({
                where: {
                  ...whereCondition,
                  tokenMask: token.tokenMask,
                  ip: {
                    not: ''
                  }
                },
                select: {
                  ip: true
                },
                distinct: ['ip']
              });
              
              return {
                tokenMask: token.tokenMask!,
                clientCount: uniqueClients.length
              };
            })
        );
        
        const totalClients = tokenClientCounts.reduce((sum, item) => sum + item.clientCount, 0);
        
        distribution = tokenClientCounts
          .filter(item => item.clientCount > 0)
          .map(item => ({
            tokenId: item.tokenMask.substring(0, 8) + '...',
            tokenName: `Token ${item.tokenMask.substring(0, 8)}...`,
            value: item.clientCount,
            percentage: totalClients > 0 ? Math.round((item.clientCount / totalClients) * 1000) / 10 : 0
          }))
          .sort((a, b) => b.value - a.value);
        break;
      }
      
      case 3: { // 按成功率分布
        const tokenStats = await prisma.log.groupBy({
          by: ['tokenMask'],
          where: whereCondition,
          _count: {
            id: true
          }
        });
        
        // 为每个token计算成功率
        const tokenSuccessRates = await Promise.all(
          tokenStats
            .filter(item => item.tokenMask)
            .map(async (item) => {
              const totalRequests = item._count.id;
              
              // 查询成功请求数
              const successRequests = await prisma.log.count({
                where: {
                  ...whereCondition,
                  tokenMask: item.tokenMask,
                  statusCode: {
                    gte: 200,
                    lt: 300
                  }
                }
              });
              
              const successRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
              
              return {
                tokenMask: item.tokenMask!,
                successRate: Math.round(successRate * 10) / 10,
                totalRequests
              };
            })
        );
        
        distribution = tokenSuccessRates
          .filter(item => item.totalRequests > 0) // 只显示有请求的token
          .map(item => ({
            tokenId: item.tokenMask.substring(0, 8) + '...',
            tokenName: `Token ${item.tokenMask.substring(0, 8)}...`,
            value: Math.round(item.successRate), // 成功率作为整数值
            percentage: 0 // 成功率分布不使用百分比，而是显示实际成功率
          }))
          .sort((a, b) => b.value - a.value);
        
        // 为成功率分布计算相对百分比（基于最高成功率）
        if (distribution.length > 0) {
          const maxSuccessRate = distribution[0].value;
          distribution = distribution.map(item => ({
            ...item,
            percentage: maxSuccessRate > 0 ? Math.round((item.value / maxSuccessRate) * 100) : 0
          }));
        }
        break;
      }
      
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported metric type' });
    }
    
    const response: Response21006Data = {
      distribution
    };
    
    console.log('Protocol 21006 response:', {
      metricType,
      distributionCount: distribution.length,
      timeRange
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 21006 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage distribution' });
  }
}
