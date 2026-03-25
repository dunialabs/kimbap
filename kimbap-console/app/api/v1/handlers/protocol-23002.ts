import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';
import { parseTimeRange, getDomainLabel } from '@/lib/log-utils';

interface Request23002 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: string; // "1h", "6h", "24h", "7d", "30d"
  };
}

interface DomainStats {
  domain: string; // Domain name (e.g. "mcp-request", "lifecycle")
  label: string; // Human-readable label
  logCount: number;
  errorCount: number;
  percentage: number;
}

interface HourlyStats {
  hour: string; // "14:00" or "03/04 14:00" for multi-day ranges
  totalCount: number;
  errorCount: number;
  timestamp: number;
}

interface LogStatistics {
  totalLogs: number;
  errorLogs: number;
  warnLogs: number;
  infoLogs: number;
  debugLogs: number;
  errorRate: number;
  domainStats: DomainStats[];
  hourlyStats: HourlyStats[];
}

interface Response23002Data {
  statistics: LogStatistics;
}

// Raw-query result types (PostgreSQL returns bigint as BigInt)
interface LevelCountRow {
  error_count: bigint;
  warn_count: bigint;
  info_count: bigint;
  debug_count: bigint;
}

interface DomainRow {
  domain: string;
  log_count: bigint;
  error_count: bigint;
}

interface HourlyRow {
  hour_bucket: bigint;
  total_count: bigint;
  error_count: bigint;
}

/**
 * Protocol 23002 - Get Log Statistics
 *
 * Fixes applied:
 *  - Added proxyKey filter (was missing — showed all proxies' logs)
 *  - Replaced full-memory loading with DB-level aggregation
 *  - Fixed inferSource (was treating Int action as String — always returned 'system')
 *  - Fixed inferLogLevel (now considers error field + action type, not just statusCode)
 *  - hourlyStats now respects the actual timeRange (was hardcoded to 24h)
 */
export async function handleProtocol23002(body: Request23002): Promise<Response23002Data> {
  try {
    const { timeRange = '24h' } = body.params;

    // 1. Get proxyKey
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-23002] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information',
      });
    }

    const startTime = parseTimeRange(timeRange);

    // 2. Run all aggregation queries in parallel
    const [levelCounts, domainRows, hourlyRows, lifecycleInfoResult] = await Promise.all([
      // Level counts — single row with 4 counters
      prisma.$queryRawUnsafe<LevelCountRow[]>(
        `
        SELECT
          COUNT(*) FILTER (WHERE
            (error IS NOT NULL AND error != '')
            OR status_code >= 500
            OR action IN (4001, 2010, 3010)
          ) as error_count,
          COUNT(*) FILTER (WHERE
            status_code >= 400 AND status_code < 500
            AND (error IS NULL OR error = '')
            AND (action IS NULL OR action NOT IN (4001, 2010, 3010))
          ) as warn_count,
          COUNT(*) FILTER (WHERE
            status_code IS NOT NULL AND status_code >= 200 AND status_code < 400
            AND (error IS NULL OR error = '')
            AND (action IS NULL OR action NOT IN (4001, 2010, 3010))
          ) as info_count,
          COUNT(*) FILTER (WHERE
            (status_code IS NULL OR status_code < 200)
            AND (error IS NULL OR error = '')
            AND (action IS NULL OR action NOT IN (4001, 2010, 3010))
            AND NOT (action BETWEEN 1301 AND 1314 AND status_code IS NULL)
          ) as debug_count
        FROM log
        WHERE proxy_key = $1
          AND addtime >= $2
      `,
        proxyKey,
        BigInt(startTime),
      ),

      // Domain stats — one row per domain
      prisma.$queryRawUnsafe<DomainRow[]>(
        `
        SELECT
          CASE
            WHEN action BETWEEN 1001 AND 1009 THEN 'mcp-request'
            WHEN action BETWEEN 1201 AND 1206 THEN 'reverse'
            WHEN action BETWEEN 1301 AND 1314 THEN 'lifecycle'
            WHEN action BETWEEN 2001 AND 2010 THEN 'oauth'
            WHEN action BETWEEN 3001 AND 3010 THEN 'auth'
            WHEN action BETWEEN 4001 AND 4099 THEN 'error'
            WHEN action BETWEEN 5001 AND 5011 THEN 'admin'
            ELSE 'system'
          END as domain,
          COUNT(*) as log_count,
          COUNT(*) FILTER (WHERE
            (error IS NOT NULL AND error != '')
            OR status_code >= 500
            OR action IN (4001, 2010, 3010)
          ) as error_count
        FROM log
        WHERE proxy_key = $1
          AND addtime >= $2
        GROUP BY 1
        ORDER BY log_count DESC
      `,
        proxyKey,
        BigInt(startTime),
      ),

      // Hourly stats — skip entirely for 'all' (no bounded time range to bucket)
      timeRange === 'all'
        ? ([] as HourlyRow[])
        : prisma.$queryRawUnsafe<HourlyRow[]>(
            `
        SELECT
          (addtime / 3600) * 3600 as hour_bucket,
          COUNT(*) as total_count,
          COUNT(*) FILTER (WHERE
            (error IS NOT NULL AND error != '')
            OR status_code >= 500
            OR action IN (4001, 2010, 3010)
          ) as error_count
        FROM log
        WHERE proxy_key = $1
          AND addtime >= $2
        GROUP BY 1
        ORDER BY hour_bucket ASC
      `,
            proxyKey,
            BigInt(startTime),
          ),

      // Lifecycle-as-info count (events 1301-1314 with no statusCode and no error)
      prisma.$queryRawUnsafe<{ cnt: bigint }[]>(
        `
        SELECT COUNT(*) as cnt
        FROM log
        WHERE proxy_key = $1
          AND addtime >= $2
          AND action BETWEEN 1301 AND 1314
          AND (error IS NULL OR error = '')
          AND (status_code IS NULL)
      `,
        proxyKey,
        BigInt(startTime),
      ),
    ]);

    // 3. Parse level counts
    const lc = levelCounts[0];
    const errorLogs = lc ? Number(lc.error_count) : 0;
    const warnLogs = lc ? Number(lc.warn_count) : 0;
    const infoLogs = lc ? Number(lc.info_count) : 0;
    const debugLogs = lc ? Number(lc.debug_count) : 0;
    // Lifecycle events without statusCode counted as INFO (not captured by the 4 buckets above)
    const countedTotal = errorLogs + warnLogs + infoLogs + debugLogs;
    const lifecycleInfoCount = Number(lifecycleInfoResult[0]?.cnt ?? 0);

    const totalLogs = countedTotal + lifecycleInfoCount;
    const adjustedInfoLogs = infoLogs + lifecycleInfoCount;
    const errorRate = totalLogs > 0 ? (errorLogs / totalLogs) * 100 : 0;

    // 4. Build domain stats
    const domainStats: DomainStats[] = domainRows.map((row) => ({
      domain: row.domain,
      label: getDomainLabel(row.domain),
      logCount: Number(row.log_count),
      errorCount: Number(row.error_count),
      percentage: totalLogs > 0 ? Math.round((Number(row.log_count) / totalLogs) * 1000) / 10 : 0,
    }));

    // 5. Build hourly stats — fill empty hours from DB results
    const hourlyDbMap = new Map<number, { total: number; errors: number }>();
    for (const row of hourlyRows) {
      hourlyDbMap.set(Number(row.hour_bucket), {
        total: Number(row.total_count),
        errors: Number(row.error_count),
      });
    }

    const now = Math.floor(Date.now() / 1000);
    // Skip hourly stats for 'all' — impractical to bucket entire history hourly
    const hourlyStats: HourlyStats[] = [];
    if (timeRange !== 'all') {
      const firstBucket = Math.floor(startTime / 3600) * 3600;
      const nowBucket = Math.floor(now / 3600) * 3600;
      const hoursCount = Math.floor((nowBucket - firstBucket) / 3600) + 1;
      const showDate = timeRange === '7d' || timeRange === '30d';

      for (let i = 0; i < hoursCount; i++) {
        const bucket = firstBucket + i * 3600;
        if (bucket > now) break;

        const data = hourlyDbMap.get(bucket) ?? { total: 0, errors: 0 };
        const date = new Date(bucket * 1000);

        const hourLabel = showDate
          ? `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:00`
          : `${String(date.getHours()).padStart(2, '0')}:00`;

        hourlyStats.push({
          hour: hourLabel,
          totalCount: data.total,
          errorCount: data.errors,
          timestamp: bucket,
        });
      }
    }

    // 6. Assemble response
    const statistics: LogStatistics = {
      totalLogs,
      errorLogs,
      warnLogs,
      infoLogs: adjustedInfoLogs,
      debugLogs,
      errorRate: Math.round(errorRate * 10) / 10,
      domainStats,
      hourlyStats,
    };

    console.log('[Protocol-23002] Response:', {
      totalLogs,
      errorRate: statistics.errorRate,
      domainsCount: domainStats.length,
      hourlyBuckets: hourlyStats.length,
      timeRange,
      proxyKey,
    });

    return { statistics };
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('[Protocol-23002] Error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to get log statistics',
    });
  }
}
