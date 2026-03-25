import { Prisma } from '@prisma/client';
import { prisma } from '@/lib/prisma';
import { formatRecentActivityRows, type RecentActivityLogRow } from './protocol-10023.helpers';

export async function getRecentActivity(
  usersMap: Map<string, string>,
  proxyKey: string,
): Promise<Array<{ action: string; status: 'success' | 'warning' | 'info'; time: string }>> {
  try {
    if (!proxyKey) {
      return [];
    }

    const scopedFilter = Prisma.sql`AND proxy_key = ${proxyKey}`;

    const recentLogs = await prisma.$queryRaw<RecentActivityLogRow[]>`
      SELECT action, userid, server_id AS "serverId", addtime, error
      FROM (
        SELECT action, userid, server_id, addtime, error
        FROM log
        WHERE 1 = 1
        ${scopedFilter}
        ORDER BY addtime DESC
        LIMIT 20
      ) recent
      WHERE userid !~ '^[[:space:]]*$'
        AND LOWER(userid) <> 'unknown'
      ORDER BY addtime DESC
      LIMIT 10
    `;

    return formatRecentActivityRows(recentLogs, usersMap);
  } catch (error) {
    console.error('Failed to get recent activity:', error);
    return [];
  }
}
