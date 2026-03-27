import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getServers, getServersStatus, ServerStatus } from '@/lib/proxy-api';



interface Request20008 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    serverId: number; // ID，0
  };
}

interface ToolStatus {
  toolId: string;           // ID
  toolName: string;         // 
  status: number;           // : 0-online, 1-offline, 2-connecting, 3-error
  activeConnections: number; // 
  queuedRequests: number;   // 
  lastHeartbeat: number;    // ()
  errorMessage: string;     // （）
}

interface Response20008Data {
  toolStatus: ToolStatus[];
}

/**
 * Protocol 20008 - Get Real-time Tool Status
 * 
 */
export async function handleProtocol20008(body: Request20008): Promise<Response20008Data> {
  try {
    const { serverId } = body.params;
    const rawToken = body.common?.rawToken;

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
    
    const now = Math.floor(Date.now() / 1000);
    const oneHourAgo = now - (60 * 60); // 1
    const fiveMinutesAgo = now - (5 * 60); // 5

    let serverStatusMap: { [serverId: string]: ServerStatus } = {};
    let serversList: any[] = [];
    try {
      const [statusResult, serversResult] = await Promise.all([
        getServersStatus(body.common.userid, rawToken),
        getServers({}, body.common.userid, rawToken).catch(() => ({ servers: [] }))
      ]);
      serverStatusMap = statusResult;
      serversList = serversResult.servers || [];
    } catch (error) {
      console.warn('[Protocol-20008] Failed to get server status map:', error);
    }

    const toolNameMap = new Map<string, string>();
    serversList.forEach((server: any) => {
      if (server?.serverId) {
        toolNameMap.set(server.serverId, server.serverName || `Tool ${server.serverId}`);
      }
    });

    const discoveredToolIds = new Set<string>([
      ...Object.keys(serverStatusMap),
      ...serversList.map((server: any) => server?.serverId).filter(Boolean)
    ]);

    if (discoveredToolIds.size === 0) {
      const logRows = await prisma.log.findMany({
        where: { proxyKey, addtime: { gte: BigInt(oneHourAgo) } },
        distinct: ['serverId'],
        select: { serverId: true },
      });
      for (const row of logRows) {
        if (row.serverId) discoveredToolIds.add(row.serverId);
      }
    }

    const targetToolIds = Array.from(discoveredToolIds).filter((toolId) => {
      if (!toolId) return false;
      if (serverId > 0) return toolId === serverId.toString();
      return true;
    });
    
    // 
    const toolStatus: ToolStatus[] = await Promise.all(
      targetToolIds.map(async (toolId) => {
          
          // 
          const [
            recentActivity,
            lastActivity,
            recentErrors,
            activeSessionsCount
          ] = await Promise.all([
            // 5
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
            
            // 
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
            
            // 1
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
            
            // （5sessionId）
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
          
          const status = serverStatusMap[toolId] ?? ServerStatus.Offline;
          let errorMessage = '';

          if (status === ServerStatus.Error && lastActivity?.error) {
            errorMessage = lastActivity.error.substring(0, 100);
          } else if (recentErrors > 0 && lastActivity?.error) {
            errorMessage = lastActivity.error.substring(0, 100);
          }
          
          // （）
          const queuedRequests = 0; // 0
          
          return {
            toolId,
            toolName: toolNameMap.get(toolId) || `Tool ${toolId}`,
            status,
            activeConnections: activeSessionsCount.length,
            queuedRequests,
            lastHeartbeat: lastActivity ? Number(lastActivity.addtime) : 0,
            errorMessage
          };
        })
    );
    
    // ：error > connecting > offline > online
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
