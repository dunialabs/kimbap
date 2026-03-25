import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';
import { sanitizeCsvField } from './csv-utils';
import * as fs from 'fs';
import * as path from 'path';
import { randomUUID } from 'crypto';

interface Request21009 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;
    format: number;
    tokenIds: string[];
    includeGeoData: boolean;
    includeClientData: boolean;
  };
}

interface TokenDetail {
  tokenId: string;
  tokenName: string;
  totalRequests: number;
  successRequests: number;
  failedRequests: number;
  successRate: number;
  clientCount: number;
  lastUsed: number;
  geoData?: Array<{
    country: string;
    city: string;
    requests: number;
  }>;
  clientData?: Array<{
    clientIP: string;
    requests: number;
    successRate: number;
  }>;
}

interface ExportData {
  summary: {
    exportTime: number;
    timeRange: number;
    totalTokens: number;
    totalRequests: number;
    averageSuccessRate: number;
  };
  tokenDetails: TokenDetail[];
}

interface Response21009Data {
  exportUrl: string;
  filename: string;
  fileSize: number;
  recordCount: number;
  format: string;
}

type LogRecord = {
  tokenMask: string;
  ip: string;
  ua: string;
  statusCode: number | null;
  addtime: bigint;
};

function buildSimplePdfBuffer(lines: string[]): Buffer {
  const escapedLines = lines.map((line) =>
    line.replace(/\\/g, '\\\\').replace(/\(/g, '\\(').replace(/\)/g, '\\)')
  );
  const contentLines = escapedLines.map((line) => `(${line}) Tj T*`).join('\n');
  const streamContent = `BT\n/F1 10 Tf\n50 780 Td\n12 TL\n${contentLines}\nET`;

  const objects: string[] = [];
  objects.push('1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n');
  objects.push('2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n');
  objects.push('3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n');
  objects.push(`4 0 obj\n<< /Length ${Buffer.byteLength(streamContent, 'utf8')} >>\nstream\n${streamContent}\nendstream\nendobj\n`);
  objects.push('5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n');

  let pdf = '%PDF-1.4\n';
  const offsets: number[] = [0];

  for (const obj of objects) {
    offsets.push(Buffer.byteLength(pdf, 'utf8'));
    pdf += obj;
  }

  const xrefOffset = Buffer.byteLength(pdf, 'utf8');
  pdf += `xref\n0 ${objects.length + 1}\n`;
  pdf += '0000000000 65535 f \n';
  for (let i = 1; i < offsets.length; i++) {
    pdf += `${offsets[i].toString().padStart(10, '0')} 00000 n \n`;
  }
  pdf += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF`;

  return Buffer.from(pdf, 'utf8');
}

function generateGeoData(tokenLogs: LogRecord[]) {
  const totalRequests = tokenLogs.length;
  if (totalRequests === 0) {
    return [];
  }

  return [{ country: 'UNKNOWN', city: 'UNKNOWN', requests: totalRequests }];
}

function generateClientData(tokenLogs: LogRecord[]) {
  const clientStats = new Map<string, { requests: number; successRequests: number }>();
  for (const log of tokenLogs) {
    if (!log.ip) {
      continue;
    }
    if (!clientStats.has(log.ip)) {
      clientStats.set(log.ip, { requests: 0, successRequests: 0 });
    }
    const stats = clientStats.get(log.ip)!;
    stats.requests += 1;
    if ((log.statusCode || 0) >= 200 && (log.statusCode || 0) < 300) {
      stats.successRequests += 1;
    }
  }

  return Array.from(clientStats.entries())
    .map(([ip, stats]) => ({
      clientIP: `${ip.substring(0, 10)}...`,
      requests: stats.requests,
      successRate: stats.requests > 0 ? (stats.successRequests / stats.requests) * 100 : 0
    }))
    .sort((a, b) => b.requests - a.requests)
    .slice(0, 10);
}

export async function handleProtocol21009(body: Request21009): Promise<Response21009Data> {
  try {
    const { timeRange, format, tokenIds, includeGeoData, includeClientData } = body.params;
    const allowedTimeRanges = [1, 7, 30, 90];
    const MAX_SCAN_LOGS = 20000;

    if (!Number.isFinite(timeRange) || !Number.isInteger(timeRange) || !allowedTimeRanges.includes(timeRange)) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'timeRange', details: 'Unsupported time range' });
    }

    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (timeRange * 24 * 60 * 60);
    const proxy = await getProxy();

    if (![1, 2, 3].includes(format)) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { details: 'Unsupported export format' });
    }

    const whereCondition: any = {
      proxyKey: proxy.proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      tokenMask: {
        not: ''
      }
    };

    if (tokenIds && tokenIds.length > 0) {
      whereCondition.userid = {
        in: tokenIds.map((id: string) => id.trim()).filter((id: string) => id.length > 0)
      };
    }

    const logs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        tokenMask: true,
        ip: true,
        ua: true,
        statusCode: true,
        addtime: true
      },
      take: MAX_SCAN_LOGS + 1
    });

    if (logs.length > MAX_SCAN_LOGS) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'timeRange', details: 'Audit range is too large' });
    }

    const totalRequests = logs.length;
    const successRequests = logs.filter((log) => (log.statusCode || 0) >= 200 && (log.statusCode || 0) < 300).length;
    const averageSuccessRate = totalRequests > 0 ? (successRequests / totalRequests) * 100 : 0;

    const tokenStatsMap = new Map<string, {
      logs: LogRecord[];
      totalRequests: number;
      successRequests: number;
      clientIPs: Set<string>;
      lastUsed: number;
    }>();

    for (const log of logs) {
      if (!tokenStatsMap.has(log.tokenMask)) {
        tokenStatsMap.set(log.tokenMask, {
          logs: [],
          totalRequests: 0,
          successRequests: 0,
          clientIPs: new Set<string>(),
          lastUsed: 0
        });
      }

      const stats = tokenStatsMap.get(log.tokenMask)!;
      stats.logs.push(log);
      stats.totalRequests += 1;
      if ((log.statusCode || 0) >= 200 && (log.statusCode || 0) < 300) {
        stats.successRequests += 1;
      }
      if (log.ip) {
        stats.clientIPs.add(log.ip);
      }
      stats.lastUsed = Math.max(stats.lastUsed, Number(log.addtime));
    }

    const tokenDetails: TokenDetail[] = Array.from(tokenStatsMap.entries())
      .map(([tokenMask, stats]) => {
        const tokenPrefix = tokenMask ? tokenMask.substring(0, 8) : 'unknown';
        const successRate = stats.totalRequests > 0 ? (stats.successRequests / stats.totalRequests) * 100 : 0;
        const detail: TokenDetail = {
          tokenId: `${tokenPrefix}...`,
          tokenName: `Token ${tokenPrefix}...`,
          totalRequests: stats.totalRequests,
          successRequests: stats.successRequests,
          failedRequests: stats.totalRequests - stats.successRequests,
          successRate: Math.round(successRate * 10) / 10,
          clientCount: stats.clientIPs.size,
          lastUsed: stats.lastUsed
        };

        if (includeGeoData) {
          detail.geoData = generateGeoData(stats.logs);
        }
        if (includeClientData) {
          detail.clientData = generateClientData(stats.logs);
        }

        return detail;
      })
      .sort((a, b) => b.totalRequests - a.totalRequests);

    const exportData: ExportData = {
      summary: {
        exportTime: now,
        timeRange,
        totalTokens: tokenStatsMap.size,
        totalRequests,
        averageSuccessRate: Math.round(averageSuccessRate * 10) / 10
      },
      tokenDetails
    };

    let fileBuffer: Buffer;
    let formatName = 'CSV';
    let fileExtension = 'csv';

    if (format === 1) {
      let csvContent = 'Token ID,Token Name,Total Requests,Success Requests,Failed Requests,Success Rate(%),Client Count,Last Used\n';
      for (const token of exportData.tokenDetails) {
        csvContent += [
          sanitizeCsvField(token.tokenId),
          sanitizeCsvField(token.tokenName),
          sanitizeCsvField(token.totalRequests),
          sanitizeCsvField(token.successRequests),
          sanitizeCsvField(token.failedRequests),
          sanitizeCsvField(token.successRate),
          sanitizeCsvField(token.clientCount),
          sanitizeCsvField(new Date(token.lastUsed * 1000).toISOString())
        ].join(',') + '\n';
      }
      fileBuffer = Buffer.from(csvContent, 'utf8');
      formatName = 'CSV';
      fileExtension = 'csv';
    } else if (format === 2) {
      fileBuffer = Buffer.from(JSON.stringify(exportData, null, 2), 'utf8');
      formatName = 'JSON';
      fileExtension = 'json';
    } else {
      const lines: string[] = [];
      lines.push('Token Usage Report');
      lines.push('');
      lines.push(`Export Time: ${new Date(exportData.summary.exportTime * 1000).toISOString()}`);
      lines.push(`Time Range: ${timeRange} days`);
      lines.push(`Total Tokens: ${exportData.summary.totalTokens}`);
      lines.push(`Total Requests: ${exportData.summary.totalRequests}`);
      lines.push(`Average Success Rate: ${exportData.summary.averageSuccessRate}%`);
      lines.push('');
      lines.push('Token Details:');
      for (let i = 0; i < exportData.tokenDetails.length; i++) {
        const token = exportData.tokenDetails[i];
        lines.push(`${i + 1}. ${token.tokenName}`);
        lines.push(`   Requests: ${token.totalRequests} (${token.successRequests} success, ${token.failedRequests} failed)`);
        lines.push(`   Success Rate: ${token.successRate}%`);
        lines.push(`   Clients: ${token.clientCount}`);
        lines.push(`   Last Used: ${new Date(token.lastUsed * 1000).toISOString()}`);
      }

      fileBuffer = buildSimplePdfBuffer(lines);
      formatName = 'PDF';
      fileExtension = 'pdf';
    }

    const dateStr = new Date().toISOString().split('T')[0];
    const expiresAt = Math.floor(Date.now() / 1000) + (24 * 60 * 60);
    const filename = `exp${expiresAt}-token-usage-report-${dateStr}-${timeRange}days-${randomUUID()}.${fileExtension}`;
    const exportDir = path.join(process.cwd(), '.exports');
    await fs.promises.mkdir(exportDir, { recursive: true });

    const filePath = path.join(exportDir, filename);
    await fs.promises.writeFile(filePath, fileBuffer);

    const response: Response21009Data = {
      exportUrl: `/api/download/exports/${filename}`,
      filename,
      fileSize: (await fs.promises.stat(filePath)).size,
      recordCount: exportData.tokenDetails.length,
      format: formatName
    };

    console.log('Protocol 21009 response:', {
      format: formatName,
      filename,
      recordCount: response.recordCount,
      fileSize: response.fileSize,
      timeRange
    });

    return response;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('Protocol 21009 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to export token usage report' });
  }
}
