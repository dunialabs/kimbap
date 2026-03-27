import { Prisma } from '@prisma/client';
import { prisma } from '@/lib/prisma';
import { mapDashboardMetricRow, toCountNumber } from './protocol-10023.helpers';

interface DashboardLogMetricScalars {
  apiRequests: bigint | number | null;
  activeTokens: bigint | number | null;
  monthlyRequests: bigint | number | null;
}

export interface DashboardToolUsageRow {
  serverId: string | null;
  requestCount: bigint | number;
}

export interface DashboardTokenUsageRow {
  userid: string;
  requestCount: bigint | number;
}

export interface DashboardLogMetrics {
  apiRequestsCount: number;
  activeTokensCount: number;
  monthlyRequests: number;
  toolsUsageRows: DashboardToolUsageRow[];
  tokenUsageRows: DashboardTokenUsageRow[];
}

export async function getDashboardLogMetrics(
  startTimeSeconds: number,
  monthStartSeconds: number,
  proxyKey: string,
): Promise<DashboardLogMetrics> {
  if (!proxyKey) {
    return {
      apiRequestsCount: 0,
      activeTokensCount: 0,
      monthlyRequests: 0,
      toolsUsageRows: [],
      tokenUsageRows: [],
    };
  }

  const startTimeBigInt = BigInt(startTimeSeconds);
  const monthStartBigInt = BigInt(monthStartSeconds);
  const scopedFilter = Prisma.sql`AND proxy_key = ${proxyKey}`;

  const [scalarResult, toolsResult, tokenResult] = await Promise.allSettled([
    prisma.$queryRaw<DashboardLogMetricScalars[]>`
      SELECT
        COALESCE(SUM(CASE WHEN addtime >= ${startTimeBigInt} THEN 1 ELSE 0 END), 0) AS "apiRequests",
        COUNT(DISTINCT CASE WHEN addtime >= ${startTimeBigInt} AND userid <> '' THEN userid ELSE NULL END) AS "activeTokens",
        COALESCE(SUM(CASE WHEN addtime >= ${monthStartBigInt} THEN 1 ELSE 0 END), 0) AS "monthlyRequests"
      FROM log
      WHERE 1 = 1
      ${scopedFilter}
    `,
    prisma.$queryRaw<DashboardToolUsageRow[]>`
      SELECT
        server_id AS "serverId",
        COUNT(*) AS "requestCount"
      FROM log
      WHERE addtime >= ${startTimeBigInt}
        ${scopedFilter}
        AND server_id IS NOT NULL
      GROUP BY server_id
      ORDER BY COUNT(*) DESC
      LIMIT 10
    `,
    prisma.$queryRaw<DashboardTokenUsageRow[]>`
      SELECT
        userid,
        COUNT(*) AS "requestCount"
      FROM log
      WHERE addtime >= ${startTimeBigInt}
        ${scopedFilter}
        AND userid <> ''
        AND userid NOT LIKE '%unknown%'
      GROUP BY userid
      ORDER BY COUNT(*) DESC
      LIMIT 10
    `,
  ]);

  if (scalarResult.status === 'rejected') {
    console.error('Failed to get dashboard scalar log metrics:', scalarResult.reason);
  }
  if (toolsResult.status === 'rejected') {
    console.error('Failed to get dashboard tools usage rows:', toolsResult.reason);
  }
  if (tokenResult.status === 'rejected') {
    console.error('Failed to get dashboard token usage rows:', tokenResult.reason);
  }

  const scalarRows = scalarResult.status === 'fulfilled' ? scalarResult.value : [];
  const toolsUsageRows = toolsResult.status === 'fulfilled' ? toolsResult.value : [];
  const tokenUsageRows = tokenResult.status === 'fulfilled' ? tokenResult.value : [];
  const scalar = scalarRows[0];

  return {
    apiRequestsCount: toCountNumber(scalar?.apiRequests),
    activeTokensCount: toCountNumber(scalar?.activeTokens),
    monthlyRequests: toCountNumber(scalar?.monthlyRequests),
    toolsUsageRows,
    tokenUsageRows,
  };
}

export function formatToolsUsage(
  toolUsageRows: DashboardToolUsageRow[],
  serversMap: Map<string, string>,
): Array<{ name: string; requests: number; percentage: number }> {
  const validToolUsageRows = toolUsageRows.filter((item) =>
    item.serverId &&
    item.serverId.trim() !== '' &&
    item.serverId.toLowerCase() !== 'unknown'
  );

  const totalRequests = validToolUsageRows.reduce((sum, item) => sum + toCountNumber(item.requestCount), 0);

  return validToolUsageRows.map((item) => {
    const serverId = item.serverId as string;
    const serverName = serversMap.get(serverId);
    const displayName = serverName || `${serverId}(Deleted)`;
    return mapDashboardMetricRow(displayName, item.requestCount, totalRequests);
  });
}

export function formatTokenUsage(
  tokenUsageRows: DashboardTokenUsageRow[],
  usersMap: Map<string, string>,
): Array<{ name: string; token: string; requests: number; percentage: number }> {
  const totalRequests = tokenUsageRows.reduce((sum, item) => sum + toCountNumber(item.requestCount), 0);

  return tokenUsageRows.map((item) => {
    const userName = usersMap.get(item.userid);
    const displayName = userName || `${item.userid}(Deleted)`;
    const metricRow = mapDashboardMetricRow(displayName, item.requestCount, totalRequests);

    return {
      name: metricRow.name,
      token: '',
      requests: metricRow.requests,
      percentage: metricRow.percentage,
    };
  });
}
