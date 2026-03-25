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
    timeRange: number;  // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    tokenIds?: string[]; // 特定令牌ID列表，空表示所有令牌
    page: number;       // 分页-页码
    pageSize: number;   // 分页-每页数量
  };
}

interface Location {
  country: string;    // 国家代码
  city: string;       // 城市名称
  requests: number;   // 请求数
  percentage: number; // 占比(%)
}

interface TokenMetrics {
  tokenId: string;           // 令牌ID（脱敏后）
  tokenName: string;         // 令牌名称
  totalRequests: number;     // 总请求数
  successfulRequests: number; // 成功请求数 (匹配前端)
  failedRequests: number;    // 失败请求数
  rateLimit: number;         // 速率限制(请求/分钟)
  lastUsed: string;          // 最后使用时间(字符串格式)
  status: "active" | "inactive" | "expired" | "limited"; // 状态字符串
  createdDate: string;       // 创建时间字符串
  expiryDate: string | null; // 过期时间字符串，null表示永不过期
  clientCount: number;       // 使用该令牌的客户端数量
  topLocations: Location[];  // 热门使用地点TOP5
  minuteUsage: Array<{       // 分钟级使用模式
    minute: string;
    requests: number;
  }>;
}

interface Response21002Data {
  tokens: TokenMetrics[]; // 改为tokens以匹配前端期望
  totalCount: number; // 总数量（用于分页）
}

/**
 * Protocol 21002 - Get Token Detailed Metrics
 * 获取各令牌详细指标数据（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol21002(body: Request21002): Promise<Response21002Data> {
  try {
    const { timeRange, tokenIds, page = 1, pageSize = 50 } = body.params;
    const rawToken = body.common?.rawToken;
    const normalizedTokenIds = Array.isArray(tokenIds)
      ? tokenIds.map((id) => String(id).trim()).filter(Boolean)
      : [];
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21002] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-21002] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. 从proxy-api获取用户列表（包括有效用户）
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
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
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
    
    // 3. 获取所有在日志中出现的unique userid（包括已删除用户的数据）
    const allLogUserIds = await prisma.log.findMany({
      where: logWhereCondition,
      select: {
        userid: true
      },
      distinct: ['userid']
    });
    
    const uniqueUserIds = allLogUserIds
      .map(log => log.userid!)
      .filter(Boolean); // 过滤掉没有userid的错误数据
    
    console.log('[Protocol-21002] Found unique user IDs in logs:', uniqueUserIds.length);
    
    // 过滤特定tokenIds（如果指定）
    let filteredUserIds = uniqueUserIds;
    if (normalizedTokenIds.length > 0) {
      const tokenIdSet = new Set(normalizedTokenIds);
      filteredUserIds = uniqueUserIds.filter(userId => tokenIdSet.has(userId));
    }
    
    const totalCount = filteredUserIds.length;
    
    // 分页处理
    const offset = (page - 1) * pageSize;
    const pagedUserIds = filteredUserIds.slice(offset, offset + pageSize);
    
    // 创建用户映射表
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

    // 4. 为每个用户计算详细指标
    const tokenMetrics: TokenMetrics[] = [];

    for (const userId of pagedUserIds) {
      const user = usersMap.get(userId);
      let userName: string;
      
      if (user) {
        // 用户存在，显示name(userName)，如果userName和name相同或为空则只显示name
        if (user.userName && user.userName !== user.name) {
          userName = `${user.name}(${user.userName})`;
        } else {
          userName = user.name || userId;
        }
      } else {
        // 用户已删除，显示userid + (Deleted)
        userName = `${userId} (Deleted)`;
      }

      const userLogs = logsByUser.get(userId) || [];
      
      // 5. 计算统计指标
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
      
      // 最后使用时间
      const lastUsedDate = lastUsedTimestamp > 0 ? new Date(lastUsedTimestamp * 1000).toLocaleString() : 'Never';
      
      // 状态判断
      let statusStr: "active" | "inactive" | "expired" | "limited" = "inactive";
      if (totalRequests > 0) {
        const recentTime = now - (24 * 60 * 60); // 最近24小时
        if (lastUsedTimestamp > recentTime) {
          statusStr = "active";
        }
      }
      
      // 5. 计算地理位置分布（基于IP地址）
      // 使用地理位置工具函数聚合统计
      const topLocations: Location[] = aggregateLocationStats(ipRequestCounts);
      
      // 6. 计算分钟级使用模式（最近60分钟）
      const minuteUsage: Array<{ minute: string; requests: number }> = [];
      
      // 生成最近60分钟的时间点
      for (let i = 59; i >= 0; i--) {
        const minuteTime = currentMinute - (i * 60); // 每分钟一个数据点
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
    console.error('Protocol 21002 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token detailed metrics' });
  }
}
