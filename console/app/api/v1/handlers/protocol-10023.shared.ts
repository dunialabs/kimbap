import { prisma } from '@/lib/prisma';

export async function resolveManualConnection(): Promise<string> {
  let manualConnection = '';

  try {
    const configData = await prisma.config.findFirst();
    if (configData && configData.kimbap_core_host) {
      const host = configData.kimbap_core_host;
      const port: number | undefined = configData.kimbap_core_port || undefined;

      if (host.startsWith('http://') || host.startsWith('https://')) {
        const url = new URL(host);
        const isHttps = url.protocol === 'https:';
        const defaultPort = isHttps ? 443 : 80;

        if (port && port !== defaultPort) {
          url.port = String(port);
        }
        manualConnection = url.toString();
      } else if (host) {
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
  }

  if (!manualConnection) {
    const mcpGatewayUrl = process.env.KIMBAP_CORE_URL?.trim();
    if (mcpGatewayUrl) {
      try {
        const parsed = new URL(mcpGatewayUrl);
        manualConnection = parsed.origin;
      } catch {}
    }
  }

  return manualConnection;
}

export function getStartTimeByRange(timeRange: string, now: number): number {
  switch (timeRange) {
    case '24h':
      return now - (24 * 60 * 60 * 1000);
    case '7d':
      return now - (7 * 24 * 60 * 60 * 1000);
    case '30d':
      return now - (30 * 24 * 60 * 60 * 1000);
    case '90d':
      return now - (90 * 24 * 60 * 60 * 1000);
    default:
      return now - (30 * 24 * 60 * 60 * 1000);
  }
}

export async function getServerUptime(proxyKey: string): Promise<string> {
  if (!proxyKey) {
    return '0 days, 0 hours';
  }

  try {
    const firstLog = await prisma.log.findFirst({
      where: {
        proxyKey,
      },
      orderBy: {
        addtime: 'asc'
      }
    });

    if (firstLog) {
      const startTimeSeconds = Number(firstLog.addtime);
      const nowSeconds = Math.floor(Date.now() / 1000);
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
