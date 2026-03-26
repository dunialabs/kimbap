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
  toolId: string;       // toolID
  toolName: string;     // Tool name
  requestCount: number; // Number of requests
}

interface UserUsage {
  userId: string;         // userID
  userName: string;       // Username
  role: number;           // user role: 1-owner, 2-admin, 3-member
  totalRequests: number;  // total requests
  toolsUsed: number;      // Number of tools used
  topTools: ToolUsage[];  // most commonly used toolsTOP5
  lastActive: number;     // Last active time(Timestamp)
}

interface Response20007Data {
  userUsage: UserUsage[];
  totalCount: number; // total quantity（for paging）
}

/**
 * Protocol 20007 - Get User Tool Usage
 * Get user tool usage status
 */
export async function handleProtocol20007(body: Request20007): Promise<Response20007Data> {
  try {
    const rawToken = body.common?.rawToken;
    const { userId } = body.params;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;
    const safePage = Math.max(1, Math.floor(Number(body.params.page) || 1));
    const safePageSize = Math.min(1000, Math.max(1, Math.floor(Number(body.params.pageSize) || 20)));

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
    
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
    // buildwherecondition
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
    
    // If specifieduserId，Add filter
    if (userId && userId.trim()) {
      whereCondition.userid = userId.trim();
    }
    
    // 1. Get all active users
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
    const offset = (safePage - 1) * safePageSize;
    const pagedUserIds = sortedUserIds.slice(offset, offset + safePageSize);
    const pagedUsers = pagedUserIds.map(id => ({ userid: id }));
    
    // 3. Calculate detailed statistics for each user
    const userUsage: UserUsage[] = await Promise.all(
      pagedUsers.map(async (user) => {
        const currentUserId = user.userid;
        
        // All log conditions for this user
        const userWhereCondition = {
          ...whereCondition,
          userid: currentUserId
        };
        
        // Query various indicators of the user in parallel
        const [
          userInfo,
          totalRequestsCount,
          userToolsUsed,
          lastActiveLog,
          toolUsageStats
        ] = await Promise.all([
          // fromuserGet user information from table
          prisma.user.findFirst({
            where: {
              userid: currentUserId
            },
            select: {
              userid: true,
              role: true
            }
          }),
          
          // total requests
          prisma.log.count({
            where: userWhereCondition
          }),
          
          // Number of tools used（Remove duplicatesserverId）
          prisma.log.findMany({
            where: userWhereCondition,
            select: {
              serverId: true
            },
            distinct: ['serverId']
          }),
          
          // Last active time
          prisma.log.findFirst({
            where: userWhereCondition,
            orderBy: {
              addtime: 'desc'
            },
            select: {
              addtime: true
            }
          }),
          
          // Usage statistics of each tool
          prisma.log.groupBy({
            by: ['serverId'],
            where: userWhereCondition,
            _count: {
              id: true
            }
          })
        ]);
        
        // Process tool usage statistics，getTOP5
        const topTools: ToolUsage[] = toolUsageStats
          .filter(stat => stat.serverId)
          .map(stat => ({
            toolId: stat.serverId!,
            toolName: `Tool ${stat.serverId}`,
            requestCount: stat._count.id
          }))
          .sort((a, b) => b.requestCount - a.requestCount)
          .slice(0, 5); // Before picking up5indivual
        
        // User role and name
        const role = userInfo?.role || 3;
        const userName = userNameMap.get(currentUserId) || currentUserId;
        
        // Last active time
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
      page: safePage,
      pageSize: safePageSize,
      timeRange: normalizedTimeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 20007 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get user tool usage' });
  }
}
