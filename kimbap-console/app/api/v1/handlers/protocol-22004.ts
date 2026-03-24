import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';
import { getActionLabel, inferDomain } from '@/lib/log-utils';



interface Request22004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    limit: number; // 返回数量限制，默认10
    timeRange?: number;
  };
}

interface ActivityEvent {
  eventType: string;   // 事件类型: "tool_request", "token_auth", "rate_limit", "error"
  description: string; // 事件描述
  details: string;     // 详细信息（如响应时间、令牌等）
  timestamp: number;   // 时间戳 (前端期望字段)
  icon: string;        // 图标 (前端期望字段)
  color: string;       // 显示颜色（用于UI状态点）
}

interface Response22004Data {
  activities: ActivityEvent[];  // Changed from recentEvents to activities to match frontend expectation
}

/**
 * Protocol 22004 - Get Recent Activity
 * 获取最近活动
 */
export async function handleProtocol22004(body: Request22004): Promise<Response22004Data> {
  try {
    const parsedLimit = Number(body.params?.limit);
    const normalizedLimit = Math.floor(parsedLimit);
    const limit = Number.isFinite(normalizedLimit) && normalizedLimit >= 1 ? normalizedLimit : 10;
    const parsedTimeRange = Number(body.params?.timeRange);
    const normalizedTimeRange = Math.floor(parsedTimeRange);
    const timeRange = Number.isFinite(normalizedTimeRange) && normalizedTimeRange >= 1 ? normalizedTimeRange : 1;
    let proxyKey = '';

    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('Protocol 22004 failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    const now = Math.floor(Date.now() / 1000);
    
    // 查询最近的日志记录
    const recentLogs = await prisma.log.findMany({
      where: {
        proxyKey,
        addtime: {
          gte: BigInt(now - (timeRange * 24 * 60 * 60))
        }
      },
      select: {
        id: true,
        addtime: true,
        action: true,
        tokenMask: true,
        statusCode: true,
        duration: true,
        error: true,
        ip: true
      },
      orderBy: {
        addtime: 'desc'
      },
      take: limit * 2 // 取多一些，然后过滤生成多样化的事件
    });
    
    // 辅助函数：判断事件类型
    const determineEventType = (log: typeof recentLogs[0]): string => {
      if (log.error && log.error.trim() !== '') {
        return 'error';
      } else if (log.statusCode === 429) {
        return 'rate_limit';
      } else if ((log.statusCode ?? 0) >= 400) {
        return 'error';
      }

      const domain = inferDomain(log.action);
      if (domain === 'mcp-request') {
        return 'tool_request';
      }
      if (domain === 'auth' || domain === 'oauth' || (log.tokenMask && log.tokenMask.trim() !== '')) {
        return 'token_auth';
      }
      return 'system';
    };
    
    // 辅助函数：生成事件描述
    const generateEventDescription = (log: typeof recentLogs[0], eventType: string): string => {
      const actionLabel = getActionLabel(log.action);
      const formattedAction = actionLabel.includes('_') 
        ? actionLabel.split('_').map(word => 
            word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()
          ).join(' ')
        : actionLabel;
      const lowercaseActionLabel = actionLabel.toLowerCase();
      
      switch (eventType) {
        case 'tool_request':
          if (lowercaseActionLabel.includes('web')) {
            return 'Web Server MCP request completed';
          } else if (lowercaseActionLabel.includes('notion')) {
            return 'New Notion page created';
          } else if (lowercaseActionLabel.includes('github')) {
            return 'GitHub repository accessed';
          } else if (lowercaseActionLabel.includes('postgres') || lowercaseActionLabel.includes('sql')) {
            return 'PostgreSQL query executed';
          } else {
            return `${formattedAction} request completed`;
          }
        case 'token_auth':
          return 'Token authentication successful';
        case 'rate_limit':
          return 'Rate limit warning';
        case 'error':
          return `Request failed - ${log.error && log.error.trim() ? log.error.substring(0, 60) : `HTTP ${log.statusCode ?? 'unknown'}`}`;
        case 'system':
          return `${formattedAction} event recorded`;
        default:
          return 'System activity';
      }
    };
    
    // 辅助函数：生成详细信息
    const generateEventDetails = (log: typeof recentLogs[0], eventType: string): string => {
      const minutesAgo = Math.floor((now - Number(log.addtime)) / 60);
      const timeStr = minutesAgo < 60 ? `${minutesAgo} minutes ago` : 
                      `${Math.floor(minutesAgo / 60)} hours ago`;
      
      switch (eventType) {
        case 'tool_request':
          {
            const responseTime = log.duration || 0;
            return `${timeStr} • ${responseTime}ms response`;
          }
        case 'token_auth':
          {
            const tokenId = log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : 'unknown';
            return `${timeStr} • ${tokenId}`;
          }
        case 'rate_limit':
          {
            const rateLimitTokenId = log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : 'unknown';
            return `${timeStr} • ${rateLimitTokenId}`;
          }
        case 'error':
          {
            const errorDetails = log.error ? log.error.substring(0, 50) + '...' : 'Unknown error';
            return `${timeStr} • ${errorDetails}`;
          }
        case 'system':
          return `${timeStr} • system event`;
        default:
          return timeStr;
      }
    };
    
    // 辅助函数：获取事件颜色
    const getEventColor = (eventType: string): string => {
      switch (eventType) {
        case 'tool_request':
          return '#10b981'; // green-500
        case 'token_auth':
          return '#3b82f6';  // blue-500
        case 'rate_limit':
          return '#f59e0b'; // amber-500
        case 'error':
          return '#ef4444';   // red-500
        case 'system':
          return '#6b7280';
        default:
          return '#6b7280';  // gray-500
      }
    };
    
    // 辅助函数：获取事件图标
    const getEventIcon = (eventType: string): string => {
      switch (eventType) {
        case 'tool_request':
          return '⚡'; // 工具请求
        case 'token_auth':
          return '🔑';  // 认证事件
        case 'rate_limit':
          return '⚠️'; // 警告
        case 'error':
          return '❌';   // 错误
        case 'system':
          return 'ℹ️';
        default:
          return '📄';  // 默认
      }
    };
    
    // 生成活动事件
    const recentEvents: ActivityEvent[] = [];
    const eventTypeCounter = new Map<string, number>();
    
    for (const log of recentLogs) {
      if (recentEvents.length >= limit) break;
      
      const eventType = determineEventType(log);
      const eventCount = eventTypeCounter.get(eventType) || 0;
      
      // 限制同类型事件数量，保证多样性
      if (eventCount >= Math.ceil(limit / 3)) continue;
      
      const minutesAgo = Math.floor((now - Number(log.addtime)) / 60);
      
      if (minutesAgo > (timeRange * 24 * 60)) continue;
      
      const event: ActivityEvent = {
        eventType,
        description: generateEventDescription(log, eventType),
        details: generateEventDetails(log, eventType),
        timestamp: Number(log.addtime), // Convert BigInt to number
        icon: getEventIcon(eventType),
        color: getEventColor(eventType)
      };
      
      recentEvents.push(event);
      eventTypeCounter.set(eventType, eventCount + 1);
    }
    
    const response: Response22004Data = {
      activities: recentEvents.slice(0, limit)  // Changed from recentEvents to activities
    };
    
    console.log('Protocol 22004 response:', {
      eventsCount: response.activities.length,
      eventTypes: Array.from(new Set(response.activities.map(e => e.eventType))),
      oldestEventMinutes: response.activities.length > 0 ? 
        Math.max(...response.activities.map(e => Math.floor((now - e.timestamp) / 60))) : 0
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 22004 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get recent activities' });
  }
}
