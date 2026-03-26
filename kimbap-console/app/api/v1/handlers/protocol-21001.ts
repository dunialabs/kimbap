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
    timeRange: number; // Time range: 1-today, 7-last 7 days, 30-last 30 days, 90-last 90 days
  };
}

interface Response21001Data {
  totalTokens: number;        // Total number of tokens
  activeTokens: number;       // Number of active tokens (requested)
  totalRequests: number;      // Total requests
  successRequests: number;    // Number of successful requests
  failedRequests: number;     // Number of failed requests
  avgSuccessRate: number;     // Average success rate (%)
  totalClients: number;       // Total number of connected clients
  expiredTokens: number;      // Number of expired tokens
  limitedTokens: number;      // Number of tokens reaching rate limit
}

/**
 * Protocol 21001 - Get Token Usage Summary  
 * Get access token usage summary statistics (based on proxyKey and action 1000-1099)
 */
export async function handleProtocol21001(body: Request21001): Promise<Response21001Data> {
  try {
    const rawToken = body.common?.rawToken;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;
    
    // 1. Get the proxyKey of the current proxy (no token is needed)
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
    
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
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
    
    // 3. Construct log query conditions (based on proxyKey and action 1000-1099)
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

    // 4. Parallel query statistics
    const [
      totalRequests,
      activeUsersFromLogs
    ] = await Promise.all([
      // Total number of requests (action 1000-1099, filter effective users)
      prisma.log.count({
        where: {
          ...logWhereCondition,
          userid: { in: validUserIdList }
        }
      }),
      
      // Number of active users (users with requests within the specified time range)
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
    
    // 5. Calculate statistical indicators
    const totalTokens = validUserIds.size; // The total number of users is the total number of tokens
    const activeTokens = activeUsersFromLogs.length; // Number of active users
    const failedRequests = totalRequests - successRequests;
    const avgSuccessRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
    
    // Number of clients (based on session_id deduplication)
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
