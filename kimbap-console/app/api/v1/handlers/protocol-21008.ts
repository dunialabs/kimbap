import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21008 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    tokenId: string;   // 特定令牌ID，空表示所有令牌
    page: number;      // 页码，从1开始
    pageSize: number;  // 每页大小
  };
}

interface ClientInfo {
  clientIP: string;         // 客户端IP（脱敏）
  userAgent: string;        // 用户代理（脱敏）
  requests: number;         // 请求数量
  successRequests: number;  // 成功请求数
  failedRequests: number;   // 失败请求数
  firstSeen: number;        // 首次使用时间
  lastSeen: number;         // 最后使用时间
  country: string;          // 地理位置-国家
  city: string;             // 地理位置-城市
  riskLevel: number;        // 风险等级 1-低, 2-中, 3-高
}

interface TokenClientAnalysis {
  tokenId: string;          // 令牌ID
  tokenName: string;        // 令牌名称
  totalClients: number;     // 总客户端数
  clients: ClientInfo[];    // 客户端详情
}

interface Response21008Data {
  clientAnalysis: TokenClientAnalysis[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

/**
 * Protocol 21008 - Get Token Client Analysis
 * 获取令牌客户端分析
 */
export async function handleProtocol21008(body: Request21008): Promise<Response21008Data> {
  try {
    const { timeRange, tokenId, page = 1, pageSize = 20 } = body.params;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-21008] Failed to get proxy info:', error);
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
      tokenMask: {
        not: ''
      },
      ip: {
        not: ''
      }
    };
    
    if (tokenId && tokenId.trim()) {
      whereCondition.userid = tokenId.trim();
    }
    
    // 获取所有相关的日志数据
    const logs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        tokenMask: true,
        ip: true,
        ua: true,
        statusCode: true,
        addtime: true
      },
      orderBy: {
        addtime: 'desc'
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
    
    // 辅助函数：IP脱敏
    const maskIPAddress = (ip: string): string => {
      const parts = ip.split('.');
      if (parts.length === 4) {
        return `${parts[0]}.${parts[1]}.xxx.xxx`;
      }
      return 'xxx.xxx.xxx.xxx';
    };
    
    // 辅助函数：用户代理脱敏
    const maskUserAgent = (ua: string): string => {
      if (!ua) return 'Unknown';
      return ua.substring(0, 50) + (ua.length > 50 ? '...' : '');
    };
    
    // 辅助函数：模拟地理位置
    const getGeolocation = (ip: string): { country: string; city: string } => {
      // 模拟地理位置数据（实际应该使用IP地理位置库）
      const geoData = [
        { country: 'US', city: 'New York' },
        { country: 'US', city: 'San Francisco' },
        { country: 'UK', city: 'London' },
        { country: 'DE', city: 'Berlin' },
        { country: 'JP', city: 'Tokyo' },
        { country: 'CA', city: 'Toronto' },
        { country: 'AU', city: 'Sydney' },
        { country: 'FR', city: 'Paris' }
      ];

      const hash = ip.split('.').reduce((acc, part) => acc + parseInt(part) || 0, 0);
      return geoData[hash % geoData.length];
    };
    
    // 辅助函数：计算风险等级
    const calculateRiskLevel = (clientStats: {
      requests: number;
      failedRequests: number;
      timeSpan: number;
    }): number => {
      const { requests, failedRequests, timeSpan } = clientStats;
      const failureRate = requests > 0 ? failedRequests / requests : 0;
      const requestRate = timeSpan > 0 ? requests / (timeSpan / 3600) : 0; // 每小时请求数
      
      // 风险评估逻辑
      if (failureRate > 0.5 || requestRate > 1000) {
        return 3; // 高风险
      } else if (failureRate > 0.2 || requestRate > 500) {
        return 2; // 中风险
      } else {
        return 1; // 低风险
      }
    };
    
    // 为每个token分析客户端
    const allClientAnalysis: TokenClientAnalysis[] = Array.from(tokenLogsMap.entries()).map(([tokenMask, tokenLogs]) => {
      // 按IP和UA组合分组客户端
      const clientGroups = new Map<string, typeof tokenLogs>();
      tokenLogs.forEach(log => {
        const clientKey = `${log.ip}_${log.ua || 'unknown'}`;
        if (!clientGroups.has(clientKey)) {
          clientGroups.set(clientKey, []);
        }
        clientGroups.get(clientKey)!.push(log);
      });
      
      // 为每个客户端生成统计信息
      const clients: ClientInfo[] = Array.from(clientGroups.entries()).map(([clientKey, clientLogs]) => {
        const [ip, ua] = clientKey.split('_');
        const requests = clientLogs.length;
        const successRequests = clientLogs.filter(log => log.statusCode != null && log.statusCode >= 200 && log.statusCode < 300).length;
        const failedRequests = requests - successRequests;
        
        const timestamps = clientLogs.map(log => Number(log.addtime)).sort((a, b) => a - b);
        const firstSeen = timestamps[0];
        const lastSeen = timestamps[timestamps.length - 1];
        const timeSpan = lastSeen - firstSeen;
        
        const geo = getGeolocation(ip);
        const riskLevel = calculateRiskLevel({ requests, failedRequests, timeSpan });
        
        return {
          clientIP: maskIPAddress(ip),
          userAgent: maskUserAgent(ua === 'unknown' ? '' : ua),
          requests,
          successRequests,
          failedRequests,
          firstSeen,
          lastSeen,
          country: geo.country,
          city: geo.city,
          riskLevel
        };
      });
      
      // 按请求数降序排列
      clients.sort((a, b) => b.requests - a.requests);
      
      return {
        tokenId: tokenMask.substring(0, 8) + '...',
        tokenName: `Token ${tokenMask.substring(0, 8)}...`,
        totalClients: clients.length,
        clients
      };
    });
    
    // 按客户端数降序排列
    allClientAnalysis.sort((a, b) => b.totalClients - a.totalClients);
    
    // 分页处理
    const total = allClientAnalysis.length;
    const totalPages = Math.ceil(total / pageSize);
    const startIndex = (page - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    const paginatedAnalysis = allClientAnalysis.slice(startIndex, endIndex);
    
    const response: Response21008Data = {
      clientAnalysis: paginatedAnalysis,
      pagination: {
        page,
        pageSize,
        total,
        totalPages
      }
    };
    
    console.log('Protocol 21008 response:', {
      totalTokens: total,
      paginatedTokens: paginatedAnalysis.length,
      totalClients: allClientAnalysis.reduce((sum, token) => sum + token.totalClients, 0),
      timeRange
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 21008 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token client analysis' });
  }
}
