import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';
import { getActionMachineName } from '@/lib/log-utils';
import { sanitizeCsvField } from './csv-utils';
import * as fs from 'fs';
import * as path from 'path';
import { randomUUID } from 'crypto';

interface Request20009 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number;
    serverId: number;
    format: number;
    toolIds: string[];
  };
}

interface Response20009Data {
  downloadUrl: string;
  fileName: string;
  expiresAt: number;
}


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

function generateFileName(format: number, timestamp: string, timeRange: number): string {
  const formatExt = format === 1 ? 'csv' : format === 2 ? 'json' : 'pdf';
  return `${timestamp}-tool-usage-report-${timeRange}days.${formatExt}`;
}

export async function handleProtocol20009(body: Request20009): Promise<Response20009Data> {
  try {
    const { timeRange, serverId, format, toolIds } = body.params;
    const allowedTimeRanges = [1, 7, 30, 90];
    const MAX_EXPORT_RECORDS = 10000;

    if (!Number.isFinite(timeRange) || !Number.isInteger(timeRange) || !allowedTimeRanges.includes(timeRange)) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'timeRange', details: 'Unsupported time range' });
    }

    if (![1, 2, 3].includes(format)) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'format', details: 'Unsupported export format' });
    }

    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (timeRange * 24 * 60 * 60);
    const proxy = await getProxy();

    const whereCondition: any = {
      proxyKey: proxy.proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      action: {
        in: [1001, 1002, 1003, 1004, 1005, 1006]
      },
      serverId: {
        not: null
      }
    };

    if (serverId > 0) {
      whereCondition.serverId = serverId.toString();
    } else if (toolIds && toolIds.length > 0) {
      whereCondition.serverId = {
        in: toolIds
      };
    }

    const exportData = await prisma.log.findMany({
      where: whereCondition,
      select: {
        addtime: true,
        action: true,
        userid: true,
        serverId: true,
        sessionId: true,
        duration: true,
        statusCode: true,
        error: true,
        requestParams: true,
        responseResult: true
      },
      orderBy: {
        addtime: 'desc'
      },
      take: MAX_EXPORT_RECORDS + 1
    });

    if (exportData.length > MAX_EXPORT_RECORDS) {
      throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'timeRange', details: 'Export range is too large' });
    }

    const formattedData = exportData.map((log) => ({
      timestamp: new Date(Number(log.addtime) * 1000).toISOString(),
      actionType: log.action,
      actionName: getActionMachineName(log.action),
      toolId: log.serverId || '',
      userId: log.userid,
      sessionId: log.sessionId,
      duration: log.duration || 0,
      statusCode: log.statusCode || 0,
      success: (log.statusCode || 0) >= 200 && (log.statusCode || 0) < 300 ? 'Yes' : 'No',
      error: log.error || '',
      requestParams: log.requestParams || '',
      responseResult: log.responseResult || ''
    }));

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const expiresAt = now + (24 * 60 * 60);
    const fileName = generateFileName(format, `exp${expiresAt}-${timestamp}-${randomUUID()}`, timeRange);
    const exportDir = path.join(process.cwd(), '.exports');
    await fs.promises.mkdir(exportDir, { recursive: true });

    const filePath = path.join(exportDir, fileName);
    if (format === 1) {
      if (formattedData.length === 0) {
        await fs.promises.writeFile(filePath, 'No data available', 'utf8');
      } else {
        const headers = Object.keys(formattedData[0]);
        const csvContent = [
          headers.join(','),
          ...formattedData.map((row) => headers.map((header) => sanitizeCsvField(row[header as keyof typeof row])).join(','))
        ].join('\n');
        await fs.promises.writeFile(filePath, csvContent, 'utf8');
      }
    } else if (format === 2) {
      const jsonContent = JSON.stringify(
        {
          exportTime: new Date().toISOString(),
          recordsCount: formattedData.length,
          data: formattedData
        },
        null,
        2
      );
      await fs.promises.writeFile(filePath, jsonContent, 'utf8');
    } else {
      const lines: string[] = [];
      lines.push('Tool Usage Report');
      lines.push(`Export Time: ${new Date().toISOString()}`);
      lines.push(`Time Range: ${timeRange} days`);
      lines.push(`Records: ${formattedData.length}`);
      lines.push('');

      for (let i = 0; i < formattedData.length; i++) {
        const row = formattedData[i];
        lines.push(`${i + 1}. ${row.actionName} (${row.actionType})`);
        lines.push(`   Timestamp: ${row.timestamp}`);
        lines.push(`   Tool: ${row.toolId}`);
        lines.push(`   User: ${row.userId}`);
        lines.push(`   Status: ${row.statusCode}`);
        lines.push(`   Duration: ${row.duration}ms`);
        if (row.error) {
          lines.push(`   Error: ${row.error}`);
        }
      }

      await fs.promises.writeFile(filePath, buildSimplePdfBuffer(lines));
    }

    const response: Response20009Data = {
      downloadUrl: `/api/download/exports/${fileName}`,
      fileName,
      expiresAt
    };

    console.log('Protocol 20009 response:', {
      fileName,
      recordsCount: formattedData.length,
      format,
      timeRange
    });

    return response;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('Protocol 20009 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to export usage report' });
  }
}
