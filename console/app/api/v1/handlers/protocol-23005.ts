import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import {
  inferDomain,
  inferLogLevel,
  getActionLabel,
  generateLogMessage,
  generateRawData,
  buildDomainFilter,
  buildLevelFilter,
} from '@/lib/log-utils';
import { getProxy } from '@/lib/proxy-api';

interface Request23005 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    lastLogId: number; // Last received log ID (for incremental retrieval)
    level: string; // Log level filtering
    source: string; // Log source filtering
    limit: number; // Return quantity limit, default 50
  };
}

interface LogDetails {
  method: string; // HTTP method
  url: string; // Request URL
  statusCode: number; // status code
  responseTime: number; // Response time (milliseconds)
  userAgent: string; // user agent
  ip: string; // Client IP
  tokenId: string; // Token ID (desensitization)
  toolName: string; // Tool name
  errorType: string; // Error type
  stackTrace: string; // Stack trace (ERROR level)
}

interface LogEntry {
  id: string; // Log ID
  timestamp: string; // Timestamp (formatted)
  level: string; // Log levels: INFO, WARN, ERROR, DEBUG
  message: string; // log message
  source: string; // Log source
  requestId: string; // Request ID (optional)
  userId: string; // User ID (optional)
  rawData: string; // Raw log data
  details: LogDetails; // Details
}

interface Response23005Data {
  newLogs: LogEntry[]; // New log list
  latestLogId: number; // Latest log ID
  hasMore: boolean; // Are there more logs?
}

/**
 * Protocol 23005 - Get Real-time Logs
 * Get real-time logs (for streaming updates)
 */
export async function handleProtocol23005(body: Request23005): Promise<Response23005Data> {
  try {
    const { lastLogId = 0, level = 'all', source = 'all', limit = 50 } = body.params;

    // Get proxy key for filtering
    const proxy = await getProxy();
    const proxyKey = proxy.proxyKey;

    // Build where condition using AND array (consistent with 23001/23004)
    const andConditions: any[] = [{ id: { gt: lastLogId } }, { proxyKey }];

    // Level filter (DB-level for all levels)
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

    const whereCondition: any =
      andConditions.length === 1 ? andConditions[0] : { AND: andConditions };

    // Get new log data
    const raw = await prisma.log.findMany({
      where: whereCondition,
      orderBy: {
        id: 'asc',
      },
      take: limit + 1,
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

    const hasMore = raw.length > limit;
    const logs = hasMore ? raw.slice(0, limit) : raw;
    const latestLogId =
      logs.length > 0 ? Math.max(...logs.map((log) => log.id)) : lastLogId;

    // Convert to responsive format
    const newLogs: LogEntry[] = logs.map((log) => {
      const timestamp = new Date(Number(log.addtime) * 1000)
        .toISOString()
        .replace('T', ' ')
        .slice(0, -5);
      const logLevel = inferLogLevel(log);
      const domain = inferDomain(log.action);
      const message = generateLogMessage(log);
      const toolName = getActionLabel(log.action);

      // Build details
      const details: LogDetails = {
        method: 'POST', // Inferred from log data or default
        url: log.action ? `/api/${log.action}` : '/api/unknown',
        statusCode: log.statusCode || 0,
        responseTime: log.duration || 0,
        userAgent: log.ua || '',
        ip: log.ip || '',
        tokenId: log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : '',
        toolName,
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

    const response: Response23005Data = {
      newLogs,
      latestLogId,
      hasMore,
    };

    console.log('Protocol 23005 response:', {
      newLogsCount: newLogs.length,
      latestLogId,
      hasMore,
      lastLogId,
      filters: { level, source },
    });

    return response;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 23005 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to get real-time logs',
    });
  }
}
