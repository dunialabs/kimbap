import { ApiError, ErrorCode } from '@/lib/error-codes';
import { isSuccessfulRequestLog } from '@/lib/log-utils';
import { getProxy, getUsers } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { aggregateLocationStats } from '@/lib/geo-location';

interface Request21002 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;  // : 1-, 7-7, 30-30, 90-90
    tokenIds?: string[]; // ID，
    page: number;       // -
    pageSize: number;   // -
  };
}

interface Location {
  country: string;    // 
  city: string;       // 
  requests: number;   // 
  percentage: number; // (%)
}

interface TokenMetrics {
  tokenId: string;           // ID（）
  tokenName: string;         // 
  totalRequests: number;     // 
  successfulRequests: number; //  ()
  failedRequests: number;    // 
  rateLimit: number;         // (/)
  lastUsed: string;          // ()
  status: "active" | "inactive" | "expired" | "limited"; // 
  createdDate: string;       // 
  expiryDate: string | null; // ，null
  clientCount: number;       // 
  topLocations: Location[];  // TOP5
  minuteUsage: Array<{       // 
    minute: string;
    requests: number;
  }>;
}

interface Response21002Data {
  tokens: TokenMetrics[]; // tokens
  totalCount: number; // （）
}

/**
 * Protocol 21002 - Get Token Detailed Metrics
 * （proxyKeyaction 1000-1099）
 */
export async function handleProtocol21002(body: Request21002): Promise<Response21002Data> {
  try {
    const { tokenIds, page = 1, pageSize = 50 } = body.params;
    const rawToken = body.common?.rawToken;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;
    const normalizedTokenIds = Array.isArray(tokenIds)
      ? tokenIds.map((id) => String(id).trim()).filter(Boolean)
      : [];
    
    // 1. proxyproxyKey（token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21002] Got proxyKey:');
    } catch (error) {
      console.error('[Protocol-21002] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. proxy-api（）
    let validUsers: any[] = [];
    try {
      let filters: any = {};
      // proxy-api getUsers currently supports single userId filter only
      if (normalizedTokenIds.length === 1) {
        filters.userId = normalizedTokenIds[0];
      }
      
      const usersResult = await getUsers(filters, body.common.userid, rawToken);
      validUsers = usersResult.users || [];
      console.log('[Protocol-21002] Got valid users from proxy-api:', validUsers.length);
    } catch (error) {
      console.warn('[Protocol-21002] Failed to get users from proxy-api:', error);
      validUsers = [];
    }
    
    // 
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
    // 3. （proxyKeyaction 1000-1099）
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
    
    // 3. unique userid（）
    const allLogUserIds = await prisma.log.findMany({
      where: logWhereCondition,
      select: {
        userid: true
      },
      distinct: ['userid']
    });
    
    const uniqueUserIds = allLogUserIds
      .map(log => log.userid!)
      .filter(Boolean); // userid
    
    console.log('[Protocol-21002] Found unique user IDs in logs:', uniqueUserIds.length);
    
    // tokenIds（）
    let filteredUserIds = uniqueUserIds;
    if (normalizedTokenIds.length > 0) {
      const tokenIdSet = new Set(normalizedTokenIds);
      filteredUserIds = uniqueUserIds.filter(userId => tokenIdSet.has(userId));
    }
    
    const totalCount = filteredUserIds.length;
    
    // 
    const offset = (page - 1) * pageSize;
    const pagedUserIds = filteredUserIds.slice(offset, offset + pageSize);
    
    // 
    const usersMap = new Map(validUsers.map(u => [u.userId, u]));
    
    const pagedUserLogs = pagedUserIds.length > 0
      ? await prisma.log.findMany({
          where: {
            ...logWhereCondition,
            userid: { in: pagedUserIds },
          },
          select: {
            userid: true,
            statusCode: true,
            error: true,
            addtime: true,
            sessionId: true,
            ip: true,
          },
        })
      : [];

    const logsByUser = new Map<string, typeof pagedUserLogs>();
    for (const log of pagedUserLogs) {
      if (!log.userid) {
        continue;
      }
      const existing = logsByUser.get(log.userid);
      if (existing) {
        existing.push(log);
      } else {
        logsByUser.set(log.userid, [log]);
      }
    }

    const currentMinute = Math.floor(now / 60) * 60;
    const sixtyMinutesAgo = currentMinute - (59 * 60);

    // 4. 
    const tokenMetrics: TokenMetrics[] = [];

    for (const userId of pagedUserIds) {
      const user = usersMap.get(userId);
      let userName: string;
      
      if (user) {
        // ，name(userName)，userNamenamename
        if (user.userName && user.userName !== user.name) {
          userName = `${user.name}(${user.userName})`;
        } else {
          userName = user.name || userId;
        }
      } else {
        // ，userid + (Deleted)
        userName = `${userId} (Deleted)`;
      }

      const userLogs = logsByUser.get(userId) || [];
      
      // 5. 
      const totalRequests = userLogs.length;
      let successRequests = 0;
      let lastUsedTimestamp = 0;
      const uniqueSessionIds = new Set<string>();
      const ipRequestCounts = new Map<string, number>();
      const minuteBucketCounts = new Map<number, number>();

      for (const log of userLogs) {
        if (isSuccessfulRequestLog(log)) {
          successRequests += 1;
        }

        const addtime = Number(log.addtime);
        if (addtime > lastUsedTimestamp) {
          lastUsedTimestamp = addtime;
        }

        if (log.sessionId && log.sessionId.trim() !== '') {
          uniqueSessionIds.add(log.sessionId);
        }

        if (log.ip && log.ip.trim() !== '') {
          ipRequestCounts.set(log.ip, (ipRequestCounts.get(log.ip) || 0) + 1);
        }

        if (addtime >= sixtyMinutesAgo && addtime <= currentMinute + 59) {
          const bucket = Math.floor(addtime / 60) * 60;
          minuteBucketCounts.set(bucket, (minuteBucketCounts.get(bucket) || 0) + 1);
        }
      }

      const failedRequests = totalRequests - successRequests;
      const successRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;
      
      // 
      const lastUsedDate = lastUsedTimestamp > 0 ? new Date(lastUsedTimestamp * 1000).toLocaleString() : 'Never';
      
      // 
      let statusStr: "active" | "inactive" | "expired" | "limited" = "inactive";
      if (totalRequests > 0) {
        const recentTime = now - (24 * 60 * 60); // 24
        if (lastUsedTimestamp > recentTime) {
          statusStr = "active";
        }
      }
      
      // 5. （IP）
      // 
      const topLocations: Location[] = aggregateLocationStats(ipRequestCounts);
      
      // 6. （60）
      const minuteUsage: Array<{ minute: string; requests: number }> = [];
      
      // 60
      for (let i = 59; i >= 0; i--) {
        const minuteTime = currentMinute - (i * 60); // 
        const minuteRequests = minuteBucketCounts.get(minuteTime) || 0;
        
        const date = new Date(minuteTime * 1000);
        const timeString = `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
        
        minuteUsage.push({
          minute: timeString,
          requests: minuteRequests
        });
      }
      
      const createdAt = user?.createdAt;
      const expiresAt = user?.expiresAt;
      tokenMetrics.push({
        tokenId: userId,
        tokenName: userName,
        totalRequests,
        successfulRequests: successRequests,
        failedRequests,
        rateLimit: user?.ratelimit ?? 100,
        lastUsed: lastUsedDate,
        status: statusStr,
        createdDate: createdAt ? new Date(createdAt * 1000).toISOString() : '',
        expiryDate: expiresAt && expiresAt > 0 ? new Date(expiresAt * 1000).toISOString() : null,
        clientCount: uniqueSessionIds.size,
        topLocations,
        minuteUsage
      });
    }
    
    const response: Response21002Data = {
      tokens: tokenMetrics,
      totalCount
    };
    
    console.log('Protocol 21002 response:', {
      tokenCount: tokenMetrics.length,
      totalCount,
      page,
      pageSize
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21002 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token detailed metrics' });
  }
}
