import { prisma } from '@/lib/prisma';
import { Prisma } from '@prisma/client';
import { 
  getServers,
  getUsers,
  getProxy
} from '@/lib/proxy-api';
import { ApiError } from '@/lib/error-codes';

interface Request10023 {
  common: {
    cmdId: number;
    userid?: string;
  };
  params: {
    timeRange?: string; // "24h", "7d", "30d", "90d"
  };
}

interface ToolUsage {
  name: string;
  requests: number;
  percentage: number;
}

interface TokenUsage {
  name: string;
  token: string;
  requests: number;
  percentage: number;
}

interface ConnectedClient {
  id: string;
  name: string;
  token: string;
  ip: string;
  location: string;
  lastActive: string;
  requests: number;
}

interface RecentActivity {
  action: string;
  status: 'success' | 'warning' | 'info';
  time: string;
}

interface Response10023Data {
  uptime: string;
  apiRequests: number;
  activeTokens: number;
  configuredTools: number;
  connectedClientsCount: number;
  monthlyUsage: number;
  toolsUsage: ToolUsage[];
  tokenUsage: TokenUsage[];
  connectedClients: ConnectedClient[];
  recentActivity: RecentActivity[];
  sshTunnelAddress: string;
  manualConnection: string;
}

/**
 * Protocol 10023: Dashboard Overview Statistics
 * Returns comprehensive dashboard statistics including:
 * - Server metrics (uptime, requests, tokens, tools, clients)
 * - Usage distributions (tools, tokens)
 * - Connected clients details
 * - Recent activity log
 * - sshTunnelAddress: SSH tunnel address from cloudflared config
 * - manualConnection: configured KIMBAP Core host and port from config table
 */
export async function handleProtocol10023(body: Request10023): Promise<Response10023Data> {
  const { timeRange = '30d' } = body.params || {};
  const userid = body.common.userid;
  
  try {
    const sshTunnelAddress = '';
    
    // Get manualConnection from config table (same as 10014)
    let manualConnection = '';
    try {
      const configData = await prisma.config.findFirst();
      if (configData && configData.kimbap_core_host) {
        const host = configData.kimbap_core_host;
        const currentPort = Reflect.get(configData, 'kimbap_core_port');
        const legacyPort = Reflect.get(configData, 'kimbap_core_prot');
        const port =
          typeof currentPort === 'number'
            ? currentPort
            : typeof legacyPort === 'number'
              ? legacyPort
              : undefined;
        
        // Build the connection string
        if (host.startsWith('http://') || host.startsWith('https://')) {
          // Host already contains protocol
          const url = new URL(host);
          const isHttps = url.protocol === 'https:';
          const defaultPort = isHttps ? 443 : 80;
          
          // Add port if it's not the default for the protocol
          if (port && port !== defaultPort) {
            manualConnection = `${host}:${port}`;
          } else {
            manualConnection = host;
          }
        } else if (host) {
          // Host doesn't contain protocol, add it based on type
          const isIP = /^(\d{1,3}\.){3}\d{1,3}$/.test(host) || host === 'localhost';
          const protocol = isIP ? 'http' : 'https';
          const defaultPort = isIP ? 80 : 443;
          
          if (port && port !== defaultPort) {
            manualConnection = `${protocol}://${host}:${port}`;
          } else {
            manualConnection = `${protocol}://${host}`;
          }
        }
      }
    } catch (error) {
      console.error('Error getting config for manualConnection:', error);
      // Keep manualConnection as empty string
    }

    // Fallback to MCP_GATEWAY_URL env var when database config is empty
    // This matches the 3-tier priority used in proxy-api.ts and protocol-10021
    if (!manualConnection) {
      const mcpGatewayUrl = process.env.MCP_GATEWAY_URL?.trim();
      if (mcpGatewayUrl) {
        try {
          const parsed = new URL(mcpGatewayUrl);
          // Use origin to strip trailing slashes, paths, or accidental suffixes
          manualConnection = parsed.origin;
        } catch {
          // Invalid URL format, keep manualConnection as empty
        }
      }
    }
    
    // Calculate time range
    const now = Date.now();
    let startTime = now;
    
    switch (timeRange) {
      case '24h':
        startTime = now - (24 * 60 * 60 * 1000);
        break;
      case '7d':
        startTime = now - (7 * 24 * 60 * 60 * 1000);
        break;
      case '30d':
        startTime = now - (30 * 24 * 60 * 60 * 1000);
        break;
      case '90d':
        startTime = now - (90 * 24 * 60 * 60 * 1000);
        break;
      default:
        startTime = now - (30 * 24 * 60 * 60 * 1000); // Default to 30 days
    }
    
    const startTimeSeconds = Math.floor(startTime / 1000);
    const monthStartDate = new Date();
    const monthStartSeconds = Math.floor(new Date(monthStartDate.getFullYear(), monthStartDate.getMonth(), 1).getTime() / 1000);
    
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('Failed to get proxy info:', error);
    }

    let usersMap = new Map<string, string>();
    let serversMap = new Map<string, string>();

    if (userid) {
      const [usersResult, serversResult] = await Promise.allSettled([
        getUsers({}, userid),
        getServers({}, userid),
      ]);

      if (usersResult.status === 'fulfilled') {
        usersResult.value.users.forEach((user: any) => {
          if (user.userId && user.name) {
            usersMap.set(user.userId, user.name);
          }
        });
      } else {
        console.error('Failed to get user names from proxy API:', usersResult.reason);
      }

      if (serversResult.status === 'fulfilled') {
        serversResult.value.servers.forEach((server: any) => {
          if (server.serverId && server.serverName) {
            serversMap.set(server.serverId, server.serverName);
          }
        });
      } else {
        console.error('Failed to get server names:', serversResult.reason);
      }
    }

    const [
      uptime,
      dashboardLogMetrics,
      configuredToolsCount,
      connectedClients,
      recentActivity,
    ] = await Promise.all([
      getServerUptime(proxyKey),
      getDashboardLogMetrics(startTimeSeconds, monthStartSeconds, proxyKey),
      getConfiguredToolsCount(serversMap),
      getConnectedClientsList(usersMap, proxyKey),
      getRecentActivity(usersMap, proxyKey),
    ]);

    const apiRequestsCount = dashboardLogMetrics.apiRequestsCount;
    const activeTokensCount = dashboardLogMetrics.activeTokensCount;
    const monthlyLimit = 100000;
    const monthlyUsage = Math.min(100, Math.round((dashboardLogMetrics.monthlyRequests * 100) / monthlyLimit));
    const toolsUsage = formatToolsUsage(dashboardLogMetrics.toolsUsageRows, serversMap);
    const tokenUsage = formatTokenUsage(dashboardLogMetrics.tokenUsageRows, usersMap);
    
    return {
      uptime,
      apiRequests: apiRequestsCount,
      activeTokens: activeTokensCount,
      configuredTools: configuredToolsCount,
      connectedClientsCount: connectedClients.length,
      monthlyUsage,
      toolsUsage,
      tokenUsage,
      connectedClients,
      recentActivity,
      sshTunnelAddress,
      manualConnection
    };
    
  } catch (error) {
    console.error('Protocol 10023 error:', error);
    
    if (error instanceof ApiError) {
      throw error;
    }
    
    // Return empty/default data on error
    return {
      uptime: '0 days, 0 hours',
      apiRequests: 0,
      activeTokens: 0,
      configuredTools: 0,
      connectedClientsCount: 0,
      monthlyUsage: 0,
      toolsUsage: [],
      tokenUsage: [],
      connectedClients: [],
      recentActivity: [],
      sshTunnelAddress: '',
      manualConnection: ''
    };
  }
}

// Helper functions

async function getServerUptime(proxyKey: string): Promise<string> {
  if (!proxyKey) {
    return '0 days, 0 hours';
  }

  try {
    // Get first log entry to determine server start time
    const firstLog = await prisma.log.findFirst({
      where: {
        proxyKey,
      },
      orderBy: {
        addtime: 'asc'
      }
    });
    
    if (firstLog) {
      const startTimeSeconds = Number(firstLog.addtime); // Already in seconds
      const nowSeconds = Math.floor(Date.now() / 1000); // Convert current time to seconds
      const uptimeSeconds = nowSeconds - startTimeSeconds;
      
      const days = Math.floor(uptimeSeconds / (24 * 60 * 60));
      const hours = Math.floor((uptimeSeconds % (24 * 60 * 60)) / (60 * 60));
      
      return `${days} days, ${hours} hours`;
    }
    
    return '0 days, 0 hours';
  } catch (error) {
    console.error('Failed to get server uptime:', error);
    return '0 days, 0 hours';
  }
}

interface DashboardLogMetricScalars {
  apiRequests: bigint | number | null;
  activeTokens: bigint | number | null;
  monthlyRequests: bigint | number | null;
}

interface DashboardToolUsageRow {
  serverId: string | null;
  requestCount: bigint | number;
}

interface DashboardTokenUsageRow {
  userid: string;
  requestCount: bigint | number;
}

interface DashboardLogMetrics {
  apiRequestsCount: number;
  activeTokensCount: number;
  monthlyRequests: number;
  toolsUsageRows: DashboardToolUsageRow[];
  tokenUsageRows: DashboardTokenUsageRow[];
}

async function getDashboardLogMetrics(startTimeSeconds: number, monthStartSeconds: number, proxyKey: string): Promise<DashboardLogMetrics> {
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
        COUNT(*) FILTER (WHERE addtime >= ${startTimeBigInt}) AS "apiRequests",
        COUNT(DISTINCT userid) FILTER (WHERE addtime >= ${startTimeBigInt} AND userid <> '') AS "activeTokens",
        COUNT(*) FILTER (WHERE addtime >= ${monthStartBigInt}) AS "monthlyRequests"
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

async function getConfiguredToolsCount(serversMap: Map<string, string>): Promise<number> {
  try {
    return serversMap.size;
  } catch (error) {
    console.error('Failed to get configured tools count:', error);
    return 0;
  }
}


function formatToolsUsage(toolUsageRows: DashboardToolUsageRow[], serversMap: Map<string, string>): ToolUsage[] {
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
    const requests = toCountNumber(item.requestCount);

    return {
      name: displayName,
      requests,
      percentage: totalRequests > 0 ? Math.round((requests * 100) / totalRequests) : 0,
    };
  });
}

function formatTokenUsage(tokenUsageRows: DashboardTokenUsageRow[], usersMap: Map<string, string>): TokenUsage[] {
  const totalRequests = tokenUsageRows.reduce((sum, item) => sum + toCountNumber(item.requestCount), 0);

  return tokenUsageRows.map((item) => {
    const userName = usersMap.get(item.userid);
    const displayName = userName || `${item.userid}(Deleted)`;
    const requests = toCountNumber(item.requestCount);

    return {
      name: displayName,
      token: '',
      requests,
      percentage: totalRequests > 0 ? Math.round((requests * 100) / totalRequests) : 0,
    };
  });
}

function toCountNumber(value: bigint | number | null | undefined): number {
  if (typeof value === 'bigint') {
    return Number(value);
  }
  if (typeof value === 'number') {
    return value;
  }
  return 0;
}

async function getConnectedClientsList(usersMap: Map<string, string>, proxyKey: string): Promise<ConnectedClient[]> {
  if (!proxyKey) {
    return [];
  }

  try {
    const last24Hours = BigInt(Math.floor((Date.now() - 24 * 60 * 60 * 1000) / 1000));

    const groupedClients = await prisma.log.groupBy({
      by: ['userid', 'ip'],
      where: {
        proxyKey,
        addtime: { gte: last24Hours },
        userid: { not: '' },
      },
      _max: { addtime: true },
      _count: { id: true },
      orderBy: [
        {
          _max: {
            addtime: 'desc',
          },
        },
        { userid: 'asc' },
        { ip: 'asc' },
      ],
      take: 20,
    });
    
    // Format client list
    const connectedClientsList: ConnectedClient[] = [];
    
    for (const client of groupedClients) {
      const userName = usersMap.get(client.userid);
      
      // Skip clients without names
      if (!userName) {
        continue;
      }
      
      const requestCount = client._count.id;
      const lastActiveTimestamp = client._max.addtime ? Number(client._max.addtime) : 0;
      
      const lastActiveTime = formatTimeAgo(lastActiveTimestamp);
      
      connectedClientsList.push({
        id: `${client.userid}-${client.ip}`,
        name: userName,  // Use the real user name
        token: '',  // Don't return the actual token
        ip: client.ip,
        location: 'Unknown', // Could be enhanced with IP geolocation
        lastActive: lastActiveTime,
        requests: requestCount
      });
    }
    
    return connectedClientsList;
  } catch (error) {
    console.error('Failed to get connected clients list:', error);
    return [];
  }
}

async function getRecentActivity(usersMap: Map<string, string>, proxyKey: string): Promise<RecentActivity[]> {
  try {
    if (!proxyKey) {
      return [];
    }

    const scopedFilter = Prisma.sql`AND proxy_key = ${proxyKey}`;

    const recentLogs = await prisma.$queryRaw<Array<{
      action: number;
      userid: string;
      serverId: string | null;
      addtime: bigint | number;
      error: string;
    }>>`
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

    const recentActivity: RecentActivity[] = recentLogs.map(log => {
        const hasError = log.error && log.error !== '';
        const action = formatAction(log.action, log.userid, usersMap);
        const time = formatTimeAgo(Number(log.addtime));
        
        return {
          action,
          status: hasError ? 'warning' : 'success',
          time
        };
      });
    
    return recentActivity;
  } catch (error) {
    console.error('Failed to get recent activity:', error);
    return [];
  }
}

// Utility functions

function formatTimeAgo(timestampSeconds: number): string {
  const nowSeconds = Math.floor(Date.now() / 1000);
  const diffSeconds = nowSeconds - timestampSeconds;
  
  const minutes = Math.floor(diffSeconds / 60);
  const hours = Math.floor(diffSeconds / (60 * 60));
  const days = Math.floor(diffSeconds / (24 * 60 * 60));
  
  if (days > 0) {
    return `${days} day${days > 1 ? 's' : ''} ago`;
  } else if (hours > 0) {
    return `${hours} hour${hours > 1 ? 's' : ''} ago`;
  } else if (minutes > 0) {
    return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
  } else {
    return 'Just now';
  }
}

function formatAction(action: number, userid: string, userNamesMap?: Map<string, string>): string {
  // Map action codes to descriptions
  const actionMap: Record<number, string> = {
    10001: 'User login',
    10002: 'Tool configuration',
    10003: 'Server start',
    10004: 'Server stop',
    10005: 'Token created',
    10006: 'Token deleted',
    // Add more action mappings as needed
  };
  
  const actionDesc = actionMap[action] || `Action ${action}`;
  
  // Get user name if available, otherwise use userid(Deleted)
  const userName = userNamesMap?.get(userid);
  const displayUser = userName || `${userid}(Deleted)`;
  
  return `${displayUser} - ${actionDesc}`;
}
