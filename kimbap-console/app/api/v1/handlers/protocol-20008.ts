import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request20008 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId: number; // 服务器ID，0表示所有服务器
  };
}

interface ToolStatus {
  toolId: string;           // 工具ID
  toolName: string;         // 工具名称
  status: number;           // 状态: 0-online, 1-offline, 2-connecting, 3-error
  activeConnections: number; // 当前活跃连接数
  queuedRequests: number;   // 排队请求数
  lastHeartbeat: number;    // 最后心跳时间(时间戳)
  errorMessage: string;     // 错误信息（如果有）
}

interface Response20008Data {
  toolStatus: ToolStatus[];
}

/**
 * Protocol 20008 - Get Real-time Tool Status
 * 获取工具实时状态
 */
export async function handleProtocol20008(body: Request20008): Promise<Response20008Data> {
  try {
    const { serverId } = body.params;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-20008] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    // 构建where条件
    const whereCondition: any = {
      proxyKey,
      serverId: {
        not: null
      }
    };
    
    // 如果指定了serverId，添加过滤条件
    if (serverId > 0) {
      whereCondition.serverId = serverId.toString();
    }
    
    // 获取所有工具ID（从历史日志中）
    const allTools = await prisma.log.findMany({
      where: whereCondition,
      select: {
        serverId: true
      },
      distinct: ['serverId']
    });
    
    const now = Math.floor(Date.now() / 1000);
    const oneHourAgo = now - (60 * 60); // 1小时前
    const fiveMinutesAgo = now - (5 * 60); // 5分钟前
    
    // 为每个工具查询状态信息
    const toolStatus: ToolStatus[] = await Promise.all(
      allTools
        .filter(tool => tool.serverId)
        .map(async (tool) => {
          const toolId = tool.serverId!;
          
          // 并行查询该工具的状态指标
          const [
            recentActivity,
            lastActivity,
            recentErrors,
            activeSessionsCount
          ] = await Promise.all([
            // 最近5分钟的活动
            prisma.log.count({
              where: {
                proxyKey,
                serverId: toolId,
                addtime: {
                  gte: BigInt(fiveMinutesAgo)
                },
                action: {
                  in: [1001, 1002, 1003, 1004, 1005, 1006]
                }
              }
            }),
            
            // 最后一次活动
            prisma.log.findFirst({
              where: {
                proxyKey,
                serverId: toolId,
                action: {
                  in: [1001, 1002, 1003, 1004, 1005, 1006]
                }
              },
              orderBy: {
                addtime: 'desc'
              },
              select: {
                addtime: true,
                error: true,
                statusCode: true
              }
            }),
            
            // 最近1小时的错误
            prisma.log.count({
              where: {
                proxyKey,
                serverId: toolId,
                addtime: {
                  gte: BigInt(oneHourAgo)
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
              }
            }),
            
            // 活跃会话数（基于最近5分钟的不同sessionId）
            prisma.log.findMany({
              where: {
                proxyKey,
                serverId: toolId,
                addtime: {
                  gte: BigInt(fiveMinutesAgo)
                },
                sessionId: {
                  not: ''
                }
              },
              select: {
                sessionId: true
              },
              distinct: ['sessionId']
            })
          ]);
          
          // 判断工具状态
          let status = 1; // offline (默认)
          let errorMessage = '';
          
          if (lastActivity) {
            const lastActivityTime = Number(lastActivity.addtime);

            if (lastActivityTime > fiveMinutesAgo) {
              if (recentErrors > 0) {
                const recentSuccesses = await prisma.log.count({
                  where: {
                    proxyKey,
                    serverId: toolId,
                    addtime: { gte: BigInt(oneHourAgo) },
                  error: '',
                  OR: [{ statusCode: null }, { statusCode: { gte: 200, lt: 400 } }],
                  },
                });
                if (recentSuccesses === 0) {
                  status = 3;
                  errorMessage = (lastActivity.error ?? '').substring(0, 100);
                } else {
                  status = 0;
                }
              } else {
                status = 0;
              }
            } else if (lastActivityTime > oneHourAgo) {
              status = 2;
            } else {
              status = 1;
            }
          }
          
          // 模拟排队请求数（实际可能需要从其他数据源获取）
          const queuedRequests = 0; // 暂时设为0
          
          return {
            toolId,
            toolName: `Tool ${toolId}`,
            status,
            activeConnections: activeSessionsCount.length,
            queuedRequests,
            lastHeartbeat: lastActivity ? Number(lastActivity.addtime) : 0,
            errorMessage
          };
        })
    );
    
    // 按状态排序：error > connecting > offline > online
    const statusPriority = { 3: 0, 2: 1, 1: 2, 0: 3 };
    toolStatus.sort((a, b) => {
      const priorityA = statusPriority[a.status as keyof typeof statusPriority] || 4;
      const priorityB = statusPriority[b.status as keyof typeof statusPriority] || 4;
      return priorityA - priorityB;
    });
    
    const response: Response20008Data = {
      toolStatus
    };
    
    console.log('Protocol 20008 response:', {
      toolsCount: toolStatus.length,
      onlineTools: toolStatus.filter(t => t.status === 0).length,
      offlineTools: toolStatus.filter(t => t.status === 1).length,
      errorTools: toolStatus.filter(t => t.status === 3).length
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20008 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get tool real-time status' });
  }
}
