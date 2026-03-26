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
    limit: number; // Return quantity limit, default 10
    timeRange?: number;
  };
}

interface ActivityEvent {
  eventType: string;   // Event types: "tool_request", "token_auth", "rate_limit", "error"
  description: string; // event description
  details: string;     // Details (such as response time, token, etc.)
  timestamp: number;   // Timestamp (front-end expected field)
  icon: string;        // icon (frontend expected field)
  color: string;       // Display color (for UI status points)
}

interface Response22004Data {
  activities: ActivityEvent[];  // Changed from recentEvents to activities to match frontend expectation
}

/**
 * Protocol 22004 - Get Recent Activity
 * Get recent activities
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
    
    // Query recent log records
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
      take: limit * 2 // Take more, then filter to generate diverse events
    });
    
    // Auxiliary function: determine event type
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
    
    // Helper function: generate event description
    const generateEventDescription = (log: typeof recentLogs[0], eventType: string): string => {
      const actionLabel = getActionLabel(log.action);
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
            return `${actionLabel} request completed`;
          }
        case 'token_auth':
          return 'Token authentication successful';
        case 'rate_limit':
          return 'Rate limit warning';
        case 'error':
          return `Request failed - ${log.error && log.error.trim() ? log.error.substring(0, 60) : `HTTP ${log.statusCode ?? 'unknown'}`}`;
        case 'system':
          return `${actionLabel} event recorded`;
        default:
          return 'System activity';
      }
    };
    
    // Helper function: generate details
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
    
    // Auxiliary function: get event color
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
    
    // Helper function: Get event icon
    const getEventIcon = (eventType: string): string => {
      switch (eventType) {
        case 'tool_request':
          return '⚡'; // tool request
        case 'token_auth':
          return '🔑';  // Authentication event
        case 'rate_limit':
          return '⚠️'; // warn
        case 'error':
          return '❌';   // mistake
        case 'system':
          return 'ℹ️';
        default:
          return '📄';  // default
      }
    };
    
    // Generate activity events
    const recentEvents: ActivityEvent[] = [];
    const eventTypeCounter = new Map<string, number>();
    
    for (const log of recentLogs) {
      if (recentEvents.length >= limit) break;
      
      const eventType = determineEventType(log);
      const eventCount = eventTypeCounter.get(eventType) || 0;
      
      // Limit the number of events of the same type to ensure diversity
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
      activities: recentEvents
    };
    
    console.log('Protocol 22004 response:', {
      eventsCount: response.activities.length,
      eventTypes: Array.from(new Set(response.activities.map(e => e.eventType))),
      oldestEventMinutes: response.activities.length > 0 ? 
        Math.max(...response.activities.map(e => Math.floor((now - e.timestamp) / 60))) : 0
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 22004 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get recent activities' });
  }
}
