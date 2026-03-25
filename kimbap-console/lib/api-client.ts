import axios from 'axios';
import { toast } from 'sonner';
import { getApiErrorMessage } from './api-error';
import { hasUrls, renderErrorMessageWithLinks } from './error-utils';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';
const isDev = process.env.NODE_ENV === 'development';
const AUTH_TOKEN_KEY = 'auth_token';
const LEGACY_AUTH_TOKEN_KEY = 'accessToken';

const readAuthTokenFromStorage = () => {
  if (typeof window === 'undefined') {
    return null;
  }

  const canonicalToken = localStorage.getItem(AUTH_TOKEN_KEY);
  if (canonicalToken) {
    return canonicalToken;
  }

  const legacyToken = localStorage.getItem(LEGACY_AUTH_TOKEN_KEY);
  if (legacyToken) {
    localStorage.setItem(AUTH_TOKEN_KEY, legacyToken);
    localStorage.removeItem(LEGACY_AUTH_TOKEN_KEY);
    return legacyToken;
  }

  return null;
};

// Create axios instance with default config
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000, // 30 seconds timeout
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token and userid
apiClient.interceptors.request.use((config) => {
  if (isDev) {
    console.log('[API Client Interceptor] Request URL:', config.url);
  }

  if (typeof window !== 'undefined') {
    const token = readAuthTokenFromStorage();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    if (config.url?.includes('/api/v1') && config.data) {
      const userid = localStorage.getItem('userid');
      if (isDev) {
        console.log('[API Client Interceptor] userid from localStorage:', userid);
      }

      if (userid && config.data.common) {
        config.data.common.userid = userid;
        if (isDev) {
          console.log(
            '[API Client Interceptor] Added userid to request. cmdId:',
            config.data.common.cmdId,
          );
        }
      } else if (isDev) {
        console.log(
          '[API Client Interceptor] NOT adding userid. has common:',
          !!config.data.common,
        );
      }
    }
  }

  return config;
});

// Response interceptor for统一错误处理 + toast
apiClient.interceptors.response.use(
  (response) => {
    try {
      const data = response?.data as any;
      const common = data?.common;

      // 业务层错误：/api/v1 标准返回里 code != 0
      if (common && typeof common.code === 'number' && common.code !== 0) {
        const fakeError: any = new Error(common.message || 'Request failed');
        fakeError.response = response;
        fakeError.config = response.config;

        const message = getApiErrorMessage(fakeError);
        fakeError.userMessage = message;

        const config: any = response.config || {};
        const silent = config.__silent || config.suppressToast;
        if (typeof window !== 'undefined' && !silent && message) {
          // Check if message contains URLs and render as clickable links
          if (hasUrls(message)) {
            toast.error(renderErrorMessageWithLinks(message));
          } else {
            toast.error(message);
          }
        }

        return Promise.reject(fakeError);
      }
    } catch {
      // 如果解析失败，不影响正常流程
    }

    return response;
  },
  (error) => {
    const status = error?.response?.status;
    const config: any = error?.config || {};

    if (status === 401) {
      if (typeof window !== 'undefined') {
        // Only clear auth if the failed request used the current token (prevent race with re-login)
        const failedToken = error?.config?.headers?.Authorization;
        const currentToken = readAuthTokenFromStorage();
        const tokenStillCurrent = failedToken === `Bearer ${currentToken}`;

        if (tokenStillCurrent || !currentToken) {
          clearAuthState();
          window.dispatchEvent(new CustomEvent('kimbap:session-expired'));
          setTimeout(() => {
            if (window.location.pathname.startsWith('/dashboard')) {
              window.location.href = '/login';
            }
          }, 100);
        }
      }
    }

    // 解析后端返回的友好错误文案，挂到 error 上，方便前端直接使用
    const message = getApiErrorMessage(error);
    (error as any).userMessage = message;

    // 如果未显式关闭，则在前端统一弹出错误提示
    const silent = config.__silent || config.suppressToast;
    if (typeof window !== 'undefined' && !silent && message) {
      // Check if message contains URLs and render as clickable links
      if (hasUrls(message)) {
        toast.error(renderErrorMessageWithLinks(message));
      } else {
        toast.error(message);
      }
    }

    return Promise.reject(error);
  },
);

// API functions
export const api = {
  // Auth
  auth: {
    // Login with access token or master password using protocol 10015
    login: (params: { accessToken?: string; masterPwd?: string }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 10015,
        },
        params,
      }),
  },

  servers: {
    getInfo: () =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10002 },
        params: {},
      }),

    getDashboardOverview: (params?: { timeRange?: string }) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10023 },
        params: params || {},
      }),

  },

  // Usage
  usage: {
    // Usage Overview Statistics (22001-22004)
    // 22001 - Get usage overview summary
    getOverviewSummary: (params: {
      timeRange: number; // days (1, 7, 30, etc.)
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 22001,
        },
        params,
      }),

    // 22002 - Get top tools by usage
    getTopTools: (params: {
      timeRange: number; // days
      limit?: number; // default 10
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 22002,
        },
        params,
      }),

    // 22003 - Get active tokens overview
    getActiveTokens: (params: {
      timeRange: number; // days
      limit?: number; // default 10
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 22003,
        },
        params,
      }),

    // 22004 - Get recent activity
    getRecentActivity: (params: {
      limit?: number; // default 10
      timeRange?: number;
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 22004,
        },
        params,
      }),

    // Tool Usage Statistics (20001-20010)
    // 20001 - Get tool usage summary
    getToolUsageSummary: (params: {
      timeRange: number; // days
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20001,
        },
        params,
      }),

    // 20002 - Get tool detailed metrics
    getToolMetrics: (params: {
      timeRange: number; // days
      page?: number;
      pageSize?: number;
      toolIds?: string[]; // specific tool IDs to query
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20002,
        },
        params,
      }),

    // 20003 - Get tool usage trends
    getToolTrends: (params: {
      timeRange: number; // days
      granularity?: number; // 1-hourly, 2-daily, 3-weekly
      toolIds?: string[]; // empty for all tools
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20003,
        },
        params,
      }),

    // 20004 - Get tool error analysis
    getToolErrors: (params: {
      timeRange: number; // days
      toolIds?: string[];
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20004,
        },
        params,
      }),

    // 20005 - Get tool usage distribution
    getToolDistribution: (params: {
      timeRange: number; // days
      metricType: number; // 1-by requests, 2-by users, 3-by response time
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20005,
        },
        params,
      }),

    // 20006 - Get tool performance comparison
    getToolComparison: (params: {
      timeRange: number; // days
      metricType: number; // 1-response time, 2-success rate, 3-request volume
      toolIds?: string[];
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20006,
        },
        params,
      }),

    // 20007 - Get user tool usage
    getUserToolUsage: (params: {
      timeRange: number; // days
      userIds?: string[]; // empty for all users
      page?: number;
      pageSize?: number;
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20007,
        },
        params,
      }),

    // 20008 - Get tool realtime status
    getToolRealtimeStatus: (params: {
      toolIds?: string[]; // empty for all tools
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20008,
        },
        params,
      }),

    // 20009 - Export usage report
    exportUsageReport: (params: {
      timeRange: number; // days
      format: number; // 1-CSV, 2-JSON, 3-PDF
      toolIds?: string[];
      includeDetails?: boolean;
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20009,
        },
        params,
      }),

    // 20010 - Get tool operation logs
    getToolOperationLogs: (params: {
      timeRange: number; // days
      toolIds?: string[];
      actionTypes?: number[]; // specific event types
      status?: number;
      page?: number;
      pageSize?: number;
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 20010,
        },
        params,
      }),

    // Token Usage Statistics (21001-21006)
    // 21001 - Get token usage summary
    getTokenUsageSummary: (params: {
      timeRange: number; // days
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21001,
        },
        params,
      }),

    // 21002 - Get token detailed metrics
    getTokenMetrics: (params: {
      timeRange: number; // days
      page?: number;
      pageSize?: number;
      tokenIds?: string[]; // specific token IDs to query
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21002,
        },
        params,
      }),

    // 21003 - Get token usage trends
    getTokenTrends: (params: {
      timeRange: number; // days
      granularity?: number; // 1-hourly, 2-daily, 3-weekly
      tokenIds?: string[]; // empty for all tokens
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21003,
        },
        params,
      }),

    // 21004 - Get token geographic distribution
    getTokenGeoDistribution: (params: {
      timeRange: number; // days
      tokenIds?: string[];
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21004,
        },
        params,
      }),

    // 21005 - Get token usage patterns
    getTokenUsagePatterns: (params: {
      tokenId: string;
      patternType: number; // 1-last 60 minutes, 2-last 24 hours, 3-last 7 days by hour
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21005,
        },
        params,
      }),

    // 21006 - Get token usage distribution
    getTokenDistribution: (params: {
      timeRange: number; // days
      metricType: number; // 1-by requests, 2-by clients, 3-by success rate
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21006,
        },
        params,
      }),

    // 21011 - Get recent log records (real-time)
    getRecentLogs: (params: {
      limit?: number; // default 50
      lastId?: number; // for incremental updates
      timeRange?: number;
      userIds?: string[];
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 21011,
        },
        params,
      }),
  },

  // Logs
  logs: {
    list: (params?: { serverId?: string; level?: string; limit?: number; offset?: number }) =>
      apiClient.get('/api/logs', { params }),

    create: (data: { level: string; message: string; metadata?: any }) =>
      apiClient.post('/api/logs', data),

    // 23001 - Get logs with filters
    getLogs: (params: {
      page?: number; // 分页-页码，从1开始
      pageSize?: number; // 分页-每页数量，默认50
      timeRange?: string; // 时间范围: "1h", "6h", "24h", "7d", "all"
      level?: string; // 日志级别: "all", "INFO", "WARN", "ERROR", "DEBUG"
      source?: string; // 日志来源: "all", "api-gateway", "tool-manager", etc.
      search?: string; // 搜索关键词（搜索message和rawData）
      requestId?: string; // 请求ID过滤
      userId?: string; // 用户ID过滤
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 23001,
        },
        params,
      }),

    exportLogs: (params: {
      timeRange?: string;
      level?: string;
      source?: string;
      search?: string;
      format?: number;
      maxRecords?: number;
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 23004,
        },
        params,
      }),

    // 23002 - Get log statistics
    getStatistics: (params: { timeRange?: string }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 23002,
        },
        params,
      }),

    getRealtimeLogs: (params: {
      lastLogId?: number;
      level?: string;
      source?: string;
      limit?: number;
    }) =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 23005,
        },
        params,
      }),
  },

  // Dashboard
  dashboard: {
    // Get dashboard overview statistics using protocol 10023
    overview: (_serverId: string, timeRange: string = '30d') =>
      apiClient.post('/api/v1', {
        common: {
          cmdId: 10023,
        },
        params: {
          timeRange,
        },
      }),
  },

  // Policies
  policies: {
    list: (serverId?: string) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10054 },
        params: { serverId },
      }),

    get: (id: string) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10051 },
        params: { id },
      }),

    create: (params: { serverId?: string; dsl: { schemaVersion: 1; rules: any[] } }) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10050 },
        params,
      }),

    update: (params: { id: string; dsl?: { schemaVersion?: 1; rules: any[] }; status?: string }) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10052 },
        params,
      }),

    delete: (id: string) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10053 },
        params: { id },
      }),

    getEffective: (serverId?: string) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10059 },
        params: { serverId },
      }),
  },

  // Approvals
  approvals: {
    list: (params?: {
      userId?: string;
      serverId?: string;
      toolName?: string;
      status?: string;
      page?: number;
      pageSize?: number;
    }) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10055 },
        params: params || {},
      }),

    get: (id: string) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10056 },
        params: { id },
      }),

    decide: (params: { id: string; decision: 'APPROVED' | 'REJECTED'; reason?: string }) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10057 },
        params,
      }),

    countPending: (userId?: string) =>
      apiClient.post('/api/v1', {
        common: { cmdId: 10058 },
        params: { userId },
      }),
  },

};

// Utility functions
export const setAuthToken = (token: string) => {
  if (typeof window === 'undefined') {
    return;
  }
  localStorage.setItem(AUTH_TOKEN_KEY, token);
  localStorage.removeItem(LEGACY_AUTH_TOKEN_KEY);
};

export const clearAuthState = () => {
  if (typeof window === 'undefined') {
    return;
  }
  localStorage.removeItem(AUTH_TOKEN_KEY);
  localStorage.removeItem(LEGACY_AUTH_TOKEN_KEY);
  localStorage.removeItem('userid');
  localStorage.removeItem('token');
  localStorage.removeItem('manualAccessToken');
  localStorage.removeItem('selectedServer');
  document.cookie = 'kimbap_session=; path=/; max-age=0';
  import('@/lib/crypto')
    .then(({ MasterPasswordManager }) => MasterPasswordManager.clearCache())
    .catch(() => {});
};

export const getAuthToken = () => {
  return readAuthTokenFromStorage();
};

export default apiClient;
