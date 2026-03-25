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
    lastLogId: number; // 最后接收到的日志ID（用于增量获取）
    level: string; // 日志级别过滤
    source: string; // 日志来源过滤
    limit: number; // 返回数量限制，默认50
  };
}

interface LogDetails {
  method: string; // HTTP方法
  url: string; // 请求URL
  statusCode: number; // 状态码
  responseTime: number; // 响应时间(毫秒)
  userAgent: string; // 用户代理
  ip: string; // 客户端IP
  tokenId: string; // 令牌ID（脱敏）
  toolName: string; // 工具名称
  errorType: string; // 错误类型
  stackTrace: string; // 堆栈跟踪（ERROR级别）
}

interface LogEntry {
  id: string; // 日志ID
  timestamp: string; // 时间戳（格式化）
  level: string; // 日志级别: INFO, WARN, ERROR, DEBUG
  message: string; // 日志消息
  source: string; // 日志来源
  requestId: string; // 请求ID（可选）
  userId: string; // 用户ID（可选）
  rawData: string; // 原始日志数据
  details: LogDetails; // 详细信息
}

interface Response23005Data {
  newLogs: LogEntry[]; // 新日志列表
  latestLogId: number; // 最新日志ID
  hasMore: boolean; // 是否还有更多日志
}

/**
 * Protocol 23005 - Get Real-time Logs
 * 获取实时日志（用于流式更新）
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

    // 获取新日志数据
    const logs = await prisma.log.findMany({
      where: whereCondition,
      orderBy: {
        id: 'asc', // 按ID升序，确保按时间顺序获取
      },
      take: limit,
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


    // 检查是否还有更多日志
    const hasMore = logs.length === limit;
    const latestLogId =
      logs.length > 0 ? Math.max(...logs.map((log) => log.id)) : lastLogId;

    // 转换为响应格式
    const newLogs: LogEntry[] = logs.map((log) => {
      const timestamp = new Date(Number(log.addtime) * 1000)
        .toISOString()
        .replace('T', ' ')
        .slice(0, -5);
      const logLevel = inferLogLevel(log);
      const domain = inferDomain(log.action);
      const message = generateLogMessage(log);
      const toolName = getActionLabel(log.action);

      // 构建详细信息
      const details: LogDetails = {
        method: 'POST', // 从log数据推断或默认
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
