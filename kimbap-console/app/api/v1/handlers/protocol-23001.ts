import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';
import {
  inferDomain,
  inferLogLevel,
  getActionLabel,
  buildDomainFilter,
  buildLevelFilter,
  parseTimeRange,
  generateLogMessage,
  generateRawData,
} from '@/lib/log-utils';

interface Request23001 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    page: number;
    pageSize: number;
    timeRange: string; // "1h", "6h", "24h", "7d", "30d", "all"
    level: string; // "all", "INFO", "WARN", "ERROR", "DEBUG"
    source: string; // "all" or domain name (e.g. "mcp-request", "lifecycle")
    search: string;
    requestId: string;
    userId: string;
  };
}

interface LogDetails {
  method: string;
  url: string;
  statusCode: number;
  responseTime: number;
  userAgent: string;
  ip: string;
  tokenId: string;
  toolName: string;
  errorType: string;
  stackTrace: string;
}

interface LogEntry {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  source: string;
  requestId: string;
  userId: string;
  rawData: string;
  details: LogDetails;
}

interface Response23001Data {
  logs: LogEntry[];
  totalCount: number;
  totalPages: number;
  availableSources: string[];
}

/**
 * Protocol 23001 - Get Logs with Filters
 *
 * Fixes applied:
 *  - Replaced inline inferSource/inferLogLevel with shared log-utils
 *  - Fixed source filter: was using `contains` on Int field (never worked);
 *    now maps domain name → action range for proper DB filtering
 *  - Used shared generateLogMessage/generateRawData
 */
export async function handleProtocol23001(body: Request23001): Promise<Response23001Data> {
  try {
    const {
      page = 1,
      pageSize = 50,
      timeRange = '24h',
      level = 'all',
      source = 'all',
      search = '',
      requestId = '',
      userId = '',
    } = body.params;

    // 1. Get proxyKey
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-23001] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-23001] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information',
      });
    }

    // 2. Build where condition using AND array (avoids OR/property conflicts)
    const startTime = parseTimeRange(timeRange);
    const andConditions: any[] = [{ proxyKey }];

    if (startTime > 0) {
      andConditions.push({ addtime: { gte: BigInt(startTime) } });
    }

    // Level filter (aligned with inferLogLevel priority)
    if (level !== 'all') {
      andConditions.push(...buildLevelFilter(level));
    }

    // Source/domain filter
    if (source !== 'all') {
      const domainFilter = buildDomainFilter(source);
      if (domainFilter) {
        andConditions.push(domainFilter);
      }
    }

    // Search filter
    if (search) {
      andConditions.push({
        OR: [
          { error: { contains: search, mode: 'insensitive' as const } },
          { ua: { contains: search, mode: 'insensitive' as const } },
          { userid: { contains: search, mode: 'insensitive' as const } },
          { sessionId: { contains: search, mode: 'insensitive' as const } },
        ],
      });
    }

    // Request ID filter
    if (requestId) {
      andConditions.push({ sessionId: { contains: requestId } });
    }

    // User ID filter
    if (userId) {
      andConditions.push({ userid: userId });
    }

    const whereCondition: any =
      andConditions.length === 1 ? andConditions[0] : { AND: andConditions };

    // 3. Query
    const totalCount = await prisma.log.count({ where: whereCondition });
    const totalPages = Math.ceil(totalCount / pageSize);
    const skip = (page - 1) * pageSize;

    const logs = await prisma.log.findMany({
      where: whereCondition,
      orderBy: { addtime: 'desc' },
      skip,
      take: pageSize,
      select: {
        id: true,
        addtime: true,
        userid: true,
        tokenMask: true,
        action: true,
        ip: true,
        ua: true,
        statusCode: true,
        error: true,
        duration: true,
        sessionId: true,
      },
    });

    // 4. Transform to response format
    const logEntries: LogEntry[] = logs.map((log) => {
      const timestamp = new Date(Number(log.addtime) * 1000)
        .toISOString()
        .replace('T', ' ')
        .slice(0, -5);
      const logLevel = inferLogLevel(log);
      const domain = inferDomain(log.action);
      const message = generateLogMessage(log);

      const details: LogDetails = {
        method: 'POST',
        url: log.action ? `/api/action/${log.action}` : '/api/unknown',
        statusCode: log.statusCode ?? 0,
        responseTime: log.duration || 0,
        userAgent: log.ua || '',
        ip: log.ip || '',
        tokenId: log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : '',
        toolName: getActionLabel(log.action),
        errorType: logLevel === 'ERROR' ? (log.error ? 'Application Error' : 'Unknown Error') : '',
        stackTrace: logLevel === 'ERROR' && log.error ? log.error : '',
      };

      return {
        id: log.id.toString(),
        timestamp,
        level: logLevel,
        message,
        source: domain,
        requestId: log.sessionId || '',
        userId: log.userid || '',
        rawData: generateRawData(log),
        details,
      };
    });

    // 5. Available sources (domains present in current data)
    const baseWhereCondition: any = { proxyKey };
    if (startTime > 0) {
      baseWhereCondition.addtime = { gte: BigInt(startTime) };
    }

    const distinctActions = await prisma.log.findMany({
      where: baseWhereCondition,
      select: { action: true },
      distinct: ['action'],
    });

    const availableSources = Array.from(
      new Set(
        distinctActions
          .filter((log) => log.action != null && typeof log.action === 'number')
          .map((log) => inferDomain(log.action)),
      ),
    );

    const response: Response23001Data = {
      logs: logEntries,
      totalCount,
      totalPages,
      availableSources,
    };

    console.log('[Protocol-23001] Response:', {
      totalLogs: logEntries.length,
      totalCount,
      totalPages,
      timeRange,
      level,
      source,
      hasSearch: !!search,
      proxyKey,
    });

    return response;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('[Protocol-23001] Error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to get log list',
    });
  }
}
