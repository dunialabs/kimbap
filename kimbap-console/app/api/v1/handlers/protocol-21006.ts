import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21006 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;   // : 1-, 7-7, 30-30, 90-90
    metricType: number;  // : 1-, 2-, 3-
  };
}

interface TokenDistribution {
  tokenId: string;      // ID
  tokenName: string;    // 
  value: number;        // （//*100）
  percentage: number;   // (%)
}

interface Response21006Data {
  distribution: TokenDistribution[];
}

/**
 * Protocol 21006 - Get Token Distribution
 * （）
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
      tokenMask: {
        not: ''
      }
    };
    
    let distribution: TokenDistribution[] = [];
    
    switch (metricType) {
      case 1: { // 
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
      
      case 2: { // 
        // token
        const allTokens = await prisma.log.findMany({
          where: whereCondition,
          select: {
            tokenMask: true
          },
          distinct: ['tokenMask']
        });
        
        // token
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
      
      case 3: { // 
        const tokenStats = await prisma.log.groupBy({
          by: ['tokenMask'],
          where: whereCondition,
          _count: {
            id: true
          }
        });
        
        // token
        const tokenSuccessRates = await Promise.all(
          tokenStats
            .filter(item => item.tokenMask)
            .map(async (item) => {
              const totalRequests = item._count.id;
              
              // 
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
          .filter(item => item.totalRequests > 0) // token
          .map(item => ({
            tokenId: item.tokenMask.substring(0, 8) + '...',
            tokenName: `Token ${item.tokenMask.substring(0, 8)}...`,
            value: Math.round(item.successRate), // 
            percentage: 0 // ，
          }))
          .sort((a, b) => b.value - a.value);
        
        // （）
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
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21006 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage distribution' });
  }
}
