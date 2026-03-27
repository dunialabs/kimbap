import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';

interface Request21003 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;      // : 7-7, 30-30, 90-90
    tokenIds: string[];     // ID，
    granularity: number;    // : 1-, 2-, 3-
  };
}

interface TrendPoint {
  date: string;           // /
  [tokenName: string]: string | number; // ，token
}

interface Response21003Data {
  trends: TrendPoint[];
}

/**
 * Protocol 21003 - Get Token Usage Trends
 * （proxyKeyaction 1000-1099）
 */
export async function handleProtocol21003(body: Request21003): Promise<Response21003Data> {
  try {
    const { tokenIds, granularity } = body.params;
    const rawToken = body.common?.rawToken;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;
    
    // 1. proxyproxyKey（token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21003] Got proxyKey:');
    } catch (error) {
      console.error('[Protocol-21003] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. proxy-api（）
    let validUsers: any[] = [];
    try {
      const filters: any = {};
      if (tokenIds && tokenIds.length > 0) {
        // tokenIds，
        filters.userIds = tokenIds;
      }
      
      const usersResult = await getUsers(filters, body.common.userid, rawToken);
      validUsers = usersResult.users || [];
      console.log('[Protocol-21003] Got valid users:', validUsers.length);
    } catch (error) {
      console.warn('[Protocol-21003] Failed to get users from proxy-api:', error);
      validUsers = [];
    }
    
    // 
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
    // 
    let intervalSeconds: number;
    let dateFormat: (timestamp: number) => string;
    
    switch (granularity) {
      case 1: // 
        intervalSeconds = 60 * 60;
        dateFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`;
        };
        break;
      case 3: // 
        intervalSeconds = 7 * 24 * 60 * 60;
        dateFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          const startOfWeek = new Date(date);
          startOfWeek.setDate(date.getDate() - date.getDay());
          return `${String(startOfWeek.getMonth() + 1).padStart(2, '0')}-${String(startOfWeek.getDate()).padStart(2, '0')}`;
        };
        break;
      case 2: // （）
      default:
        intervalSeconds = 24 * 60 * 60;
        dateFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
        };
        break;
    }
    
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
    
    // 
    const timePoints: number[] = [];
    for (let time = startTime; time <= now; time += intervalSeconds) {
      timePoints.push(time);
    }
    
    // 3.5. unique userid（）
    const allLogUserIds = await prisma.log.findMany({
      where: logWhereCondition,
      select: {
        userid: true
      },
      distinct: ['userid']
    });
    
    let uniqueUserIds = allLogUserIds
      .map(log => log.userid!)
      .filter(Boolean); // userid
    
    // tokenIds（）
    if (tokenIds && tokenIds.length > 0) {
      uniqueUserIds = uniqueUserIds.filter(userId => tokenIds.includes(userId));
    }
    
    // 
    const usersMap = new Map(validUsers.map(u => [u.userId, u]));
    
    console.log('[Protocol-21003] Found unique user IDs in logs:', uniqueUserIds.length);
    
    // 4. Fetch all matching logs in a single query, then aggregate in memory
    const allLogs = await prisma.log.findMany({
      where: logWhereCondition,
      select: {
        userid: true,
        addtime: true,
      },
    });

    const uniqueUserIdSet = new Set(uniqueUserIds);

    // Build a lookup: bucketKey (timePoint + userId) -> count
    const countMap = new Map<string, number>();
    for (const log of allLogs) {
      const uid = log.userid;
      if (!uid || !uniqueUserIdSet.has(uid)) continue;
      const ts = Number(log.addtime);
      const bucketIndex = Math.floor((ts - startTime) / intervalSeconds);
      if (bucketIndex < 0 || bucketIndex >= timePoints.length) continue;
      const key = `${timePoints[bucketIndex]}:${uid}`;
      countMap.set(key, (countMap.get(key) ?? 0) + 1);
    }

    const trends: TrendPoint[] = [];

    for (const timePoint of timePoints) {
      const dateStr = dateFormat(timePoint);
      const trendPoint: TrendPoint = { date: dateStr };

      for (const userId of uniqueUserIds) {
        const user = usersMap.get(userId);
        let userName: string;
        if (user) {
          userName = user.userName && user.userName !== user.name
            ? `${user.name}(${user.userName})`
            : (user.name || userId);
        } else {
          userName = `${userId} (Deleted)`;
        }
        trendPoint[userName] = countMap.get(`${timePoint}:${userId}`) ?? 0;
      }

      trends.push(trendPoint);
    }
    
    const response: Response21003Data = {
      trends
    };
    
    console.log('[Protocol-21003] Response:', {
      trendPointsCount: trends.length,
      usersCount: uniqueUserIds.length,
      validUsersCount: validUsers.length,
      granularity,
      timeRange: normalizedTimeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21003 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage trends' });
  }
}
