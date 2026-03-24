import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request20004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;  // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    serverId: number;   // 服务器ID，0表示所有服务器
    toolId: string;     // 特定工具ID，空表示所有工具
  };
}

interface ErrorType {
  errorCode: string;      // 错误代码
  errorMessage: string;   // 错误描述
  count: number;          // 发生次数
  percentage: number;     // 占比(%)
  lastOccurred: number;   // 最后发生时间(时间戳)
}

interface ToolErrors {
  toolId: string;         // 工具ID
  toolName: string;       // 工具名称
  totalErrors: number;    // 总错误数
  errorTypes: ErrorType[]; // 错误类型分布
}

interface Response20004Data {
  toolErrors: ToolErrors[];
}

/**
 * Protocol 20004 - Get Tool Error Analysis
 * 获取工具错误分析数据
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
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 构建where条件 - 只查询失败的请求
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
    
    // 如果指定了serverId，添加过滤条件
    if (serverId > 0) {
      whereCondition.serverId = serverId.toString();
    }
    
    // 如果指定了toolId，添加过滤条件
    if (toolId && toolId.trim()) {
      whereCondition.serverId = toolId.trim();
    }
    
    // 获取所有错误日志
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
    
    // 按工具ID分组
    const toolErrorsMap = new Map<string, any>();
    
    errorLogs.forEach(log => {
      const toolId = log.serverId!;
      
      if (!toolErrorsMap.has(toolId)) {
        toolErrorsMap.set(toolId, {
          toolId,
          toolName: `Tool ${toolId}`,
          totalErrors: 0,
          errorTypesMap: new Map()
        });
      }
      
      const toolError = toolErrorsMap.get(toolId);
      toolError.totalErrors++;
      
      // 分析错误类型
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
        // 尝试从错误信息中提取更具体的错误类型
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
          errorMessage = log.error.substring(0, 50); // 截取前50个字符
        }
      }
      
      // 统计错误类型
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
    
    // 转换为最终数据格式
    const toolErrors: ToolErrors[] = Array.from(toolErrorsMap.values()).map(toolError => {
      const errorTypes: ErrorType[] = Array.from(toolError.errorTypesMap.values())
        .map((errorType: any) => ({
          errorCode: errorType.errorCode,
          errorMessage: errorType.errorMessage,
          count: errorType.count,
          percentage: Math.round((errorType.count / toolError.totalErrors) * 1000) / 10, // 保留1位小数
          lastOccurred: errorType.lastOccurred
        }))
        .sort((a, b) => b.count - a.count); // 按出现次数降序排列
      
      return {
        toolId: toolError.toolId,
        toolName: toolError.toolName,
        totalErrors: toolError.totalErrors,
        errorTypes
      };
    }).sort((a, b) => b.totalErrors - a.totalErrors); // 按总错误数降序排列
    
    const response: Response20004Data = {
      toolErrors
    };
    
    console.log('Protocol 20004 response:', {
      toolsWithErrors: toolErrors.length,
      totalErrorLogs: errorLogs.length
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 20004 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool error analysis' });
  }
}
