import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';
import { getActionMachineName, getSourceLabelFromAction } from '@/lib/log-utils';

interface Request21011 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
    userRole?: number;
  };
  params: {
    limit?: number; // Returns the number of records, default 50
    lastId?: number; // The last record ID obtained last time, used to implement incremental updates
    timeRange?: number;
    userIds?: string[];
    includeSensitivePayloads?: boolean;
  };
}

interface LogRecord {
  id: number;
  addtime: number;
  action: number;
  source: string; // Source determined based on action
  actionName: string;
  userid: string;
  userName: string;
  serverId?: string;
  sessionId: string;
  ip: string;
  ua: string;
  tokenMask: string;
  error: string;
  duration?: number;
  statusCode?: number;
  timestamp: string; // Formatted timestamp
  requestParams: string | null; // Request parameters
  responseResult: string | null; // response result
}

interface Response21011Data {
  logs: LogRecord[];
  totalCount: number;
  maxId: number; // The largest ID of the current batch, used for client incremental updates
}

/**
 * Protocol 21011 - Get Recent Log Records (Real-time)
 * Get recent log records (for real-time table view of token-usage page)
 */
export async function handleProtocol21011(body: Request21011): Promise<Response21011Data> {
  try {
    const rawToken = body.common?.rawToken;
    const parsedLimit = Number(body.params?.limit);
    const normalizedLimit = Math.floor(parsedLimit);
    const limit = Number.isFinite(normalizedLimit) && normalizedLimit >= 1 ? normalizedLimit : 50;

    const parsedLastId = Number(body.params?.lastId);
    const normalizedLastId = Math.floor(parsedLastId);
    const lastId = Number.isFinite(normalizedLastId) && normalizedLastId >= 1 ? normalizedLastId : undefined;

    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : undefined;
    const userIds = Array.isArray(body.params?.userIds)
      ? body.params.userIds.filter((userId): userId is string => typeof userId === 'string' && userId.trim().length > 0)
      : [];
    const includeSensitivePayloads = body.params?.includeSensitivePayloads === true;
    const isOwner = body.common?.userRole === 1;
    const shouldIncludeSensitivePayloads = includeSensitivePayloads && isOwner;
    
    // 1. Get the proxyKey of the current proxy (no token is needed)
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21011] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-21011] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. Construct query conditions
    const whereCondition: any = {
      proxyKey: proxyKey,
      OR: [
        { action: { gte: 1001, lte: 1009 } },
        { action: { gte: 1201, lte: 1206 } },
        { action: { gte: 1301, lte: 1314 } },
        { action: { gte: 2001, lte: 2010 } },
        { action: { gte: 3001, lte: 3010 } },
        { action: { gte: 4001, lte: 4099 } },
        { action: { gte: 5001, lte: 5011 } },
      ]
    };
    
    // If lastId is specified, only records newer than this ID will be obtained.
    if (lastId && lastId > 0) {
      whereCondition.id = { gt: lastId };
    }

    if (timeRange) {
      const now = Math.floor(Date.now() / 1000);
      whereCondition.addtime = {
        gte: BigInt(now - (timeRange * 24 * 60 * 60))
      };
    }

    if (userIds.length > 0) {
      whereCondition.userid = {
        in: userIds
      };
    }

    const orderBy = lastId
      ? [{ id: 'asc' as const }]
      : [{ id: 'desc' as const }];
    
    // 3. Query recent log records
    const recentLogs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        id: true,
        addtime: true,
        action: true,
        userid: true,
        serverId: true,
        sessionId: true,
        ip: true,
        ua: true,
        tokenMask: true,
        error: true,
        duration: true,
        statusCode: true,
        requestParams: true,
        responseResult: true,
      },
      orderBy,
      take: limit
    });
    
    // 4. Get the total number (consistent with the current query conditions to avoid misalignment of the paging total number)
    const totalCount = await prisma.log.count({
      where: whereCondition
    });

    let usersMap: Record<string, string> = {};
    try {
      const usersResult = await getUsers({}, body.common.userid, rawToken);
      const usersList = usersResult.users || [];
      usersList.forEach((user: any) => {
        if (!user?.userId) return;
        usersMap[user.userId] = user.userName && user.userName !== user.name
          ? `${user.name}(${user.userName})`
          : (user.name || user.userId);
      });
    } catch (error) {
      console.warn('[Protocol-21011] Failed to enrich user names:', error);
    }
    
    // 5. Convert data format
    const logs: LogRecord[] = recentLogs.map(log => {
      const timestamp = new Date(Number(log.addtime) * 1000);
      const source = getSourceLabelFromAction(log.action);
      const actionName = getActionMachineName(log.action);
      
      return {
        id: log.id,
        addtime: Number(log.addtime),
        action: log.action,
        source,
        actionName,
        userid: log.userid,
        userName: usersMap[log.userid] || log.userid,
        serverId: log.serverId || undefined,
        sessionId: log.sessionId,
        ip: log.ip,
        ua: log.ua,
        tokenMask: log.tokenMask,
        error: log.error,
        duration: log.duration || undefined,
        statusCode: log.statusCode || undefined,
        timestamp: timestamp.toLocaleString(),
        requestParams: shouldIncludeSensitivePayloads ? log.requestParams : null,
        responseResult: shouldIncludeSensitivePayloads ? log.responseResult : null
      };
    });
    
    // 6. Get the maximum ID for incremental updates
    const maxId = logs.length > 0 ? Math.max(...logs.map(log => log.id)) : (lastId || 0);
    
    const response: Response21011Data = {
      logs,
      totalCount,
      maxId
    };
    
    console.log('[Protocol-21011] Response:', {
      logsCount: logs.length,
      totalCount,
      maxId,
      isIncremental: !!lastId,
      proxyKey: proxyKey.substring(0, 8) + '...',
      latestLogTime: logs.length > 0 ? logs[0].timestamp : 'No logs'
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21011 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
      details: 'Failed to get recent log records' 
    });
  }
}
