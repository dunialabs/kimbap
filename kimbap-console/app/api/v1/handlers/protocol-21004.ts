import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    tokenId: string;   // 特定令牌ID，空表示所有令牌
  };
}

interface GeoLocation {
  country: string;      // 国家代码
  countryName: string;  // 国家名称
  city: string;         // 城市名称
  requests: number;     // 请求数
  percentage: number;   // 占比(%)
  uniqueIPs: number;    // 独立IP数
}

interface TokenGeoUsage {
  tokenId: string;         // 令牌ID
  tokenName: string;       // 令牌名称
  locations: GeoLocation[]; // 地理位置分布
}

interface Response21004Data {
  geoUsage: TokenGeoUsage[];
}

/**
 * Protocol 21004 - Get Token Geographic Usage
 * 获取令牌地理位置使用分布
 */
export async function handleProtocol21004(body: Request21004): Promise<Response21004Data> {
  try {
    const { timeRange, tokenId } = body.params;

    const proxy = await getProxy();
    const proxyKey = proxy.proxyKey;
    
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
      tokenMask: {
        not: ''
      },
      ip: {
        not: ''
      }
    };
    
    // 如果指定了tokenId，查找对应的tokenMask
    if (tokenId && tokenId.trim()) {
      const user = await prisma.user.findFirst({
        where: {
          userid: tokenId.trim()
        },
        select: {
          accessTokenHash: true
        }
      });
      
      if (user) {
        whereCondition.tokenMask = user.accessTokenHash.substring(0, 16);
      }
    }
    
    // 获取所有相关的日志数据
    const logs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        tokenMask: true,
        ip: true
      }
    });
    
    // 按token分组
    const tokenLogsMap = new Map<string, typeof logs>();
    logs.forEach(log => {
      if (!tokenLogsMap.has(log.tokenMask)) {
        tokenLogsMap.set(log.tokenMask, []);
      }
      tokenLogsMap.get(log.tokenMask)!.push(log);
    });
    
    // 为每个token分析地理位置分布
    const geoUsage: TokenGeoUsage[] = Array.from(tokenLogsMap.entries()).map(([tokenMask, tokenLogs]) => {
      const locationGroups = new Map<string, { requests: number; ips: Set<string> }>();

      tokenLogs.forEach(log => {
        const rawIp = (log.ip || '').trim();
        if (!rawIp) return;
        const parts = rawIp.split('.');
        const subnet = parts.length === 4 ? `${parts[0]}.${parts[1]}.${parts[2]}.0/24` : rawIp;
        if (!locationGroups.has(subnet)) {
          locationGroups.set(subnet, { requests: 0, ips: new Set<string>() });
        }
        const group = locationGroups.get(subnet)!;
        group.requests += 1;
        group.ips.add(rawIp);
      });

      const locations: GeoLocation[] = [];

      const totalRequests = tokenLogs.length;

      locationGroups.forEach((group, subnet) => {
        locations.push({
          country: 'UN',
          countryName: 'Unknown',
          city: subnet,
          requests: group.requests,
          percentage: totalRequests > 0 ? Math.round((group.requests / totalRequests) * 1000) / 10 : 0,
          uniqueIPs: group.ips.size
        });
      });
      
      // 按请求数降序排列
      locations.sort((a, b) => b.requests - a.requests);
      
      return {
        tokenId: tokenMask.substring(0, 8) + '...',
        tokenName: `Token ${tokenMask.substring(0, 8)}...`,
        locations
      };
    });
    
    const response: Response21004Data = {
      geoUsage
    };
    
    console.log('Protocol 21004 response:', {
      tokensCount: geoUsage.length,
      timeRange,
      totalLocations: geoUsage.reduce((sum, token) => sum + token.locations.length, 0)
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 21004 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token geographic usage distribution' });
  }
}
