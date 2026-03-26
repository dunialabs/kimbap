import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request20004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;  // Time range: 1-today, 7-last 7 days, 30-last 30 days, 90-last 90 days
    serverId: number;   // Server ID, 0 means all servers
    toolId: string;     // Specific tool ID, empty means all tools
  };
}

interface ErrorType {
  errorCode: string;      // error code
  errorMessage: string;   // Error description
  count: number;          // Number of occurrences
  percentage: number;     // Proportion (%)
  lastOccurred: number;   // Last occurrence time (timestamp)
}

interface ToolErrors {
  toolId: string;         // Tool ID
  toolName: string;       // Tool name
  totalErrors: number;    // total errors
  errorTypes: ErrorType[]; // Error type distribution
}

interface Response20004Data {
  toolErrors: ToolErrors[];
}

/**
 * Protocol 20004 - Get Tool Error Analysis
 * Get tool error analysis data
 */
export async function handleProtocol20004(body: Request20004): Promise<Response20004Data> {
  try {
    const { timeRange, serverId, toolId } = body.params;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-20004] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    // Calculation time range
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // Build a where condition - only query failed requests
    const whereCondition: any = {
      proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      action: {
        in: [1001, 1002, 1003, 1004, 1005, 1006]
      },
      serverId: {
        not: null
      },
      OR: [
        {
          statusCode: {
            lt: 200
          }
        },
        {
          statusCode: {
            gte: 300
          }
        },
        {
          error: {
            not: ''
          }
        }
      ]
    };
    
    // If serverId is specified, add filter conditions
    if (serverId > 0) {
      whereCondition.serverId = serverId.toString();
    }
    
    // If toolId is specified, add filter conditions
    if (toolId && toolId.trim()) {
      whereCondition.serverId = toolId.trim();
    }
    
    // Get all error logs
    const errorLogs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        serverId: true,
        statusCode: true,
        error: true,
        addtime: true
      },
      orderBy: {
        addtime: 'desc'
      }
    });
    
    // Group by tool ID
    const toolErrorsMap = new Map<string, any>();
    
    errorLogs.forEach(log => {
      const logServerId = log.serverId!;
      
      if (!toolErrorsMap.has(logServerId)) {
        toolErrorsMap.set(logServerId, {
          toolId: logServerId,
          toolName: `Tool ${logServerId}`,
          totalErrors: 0,
          errorTypesMap: new Map()
        });
      }
      
      const toolError = toolErrorsMap.get(logServerId);
      toolError.totalErrors++;
      
      // Analyze error types
      let errorCode = 'UNKNOWN_ERROR';
      let errorMessage = 'Unknown error';
      
      if (log.statusCode) {
        errorCode = `HTTP_${log.statusCode}`;
        switch (Math.floor(log.statusCode / 100)) {
          case 4:
            errorMessage = 'Client error';
            break;
          case 5:
            errorMessage = 'Server error';
            break;
          default:
            errorMessage = `HTTP ${log.statusCode} error`;
        }
      }
      
      if (log.error && log.error.trim()) {
        // Try to extract a more specific error type from the error message
        const errorText = log.error.toLowerCase();
        if (errorText.includes('timeout')) {
          errorCode = 'REQUEST_TIMEOUT';
          errorMessage = 'Request timeout';
        } else if (errorText.includes('connection')) {
          errorCode = 'CONNECTION_ERROR';
          errorMessage = 'Connection error';
        } else if (errorText.includes('permission') || errorText.includes('denied')) {
          errorCode = 'PERMISSION_DENIED';
          errorMessage = 'Permission denied';
        } else if (errorText.includes('not found')) {
          errorCode = 'NOT_FOUND';
          errorMessage = 'Resource not found';
        } else if (errorText.includes('rate limit')) {
          errorCode = 'RATE_LIMIT_EXCEEDED';
          errorMessage = 'Rate limit exceeded';
        } else {
          errorCode = 'APPLICATION_ERROR';
          errorMessage = log.error.substring(0, 50); // Truncate the first 50 characters
        }
      }
      
      // Statistical error types
      const errorKey = `${errorCode}:${errorMessage}`;
      if (!toolError.errorTypesMap.has(errorKey)) {
        toolError.errorTypesMap.set(errorKey, {
          errorCode,
          errorMessage,
          count: 0,
          lastOccurred: 0
        });
      }
      
      const errorType = toolError.errorTypesMap.get(errorKey);
      errorType.count++;
      errorType.lastOccurred = Math.max(errorType.lastOccurred, Number(log.addtime));
    });
    
    // Convert to final data format
    const toolErrors: ToolErrors[] = Array.from(toolErrorsMap.values()).map(toolError => {
      const errorTypes: ErrorType[] = Array.from(toolError.errorTypesMap.values())
        .map((errorType: any) => ({
          errorCode: errorType.errorCode,
          errorMessage: errorType.errorMessage,
          count: errorType.count,
          percentage: Math.round((errorType.count / toolError.totalErrors) * 1000) / 10, // Keep 1 decimal place
          lastOccurred: errorType.lastOccurred
        }))
        .sort((a, b) => b.count - a.count); // Sort by occurrence in descending order
      
      return {
        toolId: toolError.toolId,
        toolName: toolError.toolName,
        totalErrors: toolError.totalErrors,
        errorTypes
      };
    }).sort((a, b) => b.totalErrors - a.totalErrors); // Sort by total errors in descending order
    
    const response: Response20004Data = {
      toolErrors
    };
    
    console.log('Protocol 20004 response:', {
      toolsWithErrors: toolErrors.length,
      totalErrorLogs: errorLogs.length
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20004 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool error analysis' });
  }
}
