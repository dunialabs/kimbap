import { getActionLabel } from '@/lib/log-utils';

export interface RecentActivityLogRow {
  action: number;
  userid: string;
  serverId: string | null;
  addtime: bigint | number;
  error: string;
}

export interface GroupedConnectedClientRow {
  userid: string;
  ip: string;
  _max: {
    addtime: bigint | number | null;
  };
  _count: {
    id: number;
  };
}

export function toCountNumber(value: bigint | number | null | undefined): number {
  if (typeof value === 'bigint') {
    return Number(value);
  }
  if (typeof value === 'number') {
    return value;
  }
  return 0;
}

export function mapDashboardMetricRow(displayName: string, requestCount: bigint | number, totalRequests: number): {
  name: string;
  requests: number;
  percentage: number;
} {
  const requests = toCountNumber(requestCount);
  return {
    name: displayName,
    requests,
    percentage: totalRequests > 0 ? Math.round((requests * 100) / totalRequests) : 0,
  };
}

export function formatTimeAgo(timestampSeconds: number): string {
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

export function formatAction(action: number, userid: string, userNamesMap?: Map<string, string>): string {
  const actionDesc = getActionLabel(action);
  const userName = userNamesMap?.get(userid);
  const displayUser = userName || `${userid}(Deleted)`;
  return `${displayUser} - ${actionDesc}`;
}

export function formatRecentActivityRows(recentLogs: RecentActivityLogRow[], usersMap: Map<string, string>): Array<{
  action: string;
  status: 'success' | 'warning' | 'info';
  time: string;
}> {
  return recentLogs.map((log) => {
    const hasError = log.error && log.error !== '';
    const action = formatAction(log.action, log.userid, usersMap);
    const time = formatTimeAgo(Number(log.addtime));

    return {
      action,
      status: hasError ? 'warning' : 'success',
      time,
    };
  });
}

export function mapConnectedClientRow(
  client: GroupedConnectedClientRow,
  usersMap: Map<string, string>,
): {
  id: string;
  name: string;
  token: string;
  ip: string;
  location: string;
  lastActive: string;
  requests: number;
} | null {
  const userName = usersMap.get(client.userid);
  if (!userName) {
    return null;
  }

  const requestCount = client._count.id;
  const lastActiveTimestamp = client._max.addtime ? Number(client._max.addtime) : 0;
  const lastActiveTime = formatTimeAgo(lastActiveTimestamp);

  return {
    id: `${client.userid}-${client.ip}`,
    name: userName,
    token: '',
    ip: client.ip,
    location: 'Unknown',
    lastActive: lastActiveTime,
    requests: requestCount,
  };
}
