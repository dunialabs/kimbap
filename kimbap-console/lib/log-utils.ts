/**
 * Shared log utility functions for domain/outcome classification.
 *
 * Replaces the broken `inferSource()` (treated Int action as string) and
 * the misleading `inferLogLevel()` (only checked statusCode) that were
 * duplicated across protocols 23001-23005.
 *
 * Domain  — derived from the MCPEventLogType numeric action ranges.
 * Level   — derived from error field + action type + statusCode.
 */

// ========== Domain Classification ==========

export type LogDomain =
  | 'mcp-request'
  | 'reverse'
  | 'lifecycle'
  | 'oauth'
  | 'auth'
  | 'error'
  | 'admin'
  | 'system';

export const LOG_DOMAINS: readonly { value: LogDomain; label: string }[] = [
  { value: 'mcp-request', label: 'Tool Request' },
  { value: 'reverse', label: 'Reverse Request' },
  { value: 'lifecycle', label: 'Lifecycle' },
  { value: 'oauth', label: 'OAuth' },
  { value: 'auth', label: 'Auth' },
  { value: 'error', label: 'Internal Error' },
  { value: 'admin', label: 'Admin' },
  { value: 'system', label: 'System' },
];

/**
 * Infer domain from the numeric `action` column (MCPEventLogType enum value).
 * action is Int in the DB — never a string.
 */
export function inferDomain(action: number | null | undefined): LogDomain {
  if (action == null || typeof action !== 'number') return 'system';

  if (action >= 1001 && action <= 1009) return 'mcp-request';
  if (action >= 1201 && action <= 1206) return 'reverse';
  if (action >= 1301 && action <= 1314) return 'lifecycle';
  if (action >= 2001 && action <= 2010) return 'oauth';
  if (action >= 3001 && action <= 3010) return 'auth';
  if (action >= 4001 && action <= 4099) return 'error';
  if (action >= 5001 && action <= 5011) return 'admin';

  return 'system';
}

export const TOOL_USAGE_ACTION_RANGE = { gte: 1000, lte: 1099 } as const;

export function isToolRequestAction(action: number | null | undefined): boolean {
  return typeof action === 'number' && action >= TOOL_USAGE_ACTION_RANGE.gte && action <= TOOL_USAGE_ACTION_RANGE.lte;
}

export function getSourceLabelFromAction(action: number | null | undefined): string {
  const domain = inferDomain(action);
  switch (domain) {
    case 'mcp-request':
      return 'Tool Request';
    case 'reverse':
      return 'Reverse Request';
    case 'lifecycle':
      return 'Lifecycle';
    case 'oauth':
    case 'auth':
      return 'Authentication';
    case 'admin':
      return 'Proxy API';
    case 'error':
      return 'Internal Error';
    default:
      return 'System';
  }
}

/**
 * Return a Prisma-compatible `{ gte, lte }` range for the given domain.
 * Returns `null` for 'system' / unknown (no fixed range).
 */
export function getDomainActionRange(domain: string): { gte: number; lte: number } | null {
  switch (domain) {
    case 'mcp-request':
      return { gte: 1001, lte: 1009 };
    case 'reverse':
      return { gte: 1201, lte: 1206 };
    case 'lifecycle':
      return { gte: 1301, lte: 1314 };
    case 'oauth':
      return { gte: 2001, lte: 2010 };
    case 'auth':
      return { gte: 3001, lte: 3010 };
    case 'error':
      return { gte: 4001, lte: 4099 };
    case 'admin':
      return { gte: 5001, lte: 5011 };
    default:
      return null;
  }
}

export function getDomainLabel(domain: string): string {
  return LOG_DOMAINS.find((d) => d.value === domain)?.label ?? domain;
}

/**
 * Build a Prisma-compatible where-clause fragment for the given domain.
 * Unlike getDomainActionRange, this also handles 'system' (negated ranges).
 * Returns null for unknown domains.
 */
export function buildDomainFilter(domain: string): Record<string, any> | null {
  const range = getDomainActionRange(domain);
  if (range) {
    return { action: range };
  }
  if (domain === 'system') {
    // System = actions not in any known domain range
    return {
      AND: [
        { NOT: { action: { gte: 1001, lte: 1009 } } },
        { NOT: { action: { gte: 1201, lte: 1206 } } },
        { NOT: { action: { gte: 1301, lte: 1314 } } },
        { NOT: { action: { gte: 2001, lte: 2010 } } },
        { NOT: { action: { gte: 3001, lte: 3010 } } },
        { NOT: { action: { gte: 4001, lte: 4099 } } },
        { NOT: { action: { gte: 5001, lte: 5011 } } },
      ],
    };
  }
  return null;
}

/**
 * Build Prisma-compatible where-clause conditions for a given log level.
 * Returns an array of conditions to spread into an AND array.
 * Returns [] for 'all' or unknown levels (no filtering).
 */
export function buildLevelFilter(level: string): any[] {
  switch (level) {
    case 'ERROR':
      return [
        {
          OR: [
            { error: { not: '' } },
            { action: { in: [4001, 2010, 3010] } },
            { statusCode: { gte: 500 } },
          ],
        },
      ];
    case 'WARN':
      return [
        { error: '' },
        { action: { notIn: [4001, 2010, 3010] } },
        { statusCode: { gte: 400, lt: 500 } },
      ];
    case 'INFO':
      return [
        { error: '' },
        { action: { notIn: [4001, 2010, 3010] } },
        {
          OR: [
            { statusCode: { gte: 200, lt: 400 } },
            {
              AND: [
                { action: { gte: 1301, lte: 1314 } },
                { statusCode: null },
              ],
            },
          ],
        },
      ];
    case 'DEBUG':
      return [
        { error: '' },
        { action: { notIn: [4001, 2010, 3010] } },
        {
          NOT: {
            AND: [
              { action: { gte: 1301, lte: 1314 } },
              { statusCode: null },
            ],
          },
        },
        { OR: [{ statusCode: null }, { statusCode: { lt: 200 } }] },
      ];
    default:
      return [];
  }
}

// ========== Level / Outcome Classification ==========

export type LogLevel = 'ERROR' | 'WARN' | 'INFO' | 'DEBUG';

/** Action types that are inherently error events */
const ERROR_ACTIONS = new Set([4001, 2010, 3010]);

/**
 * Infer log level using error field, action type, AND statusCode.
 *
 * Priority:
 *  1. Non-empty `error` field → ERROR
 *  2. Error-class actions (4001, 2010, 3010) → ERROR
 *  3. statusCode ≥ 500 → ERROR
 *  4. statusCode 400-499 → WARN
 *  5. statusCode 200-399 → INFO
 *  6. Lifecycle events without statusCode → INFO (not DEBUG)
 *  7. Default → DEBUG
 */
export function inferLogLevel(log: {
  action?: number | null;
  statusCode?: number | null;
  error?: string | null;
}): LogLevel {
  if (log.error != null && log.error !== '') return 'ERROR';
  if (log.action != null && ERROR_ACTIONS.has(log.action)) return 'ERROR';

  if (log.statusCode != null) {
    if (log.statusCode >= 500) return 'ERROR';
    if (log.statusCode >= 400) return 'WARN';
    if (log.statusCode >= 200) return 'INFO';
  }

  // Lifecycle events (1301-1314) with no statusCode are informational
  if (log.statusCode == null && log.action != null && log.action >= 1301 && log.action <= 1314)
    return 'INFO';

  return 'DEBUG';
}

export function isSuccessfulRequestLog(log: {
  error?: string | null;
  statusCode?: number | null;
}): boolean {
  const errorText = (log.error ?? '').trim();
  if (errorText.length > 0) return false;
  if (log.statusCode != null && log.statusCode >= 400) return false;
  return true;
}

// ========== Time Utilities ==========

/** Parse a timeRange string ("1h", "6h", "24h", "7d", "all") to a unix-seconds start time. */
export function parseTimeRange(timeRange: string): number {
  const now = Math.floor(Date.now() / 1000);
  switch (timeRange) {
    case '1h':
      return now - 60 * 60;
    case '6h':
      return now - 6 * 60 * 60;
    case '24h':
      return now - 24 * 60 * 60;
    case '7d':
      return now - 7 * 24 * 60 * 60;
    case '30d':
      return now - 30 * 24 * 60 * 60;
    case 'all':
      return 0;
    default:
      return now - 24 * 60 * 60;
  }
}

// ========== Action Label ==========

const ACTION_MAP: Record<number, [label: string, machineName: string]> = {
  1001: ['Tool Call', 'RequestTool'],
  1002: ['Resource Read', 'RequestResource'],
  1003: ['Prompt Get', 'RequestPrompt'],
  1004: ['Tool Response', 'ResponseTool'],
  1005: ['Resource Response', 'ResponseResource'],
  1006: ['Prompt Response', 'ResponsePrompt'],
  1007: ['Tool List', 'RequestToolList'],
  1008: ['Resource List', 'RequestResourceList'],
  1009: ['Prompt List', 'RequestPromptList'],
  1201: ['Sampling Request', 'ReverseSamplingRequest'],
  1202: ['Sampling Response', 'ReverseSamplingResponse'],
  1203: ['Roots Request', 'ReverseRootsRequest'],
  1204: ['Roots Response', 'ReverseRootsResponse'],
  1205: ['Elicit Request', 'ReverseElicitRequest'],
  1206: ['Elicit Response', 'ReverseElicitResponse'],
  1301: ['Session Init', 'SessionInit'],
  1302: ['Session Close', 'SessionClose'],
  1310: ['Server Init', 'ServerInit'],
  1311: ['Server Close', 'ServerClose'],
  1312: ['Status Change', 'ServerStatusChange'],
  1313: ['Capability Update', 'ServerCapabilityUpdate'],
  1314: ['Server Notification', 'ServerNotification'],
  2001: ['OAuth Register', 'OAuthRegister'],
  2002: ['OAuth Authorize', 'OAuthAuthorize'],
  2003: ['OAuth Token', 'OAuthToken'],
  2004: ['OAuth Refresh', 'OAuthRefresh'],
  2005: ['OAuth Revoke', 'OAuthRevoke'],
  2010: ['OAuth Error', 'OAuthError'],
  3001: ['Token Validation', 'AuthTokenValidation'],
  3002: ['Permission Check', 'AuthPermissionCheck'],
  3003: ['Rate Limit', 'AuthRateLimit'],
  3010: ['Auth Error', 'AuthError'],
  4001: ['Internal Error', 'ErrorInternal'],
  5001: ['User Create', 'AdminUserCreate'],
  5002: ['User Edit', 'AdminUserEdit'],
  5003: ['User Delete', 'AdminUserDelete'],
  5004: ['Server Create', 'AdminServerCreate'],
  5005: ['Server Edit', 'AdminServerEdit'],
  5006: ['Server Delete', 'AdminServerDelete'],
  5007: ['Proxy Reset', 'AdminProxyReset'],
  5008: ['DB Backup', 'AdminBackupDatabase'],
  5009: ['DB Restore', 'AdminRestoreDatabase'],
  5010: ['DNS Create', 'AdminDNSCreate'],
  5011: ['DNS Delete', 'AdminDNSDelete'],
};

export function getActionLabel(action: number | null | undefined): string {
  if (action == null) return 'Unknown';
  return ACTION_MAP[action]?.[0] ?? `Action ${action}`;
}

export function getActionMachineName(action: number | null | undefined): string {
  if (action == null) return 'Unknown';
  return ACTION_MAP[action]?.[1] ?? `Action${action}`;
}

// ========== Log Formatting ==========

interface MinimalLog {
  addtime?: bigint | number | null;
  action?: number | null;
  statusCode?: number | null;
  error?: string | null;
  sessionId?: string | null;
  userid?: string | null;
  duration?: number | null;
  ua?: string | null;
  ip?: string | null;
  tokenMask?: string | null;
}

/** Generate a human-readable one-line message for a log entry. */
export function generateLogMessage(
  log: Pick<MinimalLog, 'action' | 'statusCode' | 'error'>,
): string {
  const level = inferLogLevel(log);
  const label = getActionLabel(log.action);

  switch (level) {
    case 'ERROR':
      return `${label} failed - ${log.error || 'Unknown error'}`;
    case 'WARN':
      return `${label} completed with warnings - Status ${log.statusCode}`;
    case 'INFO':
      return `${label} processed successfully`;
    case 'DEBUG':
      return `${label} debug information logged`;
    default:
      return `${label} activity`;
  }
}

/** Generate multi-line raw log text for a log entry. */
export function generateRawData(log: MinimalLog): string {
  const addtimeNum = log.addtime != null ? Number(log.addtime) : 0;
  const ts =
    addtimeNum > 0
      ? new Date(addtimeNum * 1000).toISOString().replace('T', ' ').slice(0, -5)
      : 'Unknown time';
  const level = inferLogLevel(log);
  const domain = inferDomain(log.action);
  const message = generateLogMessage(log);

  let raw = `[${ts}] [${level}] [${domain}] ${message}`;

  if (log.sessionId) raw += `\nRequest ID: ${log.sessionId}`;
  if (log.userid) raw += `\nUser ID: ${log.userid}`;
  if (log.action != null) raw += `\nAction: ${log.action} (${getActionLabel(log.action)})`;
  if (log.statusCode != null) raw += `\nStatus: ${log.statusCode}`;
  if (log.duration != null) raw += `\nResponse Time: ${log.duration}ms`;
  if (log.ua) raw += `\nUser Agent: ${log.ua}`;
  if (log.ip) raw += `\nIP: ${log.ip}`;
  if (log.tokenMask) raw += `\nToken: ${log.tokenMask.substring(0, 8)}...`;
  if (log.error) raw += `\nError: ${log.error}`;

  return raw;
}
