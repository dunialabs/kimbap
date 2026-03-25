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
    timeRange: number;       // 时间范围: 1-今天, 7-最近7天, 30-最近30天
    toolId?: string;         // 工具ID，空表示所有工具
    toolIds?: string[];
    actionTypes?: number[];  // MCPEventLogType枚举值列表，空表示所有类型
    status?: number;
    page?: number;           // 分页-页码
    pageSize?: number;       // 分页-每页数量
  };
}

interface ActionLog {
  logId: string;           // 日志ID
  actionType: number;      // MCPEventLogType枚举值
  actionName: string;      // 操作名称（如: "RequestTool", "ResponseTool"等）
  toolId: string;          // 工具ID
  toolName: string;        // 工具名称
  userId: string;          // 用户ID
  userName: string;        // 用户名称
  timestamp: number;       // 时间戳
  responseTime: number;    // 响应时间(ms)
  status: number;          // 状态: 1-success, 2-failed
  errorMessage: string;    // 错误信息（如果失败）
  details: string;         // 详细信息（JSON格式）
}

interface Response20010Data {
  logs: ActionLog[];
  totalCount: number; // 总数量（用于分页）
}

/**
 * Protocol 20010 - Get Tool Action Logs
 * 获取工具操作日志（基于MCPEventLogType）
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
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 构建where条件
    const whereCondition: any = {
      proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      serverId: {
        not: null
      }
    };
    
    // 如果指定了toolId，添加过滤条件
    if (toolIds.length > 0) {
      whereCondition.serverId = { in: toolIds };
    } else if (toolId) {
      whereCondition.serverId = toolId;
    }
    
    // 如果指定了actionTypes，添加过滤条件
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
    
    // 1. 先查询总数量
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
    
    // 2. 分页查询日志数据
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
    
    // 3. 转换为ActionLog格式
    const logs: ActionLog[] = logData.map(log => {
      // 判断成功/失败状态
      let status = 1; // success
      let errorMessage = '';
      
      if (log.error && log.error.trim()) {
        status = 2; // failed
        errorMessage = log.error.substring(0, 200); // 限制错误信息长度
      } else if (log.statusCode && (log.statusCode < 200 || log.statusCode >= 300)) {
        status = 2; // failed
        errorMessage = `HTTP ${log.statusCode} error`;
      }
      
      // 构建详细信息
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
