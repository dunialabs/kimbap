import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';
import { getActionMachineName, getSourceLabelFromAction } from '@/lib/log-utils';

interface Request21011 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    limit?: number; // 返回记录数，默认50
    lastId?: number; // 上次获取的最后一条记录ID，用于实现增量更新
    timeRange?: number;
    userIds?: string[];
  };
}

interface LogRecord {
  id: number;
  addtime: number;
  action: number;
  source: string; // 根据action确定的source
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
  timestamp: string; // 格式化的时间戳
  requestParams: string; // 请求参数
  responseResult: string; // 响应结果
}

interface Response21011Data {
  logs: LogRecord[];
  totalCount: number;
  maxId: number; // 当前批次最大的ID，用于客户端增量更新
}

/**
 * Protocol 21011 - Get Recent Log Records (Real-time)
 * 获取最近的日志记录（用于token-usage页面的实时table view）
 */
export async function handleProtocol21011(body: Request21011): Promise<Response21011Data> {
  try {
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
    
    // 1. 获取当前proxy的proxyKey（不用token）
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
    
    // 2. 构建查询条件
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
    
    // 如果指定了lastId，只获取比这个ID更新的记录
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
    
    // 3. 查询最近的日志记录
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
    
    // 4. 获取总数（与当前查询条件保持一致，避免分页总数错位）
    const totalCount = await prisma.log.count({
      where: whereCondition
    });

    let usersMap: Record<string, string> = {};
    try {
      const usersResult = await getUsers({}, body.common.userid);
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
    
    // 5. 转换数据格式
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
        requestParams: log.requestParams,
        responseResult: log.responseResult
      };
    });
    
    // 6. 获取最大ID用于增量更新
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
    console.error('Protocol 21011 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
      details: 'Failed to get recent log records' 
    });
  }
}
