import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';

interface Request21003 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;      // 时间范围: 7-最近7天, 30-最近30天, 90-最近90天
    tokenIds: string[];     // 要查看趋势的令牌ID列表，空表示所有令牌
    granularity: number;    // 数据粒度: 1-按小时, 2-按天, 3-按周
  };
}

interface TrendPoint {
  date: string;           // 日期/时间点
  [tokenName: string]: string | number; // 动态属性，每个token作为一个属性
}

interface Response21003Data {
  trends: TrendPoint[];
}

/**
 * Protocol 21003 - Get Token Usage Trends
 * 获取令牌使用趋势数据（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol21003(body: Request21003): Promise<Response21003Data> {
  try {
    const { timeRange, tokenIds, granularity } = body.params;
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21003] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-21003] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. 从proxy-api获取用户列表（包括有效用户）
    let validUsers: any[] = [];
    try {
      const filters: any = {};
      if (tokenIds && tokenIds.length > 0) {
        // 如果指定了tokenIds，只获取这些用户
        filters.userIds = tokenIds;
      }
      
      const usersResult = await getUsers(filters, body.common.userid);
      validUsers = usersResult.users || [];
      console.log('[Protocol-21003] Got valid users:', validUsers.length);
    } catch (error) {
      console.warn('[Protocol-21003] Failed to get users from proxy-api:', error);
      validUsers = [];
    }
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 计算时间间隔
    let intervalSeconds: number;
    let dateFormat: (timestamp: number) => string;
    
    switch (granularity) {
      case 1: // 按小时
        intervalSeconds = 60 * 60;
        dateFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`;
        };
        break;
      case 3: // 按周
        intervalSeconds = 7 * 24 * 60 * 60;
        dateFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          const startOfWeek = new Date(date);
          startOfWeek.setDate(date.getDate() - date.getDay());
          return `${String(startOfWeek.getMonth() + 1).padStart(2, '0')}-${String(startOfWeek.getDate()).padStart(2, '0')}`;
        };
        break;
      case 2: // 按天（默认）
      default:
        intervalSeconds = 24 * 60 * 60;
        dateFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
        };
        break;
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
    
    // 生成时间点列表
    const timePoints: number[] = [];
    for (let time = startTime; time <= now; time += intervalSeconds) {
      timePoints.push(time);
    }
    
    // 3.5. 获取所有在日志中出现的unique userid（包括已删除用户的数据）
    const allLogUserIds = await prisma.log.findMany({
      where: logWhereCondition,
      select: {
        userid: true
      },
      distinct: ['userid']
    });
    
    let uniqueUserIds = allLogUserIds
      .map(log => log.userid!)
      .filter(Boolean); // 过滤掉没有userid的错误数据
    
    // 过滤特定tokenIds（如果指定）
    if (tokenIds && tokenIds.length > 0) {
      uniqueUserIds = uniqueUserIds.filter(userId => tokenIds.includes(userId));
    }
    
    // 创建用户映射表
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

    // Build a lookup: bucketKey (timePoint + userId) -> count
    const countMap = new Map<string, number>();
    for (const log of allLogs) {
      const uid = log.userid;
      if (!uid || !uniqueUserIds.includes(uid)) continue;
      const ts = Number(log.addtime);
      const bucket = timePoints.findIndex((tp, i) => {
        const next = timePoints[i + 1] ?? (tp + intervalSeconds);
        return ts >= tp && ts < next;
      });
      if (bucket === -1) continue;
      const key = `${timePoints[bucket]}:${uid}`;
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
      timeRange,
      proxyKey
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 21003 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage trends' });
  }
}