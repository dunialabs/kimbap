import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';

interface Request21010 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;
    tokenId: string;
    eventType: number;
    page: number;
    pageSize: number;
  };
}

interface AuditLogEntry {
  logId: string;
  tokenId: string;
  eventType: number;
  eventDescription: string;
  accessorUserId: string;
  accessorRole: number;
  clientIP: string;
  userAgent: string;
  timestamp: number;
  dataScope: string;
  requestDetails: {
    statusCode: number;
    responseTime: number;
    requestSize: number;
  };
  riskLevel: number;
}

interface Response21010Data {
  auditLogs: AuditLogEntry[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
  statistics: {
    totalEvents: number;
    eventTypeDistribution: Array<{
      eventType: number;
      count: number;
      percentage: number;
    }>;
    riskDistribution: Array<{
      riskLevel: number;
      count: number;
      percentage: number;
    }>;
  };
}

type AuditRawLog = {
  id: number;
  tokenMask: string;
  userid: string;
  ip: string;
  ua: string;
  statusCode: number | null;
  duration: number | null;
  addtime: bigint;
  error: string;
  action: number;
  requestParams: string;
  responseResult: string;
};

function parseIntegerParam(value: unknown, field: string): number {
  const parsed =
    typeof value === 'number'
      ? value
      : typeof value === 'string' && value.trim() !== ''
        ? Number(value)
        : NaN;

  if (!Number.isFinite(parsed) || !Number.isInteger(parsed)) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, {
      field,
      details: `Invalid ${field}`
    });
  }

  return parsed;
}

function inferEventType(log: AuditRawLog): number {
  if ((log.statusCode || 0) >= 400 || !!log.error) {
    return 4;
  }
  if ((log.duration || 0) > 5000) {
    return 3;
  }
  return 1;
}

function calculateRiskLevel(log: AuditRawLog, inferredEventType: number): number {
  let riskScore = 0;

  const statusCode = log.statusCode || 0;
  if (statusCode >= 500) riskScore += 3;
  else if (statusCode >= 400) riskScore += 2;
  else if (statusCode >= 300) riskScore += 1;

  if (inferredEventType === 4) riskScore += 3;
  else if (inferredEventType === 2) riskScore += 1;

  const duration = log.duration || 0;
  if (duration > 10000) riskScore += 2;
  else if (duration > 5000) riskScore += 1;

  if (log.error) riskScore += 2;

  if (riskScore >= 5) return 3;
  if (riskScore >= 2) return 2;
  return 1;
}

function generateEventDescription(inferredEventType: number, log: AuditRawLog): string {
  const tokenMask = log.tokenMask ? `${log.tokenMask.substring(0, 8)}...` : 'unknown';
  if (inferredEventType === 1) {
    return `Viewed usage statistics for token ${tokenMask}`;
  }
  if (inferredEventType === 2) {
    return `Exported usage report for token ${tokenMask}`;
  }
  if (inferredEventType === 3) {
    return `Viewed details for token ${tokenMask}`;
  }
  if (inferredEventType === 4) {
    return `Anomalous access on token ${tokenMask} - status: ${log.statusCode || 0}`;
  }
  return `Accessed token ${tokenMask}`;
}

export async function handleProtocol21010(body: Request21010): Promise<Response21010Data> {
  try {
    const params = (body as { params?: Record<string, unknown> }).params || {};
    const rawTimeRange = params.timeRange;
    const rawTokenId = params.tokenId;
    const rawEventType = params.eventType;
    const rawPage = params.page;
    const rawPageSize = params.pageSize;
    const allowedTimeRanges = [1, 7, 30, 90];
    const MAX_AUDIT_SCAN_LOGS = 20000;
    const timeRange = parseIntegerParam(rawTimeRange, 'timeRange');
    const eventType = parseIntegerParam(rawEventType, 'eventType');
    const tokenId = typeof rawTokenId === 'string' ? rawTokenId : '';

    if (!allowedTimeRanges.includes(timeRange)) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported time range' });
    }

    const page = rawPage === undefined || rawPage === null || rawPage === '' ? 1 : Math.max(1, parseIntegerParam(rawPage, 'page'));
    const pageSize =
      rawPageSize === undefined || rawPageSize === null || rawPageSize === ''
        ? 20
        : Math.min(Math.max(parseIntegerParam(rawPageSize, 'pageSize'), 1), 100);

    if (eventType < 0 || eventType > 4) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Invalid eventType' });
    }

    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (timeRange * 24 * 60 * 60);
    const proxy = await getProxy();

    const whereCondition: any = {
      proxyKey: proxy.proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      tokenMask: {
        not: ''
      }
    };

    if (tokenId && tokenId.trim()) {
      whereCondition.userid = tokenId.trim();
    }

    const logs: AuditRawLog[] = await prisma.log.findMany({
      where: whereCondition,
      select: {
        id: true,
        tokenMask: true,
        userid: true,
        ip: true,
        ua: true,
        statusCode: true,
        duration: true,
        addtime: true,
        error: true,
        action: true,
        requestParams: true,
        responseResult: true
      },
      orderBy: {
        addtime: 'desc'
      },
      take: MAX_AUDIT_SCAN_LOGS + 1
    });

    if (logs.length > MAX_AUDIT_SCAN_LOGS) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Audit range is too large' });
    }

    const uniqueUserIds = Array.from(new Set(logs.map((log) => log.userid).filter(Boolean)));
    const userRoles = await prisma.user.findMany({
      where: {
        proxyKey: proxy.proxyKey,
        userid: {
          in: uniqueUserIds
        }
      },
      select: {
        userid: true,
        role: true
      }
    });
    const roleMap = new Map<string, number>(userRoles.map((user) => [user.userid, user.role]));

    const allAuditLogs: AuditLogEntry[] = logs
      .map((log) => {
        const inferredEventType = inferEventType(log);
        if (eventType > 0 && inferredEventType !== eventType) {
          return null;
        }

        const riskLevel = calculateRiskLevel(log, inferredEventType);
        const requestSize = (log.requestParams || '').length + (log.responseResult || '').length;
        const safeTokenId = log.tokenMask ? `${log.tokenMask.substring(0, 8)}...` : 'unknown';
        const safeUserId = log.userid ? `${log.userid.substring(0, 8)}...` : 'unknown';

        return {
          logId: log.id.toString(),
          tokenId: safeTokenId,
          eventType: inferredEventType,
          eventDescription: generateEventDescription(inferredEventType, log),
          accessorUserId: safeUserId,
          accessorRole: roleMap.get(log.userid) || 3,
          clientIP: log.ip ? `${log.ip.substring(0, 10)}...` : 'unknown',
          userAgent: log.ua ? `${log.ua.substring(0, 50)}${log.ua.length > 50 ? '...' : ''}` : 'unknown',
          timestamp: Number(log.addtime),
          dataScope: inferredEventType === 2 ? 'complete_export' : inferredEventType === 3 ? 'detailed_view' : 'basic_view',
          requestDetails: {
            statusCode: log.statusCode || 0,
            responseTime: log.duration || 0,
            requestSize
          },
          riskLevel
        };
      })
      .filter((log): log is AuditLogEntry => !!log);

    const totalEvents = allAuditLogs.length;
    const eventTypeMap = new Map<number, number>();
    const riskLevelMap = new Map<number, number>();

    allAuditLogs.forEach((log) => {
      eventTypeMap.set(log.eventType, (eventTypeMap.get(log.eventType) || 0) + 1);
      riskLevelMap.set(log.riskLevel, (riskLevelMap.get(log.riskLevel) || 0) + 1);
    });

    const eventTypeDistribution = Array.from(eventTypeMap.entries()).map(([type, count]) => ({
      eventType: type,
      count,
      percentage: totalEvents > 0 ? Math.round((count / totalEvents) * 1000) / 10 : 0
    }));

    const riskDistribution = Array.from(riskLevelMap.entries()).map(([level, count]) => ({
      riskLevel: level,
      count,
      percentage: totalEvents > 0 ? Math.round((count / totalEvents) * 1000) / 10 : 0
    }));

    const total = allAuditLogs.length;
    const totalPages = Math.ceil(total / pageSize);
    const startIndex = (page - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    const paginatedLogs = allAuditLogs.slice(startIndex, endIndex);

    const response: Response21010Data = {
      auditLogs: paginatedLogs,
      pagination: {
        page,
        pageSize,
        total,
        totalPages
      },
      statistics: {
        totalEvents,
        eventTypeDistribution,
        riskDistribution
      }
    };

    console.log('Protocol 21010 response:', {
      totalEvents,
      paginatedEvents: paginatedLogs.length,
      eventTypes: eventTypeDistribution.length,
      riskLevels: riskDistribution.length,
      timeRange
    });

    return response;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('Protocol 21010 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token audit logs' });
  }
}
