/**
 * Proxy API utility functions for managing MCP servers
 * Based on docs/proxy_api.md specification
 */

import axios from 'axios';
import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { validateAndCacheMcpGatewayUrl } from '@/lib/mcp-gateway-cache';

/**
 * Format unknown error to a readable string for logging (avoids [object Object] in logs)
 */
function formatErrorForLog(error: unknown): string {
  if (error instanceof Error) {
    let s = error.message;
    if (error.stack) s += `\n${error.stack}`;
    if (axios.isAxiosError(error)) {
      if (error.code) s += ` [code: ${error.code}]`;
      if (error.response?.status) s += ` [status: ${error.response.status}]`;
    }
    return s;
  }
  if (typeof error === 'object' && error !== null) {
    try {
      return JSON.stringify(error);
    } catch {
      return String(error);
    }
  }
  return String(error);
}

function redactHeaders(headers: Record<string, unknown>): Record<string, unknown> {
  const redacted: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(headers)) {
    if (/authorization|token|cookie|x-api-key/i.test(key)) {
      redacted[key] = '[REDACTED]';
      continue;
    }
    redacted[key] = value;
  }
  return redacted;
}

function logProxyRequest(actionName: string, action: number, url: string, withToken: boolean): void {
  console.log(
    `[PROXY API] Request start - Action: ${actionName} (${action}), URL: ${url}, Auth: ${withToken ? 'token' : 'none'}`,
  );
}

function logProxyResponse(actionName: string, action: number, duration: number, success: boolean, status?: number): void {
  console.log(
    `[PROXY API] Request done - Action: ${actionName} (${action}), Duration: ${duration}ms, Status: ${status ?? 'n/a'}, Success: ${success}`,
  );
}

function logProxyFailure(actionName: string, action: number, url: string, duration: number, error: unknown): void {
  if (axios.isAxiosError(error)) {
    const safeHeaders = redactHeaders((error.config?.headers || {}) as Record<string, unknown>);
    console.error(
      `[PROXY API] Request failed - Action: ${actionName} (${action}), URL: ${url}, Duration: ${duration}ms, Status: ${error.response?.status ?? 'n/a'}, Code: ${error.code ?? 'n/a'}, Headers: ${JSON.stringify(safeHeaders)}, Error: ${formatErrorForLog(error)}`,
    );
    return;
  }

  console.error(
    `[PROXY API] Request failed - Action: ${actionName} (${action}), URL: ${url}, Duration: ${duration}ms, Error: ${formatErrorForLog(error)}`,
  );
}

// Cache for validated proxy admin URL with TTL (Time To Live)
// - Caches the complete validated URL (with /admin suffix)
// - 5 minutes TTL to balance performance and configuration changes
// - Cache is invalidated on service restart or manual configuration update
let cachedProxyAdminUrl: string | null = null;
let cacheTimestamp: number = 0;
const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

/**
 * Invalidate the cached proxy admin URL
 * Should be called when configuration is manually updated via protocol-10022 or protocol-10021
 */
export function invalidateProxyAdminUrlCache(): void {
  if (cachedProxyAdminUrl) {
    console.log(`[PROXY API] Cache invalidated: ${cachedProxyAdminUrl}`);
  }
  cachedProxyAdminUrl = null;
  cacheTimestamp = 0;
}

// Admin action types as defined in proxy_api.md
enum AdminActionType {
  // User operations (1000-1999)
  DISABLE_USER = 1001,
  CREATE_USER = 1010,
  GET_USERS = 1011,
  UPDATE_USER = 1012,
  DELETE_USER = 1013,
  COUNT_USERS = 1015,
  GET_OWNER = 1016, // Get Owner information

  // Server operations (2000-2999)
  START_SERVER = 2001,
  STOP_SERVER = 2002,
  UPDATE_SERVER_CAPABILITIES = 2003,
  UPDATE_SERVER_LAUNCH_CMD = 2004,
  CONNECT_ALL_SERVERS = 2005,
  CREATE_SERVER = 2010,
  GET_SERVERS = 2011,
  UPDATE_SERVER = 2012,
  DELETE_SERVER = 2013,
  COUNT_SERVERS = 2015,

  // Query operations (3000-3999)
  GET_AVAILABLE_SERVERS_CAPABILITIES = 3002,
  GET_USER_AVAILABLE_SERVERS_CAPABILITIES = 3003,
  GET_SERVERS_STATUS = 3004,
  GET_SERVERS_CAPABILITIES = 3005,

  // Proxy operations (5000-5999)
  GET_PROXY = 5001,
  CREATE_PROXY = 5002,
  UPDATE_PROXY = 5003,
  DELETE_PROXY = 5004,

  // Log operations (7000-7099)
  GET_LOGS = 7002,

  // Policy operations (9101-9105)
  CREATE_TOOL_POLICY = 9101,
  GET_TOOL_POLICY = 9102,
  UPDATE_TOOL_POLICY = 9103,
  DELETE_TOOL_POLICY = 9104,
  GET_EFFECTIVE_POLICY = 9105,

  // Approval operations (9201-9204)
  LIST_APPROVAL_REQUESTS = 9201,
  GET_APPROVAL_REQUEST = 9202,
  DECIDE_APPROVAL_REQUEST = 9203,
  COUNT_PENDING_APPROVALS = 9204,
}

// Request and response interfaces
interface AdminRequest<T = any> {
  action: AdminActionType;
  data: T;
}

interface AdminResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: number;
    message: string;
  };
}

// Server status enum
export enum ServerStatus {
  Online = 0,
  Offline = 1,
  Connecting = 2,
  Error = 3,
  Sleeping = 4, // Sleeping (lazy start)
}

/**
 * Get Kimbap Core configuration from database
 * Returns the host and port stored in config table
 * Throws ApiError if no configuration is found
 */
async function getKimbapCoreConfig(): Promise<{ host: string; port: number | null }> {
  try {
    const dbConfig = await prisma.config.findFirst();

    if (dbConfig && dbConfig.kimbap_core_host) {
      const host = dbConfig.kimbap_core_host;
      const port: number | undefined = dbConfig.kimbap_core_port || undefined;

      const displayStr = port === 443 || !port ? host : `${host}:${port}`;
      console.log(`[PROXY API] Using Kimbap Core config from database: ${displayStr}`);
      return {
        host: host,
        port: port === 443 ? null : port || null,
      };
    }

    // No configuration found in database
    console.error('[PROXY API] No Kimbap Core configuration found in database');
    throw new ApiError(ErrorCode.KIMBAP_CORE_CONFIG_NOT_FOUND, 500);
  } catch (error) {
    console.error('[PROXY API] Failed to get Kimbap Core config from database:', error);

    // If it's already an ApiError, re-throw it
    if (error instanceof ApiError) {
      throw error;
    }

    // Log the actual error for debugging
    if (error instanceof Error) {
      console.error('[PROXY API] Error details:', error.message);
      console.error('[PROXY API] Error stack:', error.stack);
    }

    // Otherwise, throw generic error
    throw new ApiError(ErrorCode.KIMBAP_CORE_CONFIG_NOT_FOUND, 500);
  }
}

/**
 * Get proxy admin API URL
 * New 3-tier priority system:
 * 1. Database config (user manual configuration via protocol-10022)
 * 2. KIMBAP_CORE_URL environment variable
 * 3. No config found - throw error (auto-detection is handled by protocol-10021)
 *
 * Host from database is expected to already contain protocol and be normalized
 * (host.docker.internal → kimbap-core normalization happens at save time)
 */
async function getProxyAdminUrl(): Promise<string> {
  // Check cache first (5 minutes TTL)
  if (cachedProxyAdminUrl && Date.now() - cacheTimestamp < CACHE_TTL_MS) {
    console.log(
      `[PROXY API] Using cached proxy admin URL: ${cachedProxyAdminUrl} (age: ${Math.floor((Date.now() - cacheTimestamp) / 1000)}s)`,
    );
    return cachedProxyAdminUrl;
  }

  try {
    // Priority 1: Try database config first
    const { host, port } = await getKimbapCoreConfig();

    // Database config found - build URL and return
    if (host.startsWith('http://') || host.startsWith('https://')) {
      // Host contains protocol (normal case)
      const isHttps = host.startsWith('https://');
      const defaultPort = isHttps ? 443 : 80;
      const shouldAppendPort = port && port !== defaultPort;

      const baseUrl = shouldAppendPort ? `${host}:${port}` : host;
      const adminUrl = `${baseUrl}/admin`;
      console.log(`[PROXY API] Using database config: ${adminUrl}`);
      return adminUrl;
    } else {
      // Fallback for legacy data without protocol (shouldn't happen with new normalization)
      const isIP = /^(\d{1,3}\.){3}\d{1,3}$/.test(host);
      const isLocalhost = host === 'localhost';
      const isDockerServiceName = !host.includes('.') && !isIP && !isLocalhost;
      const protocol = isIP || isLocalhost || isDockerServiceName ? 'http' : 'https';
      const defaultPort = protocol === 'https' ? 443 : 80;
      const shouldAppendPort = port && port !== defaultPort;

      const baseUrl = shouldAppendPort ? `${protocol}://${host}:${port}` : `${protocol}://${host}`;
      const adminUrl = `${baseUrl}/admin`;
      console.log(`[PROXY API] Using database config (added protocol): ${adminUrl}`);
      return adminUrl;
    }
  } catch (error) {
    // Database config not found or error, try Priority 2
    if (!(error instanceof ApiError && error.code === ErrorCode.KIMBAP_CORE_CONFIG_NOT_FOUND)) {
      // If it's not a "config not found" error, re-throw (unexpected error)
      throw error;
    }

    // Priority 2: Try KIMBAP_CORE_URL environment variable
    const kimbapCoreUrl = process.env.KIMBAP_CORE_URL?.trim();
    const normalizedKimbapCoreUrl = kimbapCoreUrl?.replace(/\/+$/, '');
    if (normalizedKimbapCoreUrl) {
      console.log(`[PROXY API] No database config found, trying KIMBAP_CORE_URL: ${normalizedKimbapCoreUrl}`);

      // Validate and cache the URL before constructing /admin
      const validation = await validateAndCacheMcpGatewayUrl(normalizedKimbapCoreUrl);

      if (validation.isValid && validation.host && validation.port) {
        // Build admin URL from normalized URL
        const adminUrl = `${normalizedKimbapCoreUrl}/admin`;
        console.log(`[PROXY API] Using KIMBAP_CORE_URL (validated): ${adminUrl}`);
        return adminUrl;
      } else {
        // Validation failed, log warning and fall through to error
        console.warn(`[PROXY API] KIMBAP_CORE_URL validation failed: ${validation.errorMessage}`);
        console.warn(`[PROXY API] Continuing to auto-detection fallback...`);
      }
    }

    // Priority 3: No config found anywhere - throw error
    console.error(
      `[PROXY API] No Kimbap Core configuration found (database empty, KIMBAP_CORE_URL invalid/missing)`,
    );
    console.error(
      `[PROXY API] Please configure Kimbap Core via protocol-10021 (auto-detection) or protocol-10022 (manual config)`,
    );
    throw new ApiError(ErrorCode.KIMBAP_CORE_CONFIG_NOT_FOUND, 500);
  }
}

// Actions that don't require token
const NO_TOKEN_ACTIONS = [
  AdminActionType.GET_PROXY, // 5001
  AdminActionType.CREATE_PROXY, // 5002
  AdminActionType.CREATE_USER, // 1010
  AdminActionType.COUNT_USERS, // 1015 - Count operations are system-level
  AdminActionType.COUNT_SERVERS, // 2015 - Count operations are system-level
];

/**
 * Make a request to the proxy admin API with automatic token resolution
 * @param action - The admin action type
 * @param data - The request data
 * @param userid - User ID to get token from local database (for actions that require token)
 * @param overrideToken - Optional token to override auto-resolution
 * @param timeout - Optional timeout in milliseconds (default: 30000ms)
 */
async function makeProxyRequestWithUserId<T = any>(
  action: AdminActionType,
  data: any,
  userid?: string,
  overrideToken?: string,
  timeout: number = 30000,
): Promise<AdminResponse<T>> {
  let token = overrideToken;

  // Check if this action requires a token
  if (!NO_TOKEN_ACTIONS.includes(action)) {
    if (!token) {
      if (userid) {
        throw new Error(`Token required for action: ${AdminActionType[action]} (userid: ${userid})`);
      }
      throw new Error(`Token required for action: ${AdminActionType[action]}`);
    }
  }

  return makeProxyRequest(action, data, token, timeout);
}

/**
 * Make a request to the proxy admin API
 * @param action - The admin action type
 * @param data - The request data
 * @param token - Optional token to override the default token (required for START_SERVER and UPDATE_SERVER_LAUNCH_CMD)
 * @param timeout - Optional timeout in milliseconds (default: 30000ms)
 */
async function makeProxyRequest<T = any>(
  action: AdminActionType,
  data: any,
  token?: string,
  timeout: number = 30000,
): Promise<AdminResponse<T>> {
  const url = await getProxyAdminUrl();
  const authToken = token || '';
  const actionName = AdminActionType[action];
  const startTime = Date.now();

  logProxyRequest(actionName, action, url, Boolean(token));

  try {
    const response = await axios.post<AdminResponse<T>>(
      url,
      {
        action,
        data,
      } as AdminRequest,
      {
        headers: {
          'Content-Type': 'application/json',
          ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
        },
        timeout: timeout,
      },
    );

    const duration = Date.now() - startTime;
    logProxyResponse(actionName, action, duration, response.data.success, response.status);

    // Cache the validated URL on successful request
    if (response.data.success) {
      cachedProxyAdminUrl = url;
      cacheTimestamp = Date.now();
      console.log(`[PROXY API] Cached validated URL: ${url}`);
    }

    return response.data;
  } catch (error) {
    const duration = Date.now() - startTime;
    logProxyFailure(actionName, action, url, duration, error);

    if (axios.isAxiosError(error)) {
      const isConnectionError =
        error.code === 'ECONNREFUSED' || error.code === 'ENOTFOUND' || error.code === 'ETIMEDOUT';

      if (isConnectionError && url.includes('host.docker.internal')) {
        console.error(
          `[PROXY API] Connection failed to host.docker.internal. If Kimbap Core is running in Docker, try using Docker service name (e.g., kimbap-core) instead.`,
        );
        console.error(
          `[PROXY API] You can set KIMBAP_CORE_URL environment variable to override, e.g., KIMBAP_CORE_URL=http://kimbap-core:3002`,
        );
      }

      if (error.response?.data) {
        return error.response.data;
      }
      throw new Error(`Proxy API request failed: ${error.message}`);
    }
    throw error;
  }
}

interface ToolCapability { enabled: boolean; dangerLevel?: number; description?: string }

interface ServerCapabilities {
  tools: Record<string, ToolCapability>;
  resources: Record<string, { enabled: boolean }>;
  prompts: Record<string, { enabled: boolean }>;
}

interface ServerCapabilitiesWithMeta extends ServerCapabilities {
  serverName: string;
  enabled: boolean;
}

interface CoreLogEntry {
  id: number;
  idInCore: bigint | null;
  action: number;
  userid: string;
  serverId: string | null;
  createdAt: number;
  sessionId: string;
  upstreamRequestId: string;
  uniformRequestId: string | null;
  parentUniformRequestId: string | null;
  proxyRequestId: string | null;
  ip: string;
  ua: string;
  tokenMask: string;
  requestParams: string;
  responseResult: string;
  error: string;
  duration: number | null;
  statusCode: number | null;
  proxyKey?: string;
}

/**
 * Dispatch an admin request with 3-way auth resolution:
 *   1. token provided → use directly
 *   2. userid provided → resolve via makeProxyRequestWithUserId
 *   3. neither → unauthenticated call
 */
async function dispatchRequest<T>(
  action: AdminActionType,
  data: any,
  userid?: string,
  token?: string,
  timeout?: number,
): Promise<AdminResponse<T>> {
  if (token) {
    return makeProxyRequest<T>(action, data, token, timeout);
  }
  if (userid) {
    return makeProxyRequestWithUserId<T>(action, data, userid, token, timeout);
  }
  return makeProxyRequest<T>(action, data, undefined, timeout);
}

function unwrapResponse<T>(result: AdminResponse<T>, errorPrefix: string): T {
  if (!result.success) {
    throw new Error(`${errorPrefix}: ${result.error?.message || 'Unknown error'}`);
  }
  return result.data!;
}

/**
 * Start a server on the proxy
 * @param serverId - The ID of the server to start
 * @param token - The authentication token for starting the server
 * @returns Promise that resolves when the server is started
 */
export async function startProxyServer(serverId: string, token: string): Promise<void> {
  console.log(`[PROXY API] startProxyServer called - ServerId: ${serverId}`);

  const result = await makeProxyRequest(
    AdminActionType.START_SERVER,
    {
      targetId: serverId,
    },
    token,
    120000,
  ); // 2 minutes timeout for server startup

  if (!result.success) {
    console.error(
      `[PROXY API] startProxyServer failed - ServerId: ${serverId}, Error:`,
      result.error,
    );
    throw new Error(
      `Failed to start server ${serverId}: ${result.error?.message || 'Unknown error'}`,
    );
  }

  console.log(`[PROXY API] startProxyServer success - ServerId: ${serverId}`);
}

export async function stopMCPServer(
  serverId: string,
  userid?: string,
  token?: string,
): Promise<void> {
  console.log(`[PROXY API] stopMCPServer called - ServerId: ${serverId}`);
  const result = await dispatchRequest(AdminActionType.STOP_SERVER, { targetId: serverId }, userid, token);
  unwrapResponse(result, `Failed to stop server ${serverId}`);
  console.log(`[PROXY API] stopMCPServer success - ServerId: ${serverId}`);
}

/**
 * Get the status of all servers
 * @param userid - User ID (for token retrieval)
 * @returns Promise with server status map
 */
export async function getServersStatus(
  userid?: string,
  token?: string,
): Promise<{ [serverId: string]: ServerStatus }> {
  const result = await dispatchRequest<{ serversStatus: { [serverId: string]: ServerStatus } }>(
    AdminActionType.GET_SERVERS_STATUS, {}, userid, token,
  );
  return unwrapResponse(result, 'Failed to get servers status')?.serversStatus ?? {};
}

/**
 * Connect all servers
 * @param userid - User ID (for token retrieval)
 * @param token - Optional admin token for authorization
 * @returns Promise with success and failed server lists
 */
export async function connectAllServers(
  userid?: string,
  token?: string,
): Promise<{
  successServers: any[];
  failedServers: any[];
}> {
  const result = await dispatchRequest<{ successServers: any[]; failedServers: any[] }>(
    AdminActionType.CONNECT_ALL_SERVERS, {}, userid, token, 120000,
  );
  const data = unwrapResponse(result, 'Failed to connect servers');
  return data || { successServers: [], failedServers: [] };
}

/**
 * Disable a user
 * @param userId - The ID of the user to disable
 * @param requestUserId - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over requestUserId)
 * @returns Promise that resolves when user is disabled
 */
export async function disableUser(
  userId: string,
  requestUserId?: string,
  token?: string,
): Promise<void> {
  const result = await dispatchRequest(AdminActionType.DISABLE_USER, { targetId: userId }, requestUserId, token);
  unwrapResponse(result, `Failed to disable user ${userId}`);
}

/**
 * Get all available servers' capabilities
 * @param userid - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over userid)
 * @returns Promise with all servers' capabilities
 */
export async function getAvailableServersCapabilities(
  userid?: string,
  token?: string,
): Promise<Record<string, ServerCapabilitiesWithMeta>> {
  const result = await dispatchRequest<{ capabilities: Record<string, ServerCapabilitiesWithMeta> }>(
    AdminActionType.GET_AVAILABLE_SERVERS_CAPABILITIES, {}, userid, token,
  );
  return unwrapResponse(result, 'Failed to get available servers capabilities')?.capabilities ?? {};
}

/**
 * Get user-available servers' capabilities
 * @param userId - The ID of the user to get capabilities for
 * @param requestUserId - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over requestUserId)
 * @returns Promise with user-available servers' capabilities
 */
export async function getUserAvailableServersCapabilities(
  userId: string,
  requestUserId?: string,
  token?: string,
): Promise<Record<string, ServerCapabilities & { enabled: boolean }>> {
  const result = await dispatchRequest<{ capabilities: Record<string, ServerCapabilities & { enabled: boolean }> }>(
    AdminActionType.GET_USER_AVAILABLE_SERVERS_CAPABILITIES, { targetId: userId }, requestUserId, token,
  );
  return unwrapResponse(result, `Failed to get user available servers capabilities for ${userId}`)?.capabilities ?? {};
}

/**
 * Get capabilities for a specific server
 * @param serverId - The ID of the server to get capabilities for
 * @param userid - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over userid)
 * @returns Promise with server capabilities
 */
export async function getServersCapabilities(
  serverId: string,
  userid?: string,
  token?: string,
): Promise<ServerCapabilities> {
  const result = await dispatchRequest<{ capabilities: ServerCapabilities }>(
    AdminActionType.GET_SERVERS_CAPABILITIES, { targetId: serverId }, userid, token,
  );
  return unwrapResponse(result, `Failed to get capabilities for server ${serverId}`)?.capabilities
    ?? { tools: {}, resources: {}, prompts: {} };
}

// ============================================================================
// DATABASE API FUNCTIONS
// ============================================================================

// -------------------- PROXY OPERATIONS --------------------

/**
 * Create proxy server
 * @param data - Proxy server configuration information
 * @returns Promise with created proxy information
 */
export async function createProxy(data: {
  name: string;
  proxyKey: string;
  startPort: number;
}): Promise<{ id: number; name: string; proxyKey: string; addtime: number; startPort: number }> {
  const result = await makeProxyRequest<{ proxy: any }>(AdminActionType.CREATE_PROXY, data);

  if (!result.success || !result.data) {
    throw new Error(`Failed to create proxy: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data!.proxy;
}

/**
 * Query proxy server information
 * @returns Promise with proxy information
 */
export async function getProxy(): Promise<{
  id: number;
  name: string;
  proxyKey: string;
  addtime: number;
  startPort: number;
}> {
  const result = await makeProxyRequest<{ proxy: any }>(AdminActionType.GET_PROXY, {});

  if (!result.success) {
    throw new Error(`Failed to get proxy: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data!.proxy;
}

/**
 * Update proxy server information
 * @param proxyId - Proxy server ID
 * @param data - Update data
 * @param userid - User ID (for token retrieval)
 * @returns Promise with updated proxy information
 */
export async function updateProxy(
  proxyId: number,
  data: { name: string },
  userid?: string,
  token?: string,
): Promise<{ proxy: any }> {
  const result = await dispatchRequest<{ proxy: any }>(
    AdminActionType.UPDATE_PROXY, { proxyId, ...data }, userid, token,
  );
  invalidateProxyAdminUrlCache();
  return unwrapResponse(result, 'Failed to update proxy');
}

/**
 * Delete proxy server
 * @param proxyId - Proxy server ID
 * @param userid - User ID (for token retrieval)
 * @returns Promise that resolves when proxy is deleted
 */
export async function deleteProxy(
  proxyId: number,
  userid: string,
  token?: string,
): Promise<{ message: string }> {
  const result = token
    ? await makeProxyRequest<{ message: string }>(AdminActionType.DELETE_PROXY, { proxyId }, token)
    : await makeProxyRequestWithUserId<{ message: string }>(AdminActionType.DELETE_PROXY, { proxyId }, userid, token);
  return unwrapResponse(result, 'Failed to delete proxy');
}

// -------------------- USER OPERATIONS --------------------

/**
 * Create user
 * @param data - User information
 * @returns Promise with created user information
 */
export async function createUser(
  data: {
    userId: string;
    status: number;
    role: number;
    permissions: object | string;
    serverApiKeys: string[] | string;
    ratelimit: number;
    name: string;
    encryptedToken: string;
    proxyId: number;
    notes?: string;
    expiresAt?: number;
  },
  token?: string,
): Promise<{ user: any }> {
  const result = await makeProxyRequest<{ user: any }>(AdminActionType.CREATE_USER, data, token);

  if (!result.success) {
    throw new Error(`Failed to create user: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data!;
}

/**
 * Query user list
 * @param filters - Filter conditions (optional)
 * @param userid - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over userid)
 * @returns Promise with array of users
 */
export async function getUsers(
  filters?: {
    proxyId?: number;
    role?: number;
    excludeRole?: number;
    userId?: string;
    includePermissions?: boolean;
  },
  userid?: string,
  token?: string,
): Promise<{ users: any[] }> {
  const result = await dispatchRequest<{ users: any[] }>(
    AdminActionType.GET_USERS, filters || {}, userid, token,
  );
  return unwrapResponse(result, 'Failed to get users');
}

/**
 * Get user information by accessToken directly
 * Used for authentication without relying on local cache
 * @param userId - The user ID to query
 * @param accessToken - The access token for authentication
 * @returns Promise with user information
 */
export async function getUserByAccessToken(userId: string, accessToken: string): Promise<any> {
  const result = await makeProxyRequest<{ users: any[] }>(
    AdminActionType.GET_USERS,
    { userId: userId },
    accessToken,
  );

  if (!result.success) {
    throw new Error(`Failed to authenticate user: ${result.error?.message || 'Unknown error'}`);
  }

  const user = result.data?.users?.find((u: any) => u.userId === userId);
  if (!user) {
    throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, {
      details: 'User not found',
    });
  }

  return user;
}

/**
 * Get Owner information
 * Get complete information of system Owner user without authentication
 * @returns Promise with owner information
 */
export async function getOwner(): Promise<{ owner: any }> {
  const result = await makeProxyRequest<{ owner: any }>(AdminActionType.GET_OWNER, {});

  if (!result.success) {
    // Check if error code is USER_NOT_FOUND
    if (result.error?.code === 2001) {
      throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, {
        details: 'Owner account not found in the system',
      });
    }
    throw new Error(`Failed to get owner: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data!;
}

/**
 * Update user information
 * @param userId - User ID
 * @param data - Update data
 * @param requestUserId - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over requestUserId)
 * @returns Promise with updated user information
 */
export async function updateUser(
  userId: string,
  data: {
    name?: string;
    notes?: string;
    permissions?: object | string;
    serverApiKeys?: string[];
    status?: number;
    encryptedToken?: string;
  },
  requestUserId?: string,
  token?: string,
): Promise<{ user: any }> {
  const result = await dispatchRequest<{ user: any }>(
    AdminActionType.UPDATE_USER, { userId, ...data }, requestUserId, token,
  );
  return unwrapResponse(result, 'Failed to update user');
}

/**
 * Delete user
 * @param userId - User ID
 * @param requestUserId - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over requestUserId)
 * @returns Promise that resolves when user is deleted
 */
export async function deleteUser(
  userId: string,
  requestUserId?: string,
  token?: string,
): Promise<{ message: string }> {
  const result = await dispatchRequest<{ message: string }>(
    AdminActionType.DELETE_USER, { userId }, requestUserId, token,
  );
  return unwrapResponse(result, 'Failed to delete user');
}

/**
 * Count users
 * @param filters - Filter conditions (optional)
 * @returns Promise with user count
 */
export async function countUsers(
  filters?: {
    excludeRole?: number;
  },
  requestUserId?: string,
): Promise<{ count: number }> {
  const result = await dispatchRequest<{ count: number }>(
    AdminActionType.COUNT_USERS, filters || {}, requestUserId,
  );
  return unwrapResponse(result, 'Failed to count users');
}

// -------------------- SERVER OPERATIONS --------------------

/**
 * Create MCP server
 * @param data - Server configuration information
 * @param requestUserId - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over requestUserId)
 * @returns Promise with created server information
 */
export async function createServer(
  data: {
    serverId: string;
    serverName: string;
    enabled: boolean;
    launchConfig: string;
    capabilities: object;
    allowUserInput: boolean;
    proxyId: number;
    toolTmplId: string;
    authType: number;
    configTemplate?: string;
    category?: number;
    toolDefaultConfig?: string;
    oAuthConfig?: string;
    lazyStartEnabled?: boolean;
    publicAccess?: boolean;
    anonymousAccess?: boolean;
    anonymousRateLimit?: number;
  },
  requestUserId?: string,
  token?: string,
): Promise<{ server: any }> {
  const result = await dispatchRequest<{ server: any }>(
    AdminActionType.CREATE_SERVER, data, requestUserId, token,
  );
  return unwrapResponse(result, 'Failed to create server');
}

/**
 * Query server list
 * @param filters - Filter conditions (optional)
 * @param userid - User ID for token lookup from local database (optional)
 * @param token - Access token to use directly (optional, takes precedence over userid)
 * @returns Promise with array of servers
 */
export async function getServers(
  filters?: {
    proxyId?: number;
    enabled?: boolean;
    serverId?: string;
  },
  userid?: string,
  token?: string,
): Promise<{ servers: any[] }> {
  const result = await dispatchRequest<{ servers: any[] }>(
    AdminActionType.GET_SERVERS, filters || {}, userid, token,
  );
  return unwrapResponse(result, 'Failed to get servers');
}

/**
 * Update server configuration
 * @param serverId - Server ID
 * @param data - Update data
 * @param userid - User ID (for token retrieval)
 * @returns Promise with updated server information
 */
export async function updateServer(
  serverId: string,
  data: {
    serverName?: string;
    launchConfig?: object | string;
    capabilities?: object | string;
    configTemplate?: string;
    enabled?: boolean;
    allowUserInput?: boolean;
    lazyStartEnabled?: boolean;
    publicAccess?: boolean;
    anonymousAccess?: boolean;
    anonymousRateLimit?: number;
  },
  userid?: string,
  token?: string,
): Promise<{ server: any }> {
  const result = await dispatchRequest<{ server: any }>(
    AdminActionType.UPDATE_SERVER, { serverId, ...data }, userid, token,
  );
  return unwrapResponse(result, 'Failed to update server');
}

/**
 * Delete server
 * @param serverId - Server ID
 * @param userid - User ID (for token retrieval)
 * @returns Promise that resolves when server is deleted
 */
export async function deleteServer(
  serverId: string,
  userid?: string,
  token?: string,
): Promise<{ message: string }> {
  const result = await dispatchRequest<{ message: string }>(
    AdminActionType.DELETE_SERVER, { serverId }, userid, token,
  );
  return unwrapResponse(result, 'Failed to delete server');
}

/**
 * Count servers
 * @param filters - Filter conditions (optional)
 * @param userid - User ID (for token retrieval)
 * @param token - Optional token for direct authentication
 * @returns Promise with server count
 */
export async function countServers(
  filters?: {
    proxyId?: number;
  },
  userid?: string,
  token?: string,
): Promise<{ count: number }> {
  const result = await dispatchRequest<{ count: number }>(
    AdminActionType.COUNT_SERVERS, filters || {}, userid, token,
  );
  return unwrapResponse(result, 'Failed to count servers');
}

// -------------------- LOG OPERATIONS --------------------

/**
 * Get log records (Owner only)
 * @param params - Query parameters
 * @param userid - User ID (for token retrieval)
 * @returns Promise with logs data
 */
export async function getLogs(
  params?: {
    id?: number;
    limit?: number;
  },
  userid?: string,
  token?: string,
): Promise<{
  logs: CoreLogEntry[];
  count: number;
  startId: number;
  limit: number;
}> {
  const result = await dispatchRequest<{ logs: CoreLogEntry[]; count: number; startId: number; limit: number }>(
    AdminActionType.GET_LOGS, params || {}, userid, token,
  );
  return unwrapResponse(result, 'Failed to get logs');
}

// -------------------- USER API OPERATIONS --------------------

// User action types as defined in USER_API.md
enum UserActionType {
  // Capability configuration operations (1000-1999)
  GET_CAPABILITIES = 1001, // Get user's capability configuration
  SET_CAPABILITIES = 1002, // Set user's capability configuration

  // Server configuration operations (2000-2999)
  CONFIGURE_SERVER = 2001, // Configure a server for user
  UNCONFIGURE_SERVER = 2002, // Unconfigure a server for user

  // Session query operations (3000-3999)
  GET_ONLINE_SESSIONS = 3001, // Get user's online session list
}

// User API request and response interfaces
interface UserRequest<T = any> {
  action: UserActionType;
  data?: T;
}

interface UserResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: number;
    message: string;
  };
}

/**
 * Get proxy user API URL (based on getProxyAdminUrl, but uses /user endpoint)
 */
async function getProxyUserUrl(): Promise<string> {
  const adminUrl = await getProxyAdminUrl();
  // Replace /admin with /user
  return adminUrl.replace(/\/admin$/, '/user');
}

/**
 * Make a request to the proxy user API
 * @param action - The user action type
 * @param data - The request data
 * @param token - Access token for authentication (required)
 * @param timeout - Optional timeout in milliseconds (default: 30000ms)
 */
async function makeProxyUserRequest<T = any>(
  action: UserActionType,
  data: any,
  token: string,
  timeout: number = 30000,
): Promise<UserResponse<T>> {
  const url = await getProxyUserUrl();
  const actionName = UserActionType[action];

  if (!url) {
    throw new Error('Proxy user URL not configured');
  }

  if (!token) {
    throw new Error('Token is required for user API requests');
  }

  const startTime = Date.now();
  logProxyRequest(actionName, action, url, true);

  try {
    const response = await axios.post<UserResponse<T>>(
      url,
      {
        action,
        data,
      } as UserRequest,
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        timeout: timeout,
      },
    );

    const duration = Date.now() - startTime;
    logProxyResponse(actionName, action, duration, response.data.success, response.status);

    return response.data;
  } catch (error) {
    const duration = Date.now() - startTime;
    logProxyFailure(actionName, action, url, duration, error);

    if (axios.isAxiosError(error)) {
      if (error.response?.data) {
        return error.response.data;
      }
      throw new Error(`Proxy User API request failed: ${error.message}`);
    }
    throw error;
  }
}

/**
 * Server capability info returned by GET_CAPABILITIES
 */
export interface ServerCapabilityInfo {
  enabled: boolean;
  serverName: string;
  allowUserInput?: boolean;
  authType?: number;
  category?: number;
  configTemplate?: string;
  configured?: boolean;
  status?: number;
  tools: Record<
    string,
    {
      enabled: boolean;
      description?: string;
      dangerLevel?: number;
    }
  >;
  resources: Record<
    string,
    {
      enabled: boolean;
      description?: string;
    }
  >;
  prompts: Record<
    string,
    {
      enabled: boolean;
      description?: string;
    }
  >;
}

/**
 * Get user capabilities from Kimbap Core
 * Proxies to Core's 1001 GET_CAPABILITIES operation
 * @param token - Access token of the user to get capabilities for
 * @returns Promise with user capabilities (map of serverId to ServerCapabilityInfo)
 */
export async function getUserCapabilities(
  token: string,
): Promise<Record<string, ServerCapabilityInfo>> {
  const result = await makeProxyUserRequest<Record<string, ServerCapabilityInfo>>(
    UserActionType.GET_CAPABILITIES,
    {},
    token,
  );

  if (!result.success) {
    throw new Error(`Failed to get user capabilities: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data || {};
}

/**
 * Input type for SET_CAPABILITIES operation
 */
export interface SetCapabilitiesInput {
  [serverId: string]: {
    enabled?: boolean;
    tools?: {
      [toolName: string]: {
        enabled?: boolean;
        dangerLevel?: 0 | 1 | 2;
      };
    };
    resources?: {
      [resourceName: string]: { enabled?: boolean };
    };
    prompts?: {
      [promptName: string]: { enabled?: boolean };
    };
  };
}

/**
 * Set user capabilities in Kimbap Core
 * Proxies to Core's 1002 SET_CAPABILITIES operation
 * @param capabilities - Partial capabilities configuration to set
 * @param token - Access token of the user to set capabilities for
 * @returns Promise with success message
 */
export async function setUserCapabilities(
  capabilities: SetCapabilitiesInput,
  token: string,
): Promise<{ message: string }> {
  const result = await makeProxyUserRequest<{ message: string }>(
    UserActionType.SET_CAPABILITIES,
    capabilities,
    token,
  );

  if (!result.success) {
    throw new Error(`Failed to set user capabilities: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data || { message: 'Capabilities updated successfully' };
}

/**
 * Auth configuration for Template servers (category=1)
 */
export interface TemplateAuthConf {
  key: string; // Placeholder key from template.credentials
  value: string; // Actual credential value
  dataType: number; // Data type identifier (1 = string replacement)
}

/**
 * Auth configuration for Custom Remote HTTP servers (category=2)
 */
export interface RemoteAuth {
  params?: Record<string, string>; // Query parameters to append to URL
  headers?: Record<string, string>; // HTTP headers to add
}

/**
 * Auth configuration for RestApi servers (category=3)
 */
export interface RestfulApiAuth {
  type: 'bearer' | 'basic' | 'header' | 'query_param';
  value?: string; // Required for bearer, header, query_param
  header?: string; // Required for header type
  param?: string; // Required for query_param type
  username?: string; // Required for basic type
  password?: string; // Required for basic type
}

/**
 * Input type for CONFIGURE_SERVER operation
 */
export interface ConfigureServerInput {
  serverId: string;
  authConf?: TemplateAuthConf[]; // For Template servers (category=1)
  remoteAuth?: RemoteAuth; // For Custom Remote HTTP servers (category=2); Custom Stdio uses category=5
  restfulApiAuth?: RestfulApiAuth; // For RestApi servers (category=3)
}

/**
 * Configure a server for a user in Kimbap Core
 * Proxies to Core's 2001 CONFIGURE_SERVER operation
 * @param config - Server configuration input
 * @param token - Access token of the user
 * @returns Promise with success message
 */
export async function configureUserServer(
  config: ConfigureServerInput,
  token: string,
): Promise<{ serverId: string; message: string }> {
  const result = await makeProxyUserRequest<{ serverId: string; message: string }>(
    UserActionType.CONFIGURE_SERVER,
    config,
    token,
  );

  if (!result.success) {
    // Map Core error codes to meaningful messages
    const errorCode = result.error?.code;
    let errorMessage = result.error?.message || 'Unknown error';

    // Add more context based on error code
    if (errorCode === 2001) {
      errorMessage = `Server not found: ${config.serverId}`;
    } else if (errorCode === 2002) {
      errorMessage = `Server is disabled: ${config.serverId}`;
    } else if (errorCode === 2003) {
      errorMessage = `Invalid server configuration: ${errorMessage}`;
    } else if (errorCode === 2004) {
      errorMessage = `Server does not allow user configuration: ${config.serverId}`;
    } else if (errorCode === 2005) {
      errorMessage = `Server missing configuration template: ${config.serverId}`;
    }

    throw new Error(`Failed to configure server: ${errorMessage}`);
  }

  return result.data || { serverId: config.serverId, message: 'Server configured successfully' };
}

/**
 * Unconfigure a server for a user in Kimbap Core
 * Proxies to Core's 2002 UNCONFIGURE_SERVER operation
 * This is an idempotent operation - safe to call multiple times
 * @param serverId - The server ID to unconfigure
 * @param token - Access token of the user
 * @returns Promise with success message
 */
export async function unconfigureUserServer(
  serverId: string,
  token: string,
): Promise<{ serverId: string; message: string }> {
  const result = await makeProxyUserRequest<{ serverId: string; message: string }>(
    UserActionType.UNCONFIGURE_SERVER,
    { serverId },
    token,
  );

  if (!result.success) {
    throw new Error(`Failed to unconfigure server: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data || { serverId, message: 'Server unconfigured successfully' };
}

/**
 * Session data returned by GET_ONLINE_SESSIONS
 */
export interface SessionData {
  sessionId: string;
  clientName: string;
  userAgent: string;
  lastActive: string; // ISO 8601 timestamp
}

/**
 * Get online sessions for a user from Kimbap Core
 * Proxies to Core's 3001 GET_ONLINE_SESSIONS operation
 * @param token - Access token of the user to get sessions for
 * @returns Promise with array of session data
 */
export async function getOnlineSessions(token: string): Promise<SessionData[]> {
  const result = await makeProxyUserRequest<SessionData[]>(
    UserActionType.GET_ONLINE_SESSIONS,
    {},
    token,
  );

  if (!result.success) {
    throw new Error(`Failed to get online sessions: ${result.error?.message || 'Unknown error'}`);
  }

  return result.data || [];
}

// Export the new function and AdminActionType for use in protocol handlers
export {
  makeProxyRequestWithUserId,
  makeProxyRequest,
  AdminActionType,
  UserActionType,
  makeProxyUserRequest,
};
