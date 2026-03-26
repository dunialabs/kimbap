import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers, getUsers } from '@/lib/proxy-api';
import { getActionMachineName, TOOL_USAGE_ACTION_RANGE } from '@/lib/log-utils';



interface Request20010 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;       // Time range: 1-today, 7-last 7 days, 30-last 30 days
    toolId?: string;         // Tool ID, empty means all tools
    toolIds?: string[];
    actionTypes?: number[];  // MCPEventLogType enumeration value list, empty means all types
    status?: number;
    page?: number;           // Pagination - page number
    pageSize?: number;       // Pagination - number per page
  };
}

interface ActionLog {
  logId: string;           // Log ID
  actionType: number;      // MCPEventLogType enumeration value
  actionName: string;      // Operation name (such as: "RequestTool", "ResponseTool", etc.)
  toolId: string;          // Tool ID
  toolName: string;        // Tool name
  userId: string;          // User ID
  userName: string;        // Username
  timestamp: number;       // Timestamp
  responseTime: number;    // Response time(ms)
  status: number;          // Status: 1-success, 2-failed
  errorMessage: string;    // Error message (if failure)
  details: string;         // Details (JSON format)
}

interface Response20010Data {
  logs: ActionLog[];
  totalCount: number; // Total quantity (for pagination)
}

/**
 * Protocol 20010 - Get Tool Action Logs
 * Get tool operation log (based on MCPEventLogType)
 */
export async function handleProtocol20010(body: Request20010): Promise<Response20010Data> {
  try {
    const rawToken = body.common?.rawToken;
    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : 1;

    const parsedPage = Number(body.params?.page);
    const normalizedPage = Math.floor(parsedPage);
    const page = Number.isFinite(normalizedPage) && normalizedPage >= 1 ? normalizedPage : 1;

    const parsedPageSize = Number(body.params?.pageSize);
    const normalizedPageSize = Math.floor(parsedPageSize);
    const pageSize = Number.isFinite(normalizedPageSize) && normalizedPageSize >= 1 ? normalizedPageSize : 50;

    const toolId = typeof body.params?.toolId === 'string' ? body.params.toolId.trim() : '';
    const toolIds = Array.isArray(body.params?.toolIds)
      ? body.params.toolIds.map((id) => (typeof id === 'string' ? id.trim() : '')).filter((id) => id.length > 0)
      : [];
    const actionTypes = Array.isArray(body.params?.actionTypes)
      ? body.params.actionTypes.filter((actionType): actionType is number => Number.isInteger(actionType))
      : [];
    const parsedStatus = Number(body.params?.status);
    const status = parsedStatus === 1 || parsedStatus === 2 ? parsedStatus : undefined;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-20010] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    // Calculation time range
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // Build where condition
    const whereCondition: any = {
      proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      serverId: {
        not: null
      }
    };
    
    // If toolId is specified, add filter conditions
    if (toolIds.length > 0) {
      whereCondition.serverId = { in: toolIds };
    } else if (toolId) {
      whereCondition.serverId = toolId;
    }
    
    // If actionTypes is specified, add filter conditions
    if (actionTypes && actionTypes.length > 0) {
      whereCondition.action = {
        in: actionTypes
      };
    } else {
      whereCondition.action = {
        gte: TOOL_USAGE_ACTION_RANGE.gte,
        lte: TOOL_USAGE_ACTION_RANGE.lte
      };
    }

    if (status === 1) {
      whereCondition.AND = [
        {
          OR: [
            { error: '' },
            { error: null }
          ]
        },
        {
          OR: [
            { statusCode: null },
            {
              statusCode: {
                gte: 200,
                lt: 300
              }
            }
          ]
        }
      ];
    }

    if (status === 2) {
      whereCondition.OR = [
        {
          error: {
            not: ''
          }
        },
        {
          statusCode: {
            lt: 200
          }
        },
        {
          statusCode: {
            gte: 300
          }
        }
      ];
    }
    
    // 1. First query the total quantity
    const totalCount = await prisma.log.count({
      where: whereCondition
    });

    let usersMap: Record<string, string> = {};
    let serversMap: Record<string, string> = {};
    try {
      const [usersResult, serversResult] = await Promise.all([
        getUsers({}, body.common.userid, rawToken),
        getServers({}, body.common.userid, rawToken)
      ]);

      const usersList = usersResult.users || [];
      usersList.forEach((user: any) => {
        if (!user?.userId) return;
        const displayName = user.userName && user.userName !== user.name
          ? `${user.name}(${user.userName})`
          : (user.name || user.userId);
        usersMap[user.userId] = displayName;
      });

      const serversList = serversResult.servers || [];
      serversList.forEach((server: any) => {
        if (!server?.serverId) return;
        serversMap[server.serverId] = server.serverName || server.serverId;
      });
    } catch (error) {
      console.warn('[Protocol-20010] Failed to enrich user/tool names:', error);
    }
    
    // 2. Query log data by page
    const offset = (page - 1) * pageSize;
    const logData = await prisma.log.findMany({
      where: whereCondition,
      select: {
        id: true,
        addtime: true,
        action: true,
        userid: true,
        serverId: true,
        duration: true,
        statusCode: true,
        error: true,
        requestParams: true,
        responseResult: true,
        sessionId: true
      },
      orderBy: [
        { addtime: 'desc' },
        { id: 'desc' }
      ],
      skip: offset,
      take: pageSize
    });
    
    // 3. Convert to ActionLog format
    const logs: ActionLog[] = logData.map(log => {
      // Determine success/failure status
      let status = 1; // success
      let errorMessage = '';
      
      if (log.error && log.error.trim()) {
        status = 2; // failed
        errorMessage = log.error.substring(0, 200); // Limit error message length
      } else if (log.statusCode && (log.statusCode < 200 || log.statusCode >= 300)) {
        status = 2; // failed
        errorMessage = `HTTP ${log.statusCode} error`;
      }
      
      // Build details
      const details = {
        sessionId: log.sessionId,
        statusCode: log.statusCode,
        requestParams: log.requestParams ? (log.requestParams.length > 500 ? log.requestParams.substring(0, 500) + '...' : log.requestParams) : '',
        responseResult: log.responseResult ? (log.responseResult.length > 500 ? log.responseResult.substring(0, 500) + '...' : log.responseResult) : ''
      };
      
      return {
        logId: log.id.toString(),
        actionType: log.action,
        actionName: getActionMachineName(log.action),
        toolId: log.serverId || '',
        toolName: log.serverId ? (serversMap[log.serverId] || `${log.serverId} (Unavailable)`) : 'Unknown Tool',
        userId: log.userid,
        userName: usersMap[log.userid] || `${log.userid} (Unavailable)`,
        timestamp: Number(log.addtime),
        responseTime: log.duration || 0,
        status,
        errorMessage,
        details: JSON.stringify(details)
      };
    });
    
    const response: Response20010Data = {
      logs,
      totalCount
    };
    
    console.log('Protocol 20010 response:', {
      logsCount: logs.length,
      totalCount,
      page,
      pageSize,
      timeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20010 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool operation logs' });
  }
}
