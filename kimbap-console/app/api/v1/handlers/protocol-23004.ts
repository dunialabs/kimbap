import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';
import * as fs from 'fs';
import * as path from 'path';
import { randomUUID } from 'crypto';
import {
  inferDomain,
  inferLogLevel,
  buildDomainFilter,
  buildLevelFilter,
  parseTimeRange,
  generateLogMessage,
  generateRawData,
} from '@/lib/log-utils';

interface Request23004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: string; // Time range: "1h", "6h", "24h", "7d", "30d"
    level: string; // Log level filtering
    source: string; // Log source filtering
    search: string; // Search keywords
    format: number; // Export format: 1-TXT, 2-JSON, 3-CSV
    maxRecords: number; // Maximum number of records, default 10000
  };
}

interface Response23004Data {
  downloadUrl: string;
  fileName: string;
  fileSize: number;
  recordCount: number;
  expiresAt: number;
}

/**
 * Protocol 23004 - Export Logs
 * Export log
 *
 * Fixes applied:
 *  - Replaced inline inferSource/inferLogLevel with shared log-utils
 *  - Fixed source filter: was using `contains` on Int field (never worked);
 *    now maps domain name → action range for proper DB filtering
 *  - Used shared generateLogMessage/generateRawData
 *  - Added proxyKey filter to match protocol-23001
 *  - Fixed ERROR level filter to check error field + action types
 *  - Hardened export: writes to filesystem with expiring download URLs
 */
export async function handleProtocol23004(body: Request23004): Promise<Response23004Data> {
  try {
    const {
      timeRange = '24h',
      level = 'all',
      source = 'all',
      search = '',
      format = 1,
      maxRecords = 10000,
    } = body.params;

    if (![1, 2, 3].includes(format)) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported export format' });
    }

    const cappedMaxRecords = Math.min(Math.max(maxRecords || 0, 1), 10000);

    // 1. Get proxyKey
    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
      console.log('[Protocol-23004] Got proxyKey:', proxyKey);
    } catch (error) {
      console.error('[Protocol-23004] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information',
      });
    }

    // 2. Build where condition using AND array (avoids OR/property conflicts)
    const startTime = parseTimeRange(timeRange);
    const andConditions: any[] = [{ proxyKey }];

    if (startTime > 0) {
      andConditions.push({ addtime: { gte: BigInt(startTime) } });
    }

    // Level filter (aligned with inferLogLevel priority)
    if (level !== 'all') {
      andConditions.push(...buildLevelFilter(level));
    }

    // Source/domain filter
    if (source !== 'all') {
      const domainFilter = buildDomainFilter(source);
      if (domainFilter) {
        andConditions.push(domainFilter);
      }
    }

    // Search filter
    if (search) {
      andConditions.push({
        OR: [
          { error: { contains: search, mode: 'insensitive' as const } },
          { ua: { contains: search, mode: 'insensitive' as const } },
          { userid: { contains: search, mode: 'insensitive' as const } },
          { sessionId: { contains: search, mode: 'insensitive' as const } },
        ],
      });
    }

    const whereCondition: any =
      andConditions.length === 1 ? andConditions[0] : { AND: andConditions };

    // 3. Query
    const logs = await prisma.log.findMany({
      where: whereCondition,
      orderBy: {
        addtime: 'desc',
      },
      take: cappedMaxRecords,
      select: {
        id: true,
        addtime: true,
        userid: true,
        tokenMask: true,
        action: true,
        ip: true,
        ua: true,
        statusCode: true,
        error: true,
        duration: true,
        sessionId: true,
      },
    });

    // 4. Generate file content
    let fileContent: string;
    let formatName: string;
    let fileExtension: string;

    switch (format) {
      case 1: {
        // TXT
        formatName = 'TXT';
        fileExtension = 'txt';

        fileContent = logs.map((log) => generateRawData(log)).join('\n\n');
        break;
      }

      case 2: {
        // JSON
        formatName = 'JSON';
        fileExtension = 'json';

        const jsonData = logs.map((log) => {
          const timestamp = new Date(Number(log.addtime) * 1000).toISOString();
          const logLevel = inferLogLevel(log);
          const domain = inferDomain(log.action);
          const message = generateLogMessage(log);

          return {
            id: log.id.toString(),
            timestamp,
            level: logLevel,
            message,
            source: domain,
            requestId: log.sessionId || null,
            userId: log.userid || null,
            details: {
              action: log.action || null,
              statusCode: log.statusCode,
              responseTime: log.duration || null,
              userAgent: log.ua || null,
              ip: log.ip || null,
              tokenId: log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : null,
              error: log.error || null,
            },
            rawData: generateRawData(log),
          };
        });

        fileContent = JSON.stringify(
          {
            exportInfo: {
              exportTime: new Date().toISOString(),
              timeRange,
              filters: { level, source, search },
              totalRecords: logs.length,
            },
            logs: jsonData,
          },
          null,
          2,
        );
        break;
      }

      case 3: {
        // CSV
        formatName = 'CSV';
        fileExtension = 'csv';

        // CSV header
        let csvContent =
          'ID,Timestamp,Level,Source,Message,Status Code,Response Time,User ID,Request ID,IP,User Agent,Token ID,Error\n';

        // CSV helper — RFC 4180 compliant
        const csvEscape = (val: string): string => {
          if (val.includes(',') || val.includes('"') || val.includes('\n') || val.includes('\r')) {
            return `"${val.replace(/"/g, '""')}"`;
          }
          return val;
        };

        // CSV data rows
        logs.forEach((log) => {
          const timestamp = new Date(Number(log.addtime) * 1000).toISOString();
          const logLevel = inferLogLevel(log);
          const domain = inferDomain(log.action);
          const message = generateLogMessage(log);

          const row = [
            log.id.toString(),
            timestamp,
            logLevel,
            domain,
            csvEscape(message),
            (log.statusCode ?? 0).toString(),
            (log.duration || 0).toString(),
            csvEscape(log.userid || ''),
            csvEscape(log.sessionId || ''),
            csvEscape(log.ip || ''),
            csvEscape(log.ua || ''),
            log.tokenMask ? log.tokenMask.substring(0, 8) + '...' : '',
            csvEscape(log.error || ''),
          ].join(',');

          csvContent += row + '\n';
        });

        fileContent = csvContent;
        break;
      }

      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, {
          details: 'Unsupported export format',
        });
    }

    // 5. Write to filesystem
    const dateStr = new Date().toISOString().split('T')[0];
    const timeStr = new Date().toTimeString().split(' ')[0].replace(/:/g, '-');
    const expiresAt = Math.floor(Date.now() / 1000) + (24 * 60 * 60);
    const fileName = `exp${expiresAt}-kimbap-logs-${dateStr}-${timeStr}-${timeRange}-${randomUUID()}.${fileExtension}`;

    const exportDir = path.join(process.cwd(), '.exports');
    await fs.promises.mkdir(exportDir, { recursive: true });

    const filePath = path.join(exportDir, fileName);
    await fs.promises.writeFile(filePath, fileContent, 'utf8');

    const response: Response23004Data = {
      downloadUrl: `/api/download/exports/${fileName}`,
      fileName,
      fileSize: (await fs.promises.stat(filePath)).size,
      recordCount: logs.length,
      expiresAt,
    };

    console.log('[Protocol-23004] Response:', {
      format: formatName,
      fileName,
      recordCount: response.recordCount,
      fileSize: response.fileSize,
      timeRange,
      filters: { level, source, hasSearch: !!search },
      proxyKey,
    });

    return response;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('[Protocol-23004] Error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to export logs',
    });
  }
}
