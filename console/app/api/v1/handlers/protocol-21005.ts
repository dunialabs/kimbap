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
    tokenId: string;      // Token ID, required
    patternType: number;  // Mode type: 1-Last 60 minutes, 2-Last 24 hours, 3-Last 7 days hourly
  };
}

interface UsagePoint {
  timeLabel: string;      // Time tag (eg: "14:30", "2024-01-15 14:00")
  requests: number;       // Number of requests
  successRequests: number; // Number of successful requests
  failedRequests: number; // Number of failed requests
  rateLimitHits: number;  // Number of trigger rate limits
}

interface Response21005Data {
  tokenId: string;        // Token ID
  tokenName: string;      // Token name
  patterns: UsagePoint[]; // Use pattern data points
}

/**
 * Protocol 21005 - Get Token Usage Patterns
 * Get token usage pattern data (based on proxyKey and action 1000-1099)
 */
export async function handleProtocol21005(body: Request21005): Promise<Response21005Data> {
  try {
    const { tokenId, patternType } = body.params;
    const rawToken = body.common?.rawToken;
    
    if (!tokenId || !tokenId.trim()) {
      throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'tokenId', details: 'Token ID cannot be empty' });
    }
    
    // 1. Get the proxyKey of the current proxy (no token is needed)
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-21005] Got proxyKey:');
    } catch (error) {
      console.error('[Protocol-21005] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
        details: 'Failed to get proxy information' 
      });
    }
    
    // 2. Verify whether the user is valid from proxy-api
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
    
    // Calculate time range and interval based on pattern type
    const now = Math.floor(Date.now() / 1000);
    let startTime: number;
    let intervalSeconds: number;
    let pointCount: number;
    let timeFormat: (timestamp: number) => string;
    
    switch (patternType) {
      case 1: // Last 60 minutes
        startTime = now - (60 * 60);
        intervalSeconds = 60; // 1 minute interval
        pointCount = 60;
        timeFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
        };
        break;
      case 2: // last 24 hours
        startTime = now - (24 * 60 * 60);
        intervalSeconds = 60 * 60; // 1 hour interval
        pointCount = 24;
        timeFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${String(date.getHours()).padStart(2, '0')}:00`;
        };
        break;
      case 3: // every hour for the last 7 days
        startTime = now - (7 * 24 * 60 * 60);
        intervalSeconds = 60 * 60; // 1 hour interval
        pointCount = 7 * 24;
        timeFormat = (ts: number) => {
          const date = new Date(ts * 1000);
          return `${date.getMonth() + 1}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`;
        };
        break;
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported pattern type' });
    }
    
    // 3. Get all logs of the token within the specified time range (based on proxyKey and action 1000-1099)
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
    
    // Generate time points and count usage at each time point
    const patterns: UsagePoint[] = [];
    
    for (let i = 0; i < pointCount; i++) {
      const timePoint = startTime + (i * intervalSeconds);
      const timePointEnd = timePoint + intervalSeconds;
      
      // Filter logs for this time period
      const periodLogs = logs.filter(log => {
        const logTime = Number(log.addtime);
        return logTime >= timePoint && logTime < timePointEnd;
      });
      
      // Statistics of various request types
      let requests = periodLogs.length;
      let successRequests = 0;
      let failedRequests = 0;
      let rateLimitHits = 0;
      
      periodLogs.forEach(log => {
        if (log.statusCode && log.statusCode >= 200 && log.statusCode < 300) {
          successRequests++;
        } else {
          failedRequests++;
          const isRateLimitHit = log.statusCode === 429 || (log.error && log.error.includes('rate limit'));
          if (isRateLimitHit) {
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
      totalRequests: patterns.reduce((sum, p) => sum + p.requests, 0)
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21005 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token usage pattern' });
  }
}
