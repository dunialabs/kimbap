import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';
import { Prisma } from '@prisma/client';



interface Request21001 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number; // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
  };
}

interface Response21001Data {
  totalTokens: number;        // 令牌总数
  activeTokens: number;       // 活跃令牌数（有请求的）
  totalRequests: number;      // 总请求数
  successRequests: number;    // 成功请求数
  failedRequests: number;     // 失败请求数
  avgSuccessRate: number;     // 平均成功率(%)
  totalClients: number;       // 连接的客户端总数
  expiredTokens: number;      // 过期令牌数
  limitedTokens: number;      // 达到速率限制的令牌数
}

/**
 * Protocol 21001 - Get Token Usage Summary  
 * 获取访问令牌使用情况汇总统计（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol21001(body: Request21001): Promise<Response21001Data> {
  try {
    const { timeRange } = body.params;
    const rawToken = body.common?.rawToken;
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21001] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-21001] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    let validUserIds = new Set<string>();
    let expiredTokensCount = 0;
    try {
      const usersResult = await getUsers({}, body.common.userid, rawToken);
      const usersList = usersResult.users || [];
      usersList.forEach((user: any) => {
        if (user.userId) {
          validUserIds.add(user.userId);
          if (user.expiresAt > 0 && user.expiresAt < now) expiredTokensCount++;
        }
      });
    } catch (error) {
      console.warn('[Protocol-21001] Failed to get users from proxy-api:', error);
    }
    
    // 3. 构建日志查询条件（基于proxyKey和action 1000-1099）
    const logWhereCondition: any = {
      proxyKey: proxyKey,
      action: {
        gte: 1000,
        lte: 1099
      },
      addtime: {
        gte: BigInt(startTime)
      },
      userid: {
        not: ''
      }
    };

    const validUserIdList = Array.from(validUserIds);

    // 4. 并行查询统计数据
    const [
      totalRequests,
      activeUsersFromLogs
    ] = await Promise.all([
      // 总请求数（action 1000-1099，过滤有效用户）
      prisma.log.count({
        where: {
          ...logWhereCondition,
          userid: { in: validUserIdList }
        }
      }),
      
      // 活跃用户数（指定时间范围内有请求的用户）
      prisma.log.findMany({
        where: {
          ...logWhereCondition,
          userid: { in: validUserIdList }
        },
        select: {
          userid: true
        },
        distinct: ['userid']
      })
    ]);
    let successRequests = 0;

    if (validUserIdList.length > 0) {
      const successCountRows = await prisma.$queryRaw<Array<{ count: bigint | number }>>`
        SELECT COUNT(*) AS count
        FROM log
        WHERE proxy_key = ${proxyKey}
          AND action BETWEEN 1000 AND 1099
          AND addtime >= ${BigInt(startTime)}
          AND userid <> ''
          AND userid IN (${Prisma.join(validUserIdList)})
          AND BTRIM(COALESCE(error, '')) = ''
          AND (status_code IS NULL OR status_code < 400)
      `;

      const successCountValue = successCountRows[0]?.count;
      successRequests = typeof successCountValue === 'bigint' ? Number(successCountValue) : Number(successCountValue || 0);
    }
    
    // 5. 计算统计指标
    const totalTokens = validUserIds.size; // 总用户数即总token数
    const activeTokens = activeUsersFromLogs.length; // 活跃用户数
    const failedRequests = totalRequests - successRequests;
    const avgSuccessRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
    
    // 客户端数（基于session_id去重）
    const uniqueClients = await prisma.log.findMany({
      where: {
        ...logWhereCondition,
        userid: { in: validUserIdList },
        sessionId: { not: '' }
      },
      select: {
        sessionId: true
      },
      distinct: ['sessionId']
    });
    
    const response: Response21001Data = {
      totalTokens,
      activeTokens,
      totalRequests,
      successRequests,
      failedRequests,
      avgSuccessRate: Math.round(avgSuccessRate * 10) / 10,
      totalClients: uniqueClients.length,
      expiredTokens: expiredTokensCount,
      limitedTokens: 0
    };
    
    console.log('[Protocol-21001] Response:', response, 'proxyKey:', proxyKey);
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21001 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage summary statistics' });
  }
}
