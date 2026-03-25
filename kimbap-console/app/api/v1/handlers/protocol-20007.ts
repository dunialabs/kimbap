import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, getUsers } from '@/lib/proxy-api';



interface Request20007 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    timeRange: number;
    userId: string;
    page: number;
    pageSize: number;
  };
}

interface ToolUsage {
  toolId: string;       // 工具ID
  toolName: string;     // 工具名称
  requestCount: number; // 请求次数
}

interface UserUsage {
  userId: string;         // 用户ID
  userName: string;       // 用户名称
  role: number;           // 用户角色: 1-owner, 2-admin, 3-member
  totalRequests: number;  // 总请求数
  toolsUsed: number;      // 使用的工具数量
  topTools: ToolUsage[];  // 最常用的工具TOP5
  lastActive: number;     // 最后活跃时间(时间戳)
}

interface Response20007Data {
  userUsage: UserUsage[];
  totalCount: number; // 总数量（用于分页）
}

/**
 * Protocol 20007 - Get User Tool Usage
 * 获取用户工具使用情况
 */
export async function handleProtocol20007(body: Request20007): Promise<Response20007Data> {
  try {
    const rawToken = body.common?.rawToken;
    const { timeRange, userId, page = 1, pageSize = 20 } = body.params;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-20007] Failed to get proxy info:', error);
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
      action: {
        in: [1001, 1002, 1003, 1004, 1005, 1006]
      },
      userid: {
        not: ''
      },
      serverId: {
        not: null
      }
    };
    
    // 如果指定了userId，添加过滤条件
    if (userId && userId.trim()) {
      whereCondition.userid = userId.trim();
    }
    
    // 1. 获取所有活跃用户
    const activeUsers = await prisma.log.findMany({
      where: whereCondition,
      select: {
        userid: true
      },
      distinct: ['userid']
    });
    
    const totalCount = activeUsers.length;

    const userIds = activeUsers.map(u => u.userid).filter(Boolean) as string[];
    const [requestCounts, usersResult] = await Promise.all([
      prisma.log.groupBy({
        by: ['userid'],
        where: { ...whereCondition, userid: { in: userIds } },
        _count: { id: true },
        orderBy: { _count: { id: 'desc' } },
      }),
      getUsers({}, body.common.userid, rawToken).catch(() => ({ users: [] })),
    ]);
    const userNameMap = new Map<string, string>(
      (usersResult.users || []).map((u: any) => [u.userId, u.name || u.userId])
    );
    const sortedUserIds = requestCounts.map(r => r.userid);
    const offset = (page - 1) * pageSize;
    const pagedUserIds = sortedUserIds.slice(offset, offset + pageSize);
    const pagedUsers = pagedUserIds.map(id => ({ userid: id }));
    
    // 3. 为每个用户计算详细统计
    const userUsage: UserUsage[] = await Promise.all(
      pagedUsers.map(async (user) => {
        const currentUserId = user.userid;
        
        // 该用户的所有日志条件
        const userWhereCondition = {
          ...whereCondition,
          userid: currentUserId
        };
        
        // 并行查询该用户的各项指标
        const [
          userInfo,
          totalRequestsCount,
          userToolsUsed,
          lastActiveLog,
          toolUsageStats
        ] = await Promise.all([
          // 从user表获取用户信息
          prisma.user.findFirst({
            where: {
              userid: currentUserId
            },
            select: {
              userid: true,
              role: true
            }
          }),
          
          // 总请求数
          prisma.log.count({
            where: userWhereCondition
          }),
          
          // 使用的工具数量（去重serverId）
          prisma.log.findMany({
            where: userWhereCondition,
            select: {
              serverId: true
            },
            distinct: ['serverId']
          }),
          
          // 最后活跃时间
          prisma.log.findFirst({
            where: userWhereCondition,
            orderBy: {
              addtime: 'desc'
            },
            select: {
              addtime: true
            }
          }),
          
          // 各工具使用统计
          prisma.log.groupBy({
            by: ['serverId'],
            where: userWhereCondition,
            _count: {
              id: true
            }
          })
        ]);
        
        // 处理工具使用统计，获取TOP5
        const topTools: ToolUsage[] = toolUsageStats
          .filter(stat => stat.serverId)
          .map(stat => ({
            toolId: stat.serverId!,
            toolName: `Tool ${stat.serverId}`,
            requestCount: stat._count.id
          }))
          .sort((a, b) => b.requestCount - a.requestCount)
          .slice(0, 5); // 取前5个
        
        // 用户角色和名称
        const role = userInfo?.role || 3;
        const userName = userNameMap.get(currentUserId) || currentUserId;
        
        // 最后活跃时间
        const lastActive = lastActiveLog ? Number(lastActiveLog.addtime) : 0;
        
        return {
          userId: currentUserId,
          userName,
          role,
          totalRequests: totalRequestsCount,
          toolsUsed: userToolsUsed.filter(t => t.serverId).length,
          topTools,
          lastActive
        };
      })
    );
    
    const response: Response20007Data = {
      userUsage,
      totalCount
    };
    
    console.log('Protocol 20007 response:', {
      userCount: userUsage.length,
      totalCount,
      page,
      pageSize,
      timeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20007 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get user tool usage' });
  }
}
