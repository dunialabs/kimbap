import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';



interface Request21005 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    tokenId: string;      // 令牌ID，必填
    patternType: number;  // 模式类型: 1-最近60分钟, 2-最近24小时, 3-最近7天每小时
  };
}

interface UsagePoint {
  timeLabel: string;      // 时间标签 (如: "14:30", "2024-01-15 14:00")
  requests: number;       // 请求数
  successRequests: number; // 成功请求数
  failedRequests: number; // 失败请求数
  rateLimitHits: number;  // 触发速率限制次数
}

interface Response21005Data {
  tokenId: string;        // 令牌ID
  tokenName: string;      // 令牌名称
  patterns: UsagePoint[]; // 使用模式数据点
}

/**
 * Protocol 21005 - Get Token Usage Patterns
 * 获取令牌使用模式数据（基于proxyKey和action 1000-1099）
 */
export async function handleProtocol21005(body: Request21005): Promise<Response21005Data> {
  try {
    const { tokenId, patternType } = body.params;
    const rawToken = body.common?.rawToken;
    
    if (!tokenId || !tokenId.trim()) {
      throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'tokenId', details: 'Token ID cannot be empty' });
    }
    
    // 1. 获取当前proxy的proxyKey（不用token）
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21005] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-21005] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. 从proxy-api验证用户是否有效
    let validUser: any = null;
    try {
      const usersResult = await getUsers({ userId: tokenId }, body.common.userid, rawToken);
      validUser = (usersResult.users || []).find((u: any) => u.userId === tokenId);
      
      if (!validUser) {
        throw new ApiError(ErrorCode.RECORD_NOT_FOUND, 404, { details: 'Specified token not found or deleted' });
      }
      
      console.log('[Protocol-21005] Found valid user:', validUser.userId);
    } catch (error) {
      console.error('[Protocol-21005] Failed to get user from proxy-api:', error);
      if (error instanceof ApiError) throw error;
      throw new ApiError(ErrorCode.RECORD_NOT_FOUND, 404, { details: 'Specified token not found' });
    }
    
    const tokenName = validUser.name || tokenId;
    
    // 根据模式类型计算时间范围和间隔
    const now = Math.floor(Date.now() / 1000);
    let startTime: number;
    let intervalSeconds: number;
    let pointCount: number;
    let timeFormat: (timestamp: number) => string;
    
    switch (patternType) {
      case 1: // 最近60分钟
        startTime = now - (60 * 60);
        intervalSeconds = 60; // 1分钟间隔
        pointCount = 60;
        timeFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
        };
        break;
      case 2: // 最近24小时
        startTime = now - (24 * 60 * 60);
        intervalSeconds = 60 * 60; // 1小时间隔
        pointCount = 24;
        timeFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getHours()).padStart(2, '0')}:00`;
        };
        break;
      case 3: // 最近7天每小时
        startTime = now - (7 * 24 * 60 * 60);
        intervalSeconds = 60 * 60; // 1小时间隔
        pointCount = 7 * 24;
        timeFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${date.getMonth() + 1}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`;
        };
        break;
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported pattern type' });
    }
    
    // 3. 获取指定时间范围内该token的所有日志（基于proxyKey和action 1000-1099）
    const logs = await prisma.log.findMany({
      where: {
        proxyKey: proxyKey,
        action: {
          gte: 1000,
          lte: 1099
        },
        addtime: {
          gte: BigInt(startTime)
        },
        userid: tokenId
      },
      select: {
        addtime: true,
        statusCode: true,
        error: true
      },
      orderBy: {
        addtime: 'asc'
      }
    });
    
    // 生成时间点并统计每个时间点的使用情况
    const patterns: UsagePoint[] = [];
    
    for (let i = 0; i < pointCount; i++) {
      const timePoint = startTime + (i * intervalSeconds);
      const timePointEnd = timePoint + intervalSeconds;
      
      // 筛选该时间段的日志
      const periodLogs = logs.filter(log => {
        const logTime = Number(log.addtime);
        return logTime >= timePoint && logTime < timePointEnd;
      });
      
      // 统计各种请求类型
      let requests = periodLogs.length;
      let successRequests = 0;
      let failedRequests = 0;
      let rateLimitHits = 0;
      
      periodLogs.forEach(log => {
        if (log.statusCode && log.statusCode >= 200 && log.statusCode < 300) {
          successRequests++;
        } else {
          failedRequests++;
          // 检查是否为速率限制错误 (HTTP 429)
          if (log.statusCode === 429) {
            rateLimitHits++;
          }
          // 也可以从error字段检查速率限制
          if (log.error && log.error.includes('rate limit')) {
            rateLimitHits++;
          }
        }
      });
      
      patterns.push({
        timeLabel: timeFormat(timePoint),
        requests,
        successRequests,
        failedRequests,
        rateLimitHits
      });
    }
    
    const response: Response21005Data = {
      tokenId: tokenId,
      tokenName,
      patterns
    };
    
    console.log('[Protocol-21005] Response:', {
      tokenId,
      tokenName,
      patternType,
      pointsCount: patterns.length,
      totalRequests: patterns.reduce((sum, p) => sum + p.requests, 0),
      proxyKey
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 21005 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage pattern' });
  }
}
