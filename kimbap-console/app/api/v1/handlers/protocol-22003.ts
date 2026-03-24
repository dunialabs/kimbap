import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';



interface Request22003 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // 时间范围: 1-今天, 7-最近7天, 30-最近30天
    limit: number;     // 返回数量限制，默认5
  };
}

interface ActiveTokenStat {
  tokenName: string;         // 令牌名称
  tokenMask: string;         // 令牌掩码（脱敏显示）
  requestCount: number;      // 请求数量
  isCurrentlyActive: boolean; // 是否当前活跃（最近5分钟内有请求）
  lastUsedMinutesAgo: number; // 最后使用时间（多少分钟前）
}

interface Response22003Data {
  tokens: ActiveTokenStat[];  // Changed from activeTokens to tokens to match frontend expectation
}

/**
 * Protocol 22003 - Get Active Tokens Overview
 * 获取活跃令牌概览
 */
export async function handleProtocol22003(body: Request22003): Promise<Response22003Data> {
  try {
    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : 1;
    const parsedLimit = Number(body.params?.limit);
    const normalizedLimit = Math.floor(parsedLimit);
    const limit = Number.isFinite(normalizedLimit) && normalizedLimit >= 1 ? normalizedLimit : 5;
    
    // 1. 获取当前proxy的proxyKey（不用token）
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
    
    // 2. 从proxy-api获取用户列表（过滤删除的用户）
    let allUsers: any[] = [];
    try {
      const usersResult = await getUsers({}, body.common.userid);
      allUsers = usersResult.users || [];
      console.log('[Protocol-22003] Got valid users count:', allUsers.length);
    } catch (error) {
      console.warn('[Protocol-22003] Failed to get users from proxy-api:', error);
      allUsers = [];
    }
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    const fiveMinutesAgo = now - (5 * 60);
    
    // 3. 基于proxyKey、action 1000-1099和非空userid查询活动令牌
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
      _count: {
        id: true
      },
      _max: {
        addtime: true
      },
      orderBy: {
        _count: {
          id: 'desc'
        }
      },
      take: limit
    });
    
    // 创建用户映射表
    const usersMap = new Map(allUsers.map(u => [u.userId, u]));
    
    // 转换为响应格式
    const activeTokens: ActiveTokenStat[] = tokenUsageStats.map((stat) => {
      const userId = stat.userid!;
      // 跳过没有userid的错误数据已在查询中处理
      const user = usersMap.get(userId);
      const requestCount = stat._count.id;
      const lastUsedTimestamp = Number(stat._max.addtime);
      const lastUsedMinutesAgo = Math.floor((now - lastUsedTimestamp) / 60);
      const isCurrentlyActive = lastUsedTimestamp > fiveMinutesAgo;
      
      let tokenName: string;
      if (user) {
        // 用户存在，显示name(userName)，如果userName和name相同或为空则只显示name
        if (user.userName && user.userName !== user.name) {
          tokenName = `${user.name}(${user.userName})`;
        } else {
          tokenName = user.name || userId;
        }
      } else {
        // 用户已删除，显示userid + (Deleted)
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
    
    // 按活跃程度排序：当前活跃的排在前面，然后按请求数量排序
    activeTokens.sort((a, b) => {
      if (a.isCurrentlyActive && !b.isCurrentlyActive) return -1;
      if (!a.isCurrentlyActive && b.isCurrentlyActive) return 1;
      return b.requestCount - a.requestCount;
    });
    
    const response: Response22003Data = {
      tokens: activeTokens  // Changed from activeTokens to tokens to match interface
    };
    
    console.log('Protocol 22003 response:', {
      activeTokensCount: activeTokens.length,
      currentlyActiveCount: activeTokens.filter(t => t.isCurrentlyActive).length,
      totalRequests: activeTokens.reduce((sum, t) => sum + t.requestCount, 0),
      timeRange
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 22003 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get active token overview' });
  }
}
