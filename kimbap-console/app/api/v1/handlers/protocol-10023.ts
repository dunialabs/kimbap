import { getServers, getUsers, getProxy } from '@/lib/proxy-api';
import { ApiError } from '@/lib/error-codes';
import { getRecentActivity } from './protocol-10023.activity';
import { getConnectedClientsList } from './protocol-10023.clients';
import { getDashboardLogMetrics, formatToolsUsage, formatTokenUsage } from './protocol-10023.metrics';
import { getServerUptime, getStartTimeByRange, resolveManualConnection } from './protocol-10023.shared';

interface Request10023 {
  common: {
    cmdId: number;
    userid?: string;
    rawToken?: string;
  };
  params: {
    timeRange?: string;
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

export async function handleProtocol10023(body: Request10023): Promise<Response10023Data> {
  const { timeRange = '30d' } = body.params || {};
  const userid = body.common.userid;
  const rawToken = body.common?.rawToken;

  try {
    const sshTunnelAddress = '';
    const manualConnection = await resolveManualConnection();
    const now = Date.now();
    const startTime = getStartTimeByRange(timeRange, now);
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

    const usersMap = new Map<string, string>();
    const serversMap = new Map<string, string>();

    if (userid) {
      const [usersResult, serversResult] = await Promise.allSettled([
        getUsers({}, userid, rawToken),
        getServers({}, userid, rawToken),
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

    const [uptime, dashboardLogMetrics, connectedClients, recentActivity] = await Promise.all([
      getServerUptime(proxyKey),
      getDashboardLogMetrics(startTimeSeconds, monthStartSeconds, proxyKey),
      getConnectedClientsList(usersMap, proxyKey),
      getRecentActivity(usersMap, proxyKey),
    ]);

    const configuredToolsCount = serversMap.size;
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
