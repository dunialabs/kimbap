import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';



interface Request22003 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number; // time range: 1-today, 7-recent7day, 30-recent30day
    limit: number;     // Return quantity limit，default5
  };
}

interface ActiveTokenStat {
  tokenName: string;         // Token name
  tokenMask: string;         // Token mask（Desensitized display）
  requestCount: number;      // Number of requests
  isCurrentlyActive: boolean; // Is it currently active?（recent5Request within minutes）
  lastUsedMinutesAgo: number; // last use time（how many minutes ago）
}

interface Response22003Data {
  tokens: ActiveTokenStat[];  // Changed from activeTokens to tokens to match frontend expectation
}

/**
 * Protocol 22003 - Get Active Tokens Overview
 * Get an overview of active tokens
 */
export async function handleProtocol22003(body: Request22003): Promise<Response22003Data> {
  try {
    const rawToken = body.common?.rawToken;
    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : 1;
    const parsedLimit = Number(body.params?.limit);
    const normalizedLimit = Math.floor(parsedLimit);
    const limit = Number.isFinite(normalizedLimit) && normalizedLimit >= 1 ? normalizedLimit : 5;
    
    // 1. Get currentproxyofproxyKey（Need nottoken）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-22003] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-22003] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. fromproxy-apiGet user list（Filter deleted users）
    let allUsers: any[] = [];
    try {
      const usersResult = await getUsers({}, body.common.userid, rawToken);
      allUsers = usersResult.users || [];
      console.log('[Protocol-22003] Got valid users count:', allUsers.length);
    } catch (error) {
      console.warn('[Protocol-22003] Failed to get users from proxy-api:', error);
      allUsers = [];
    }
    
    // Calculation time range
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    const fiveMinutesAgo = now - (5 * 60);
    
    // 3. based onproxyKey、action 1000-1099and non-emptyuseridQuery activity token
    const validUserIds = allUsers.map(u => u.userId).filter(Boolean);
    
    const tokenWhereCondition: any = {
      proxyKey: proxyKey,
      action: { gte: 1000, lte: 1099 },
      addtime: { gte: BigInt(startTime) },
      userid: { not: '' }
    };

    if (validUserIds.length > 0) {
      tokenWhereCondition.userid = {
        in: validUserIds,
        not: ''
      };
    }

    const tokenUsageStats = await prisma.log.groupBy({
      by: ['userid'],
      where: tokenWhereCondition,
      _count: { id: true },
      _max: { addtime: true },
    });
    
    // Create user mapping table
    const usersMap = new Map(allUsers.map(u => [u.userId, u]));
    
    // Convert to responsive format
    const activeTokens: ActiveTokenStat[] = tokenUsageStats.map((stat) => {
      const userId = stat.userid!;
      // skip nouseridof error data has been processed in the query
      const user = usersMap.get(userId);
      const requestCount = stat._count.id;
      const lastUsedTimestamp = Number(stat._max.addtime);
      const lastUsedMinutesAgo = Math.floor((now - lastUsedTimestamp) / 60);
      const isCurrentlyActive = lastUsedTimestamp > fiveMinutesAgo;
      
      let tokenName: string;
      if (user) {
        // User exists，showname(userName)，ifuserNameandnameIf the same or empty, only displayname
        if (user.userName && user.userName !== user.name) {
          tokenName = `${user.name}(${user.userName})`;
        } else {
          tokenName = user.name || userId;
        }
      } else {
        // User has been deleted，showuserid + (Deleted)
        tokenName = `${userId} (Deleted)`;
      }
      
      return {
        tokenName,
        tokenMask: userId.substring(0, 8) + '...',
        requestCount,
        isCurrentlyActive,
        lastUsedMinutesAgo
      };
    });
    
    // Sort by activity：Currently active ones are ranked first，Then sort by number of requests
    activeTokens.sort((a, b) => {
      if (a.isCurrentlyActive && !b.isCurrentlyActive) return -1;
      if (!a.isCurrentlyActive && b.isCurrentlyActive) return 1;
      return b.requestCount - a.requestCount;
    });
    
    const response: Response22003Data = {
      tokens: activeTokens.slice(0, limit)
    };
    
    console.log('Protocol 22003 response:', {
      activeTokensCount: activeTokens.length,
      currentlyActiveCount: activeTokens.filter(t => t.isCurrentlyActive).length,
      totalRequests: activeTokens.reduce((sum, t) => sum + t.requestCount, 0),
      timeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 22003 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get active token overview' });
  }
}
