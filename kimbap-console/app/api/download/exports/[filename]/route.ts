import { NextRequest, NextResponse } from 'next/server';
import * as fs from 'fs';
import * as path from 'path';

export const dynamic = 'force-dynamic';

async function cleanupExpiredExports(exportDir: string, now: number): Promise<void> {
  let entries: string[];
  try {
    entries = await fs.promises.readdir(exportDir);
  } catch {
    return;
  }

  await Promise.all(
    entries.map(async (entry) => {
      const match = /^exp(\d+)-/.exec(entry);
      if (!match) return;

      const expiresAt = Number(match[1]);
      if (!Number.isFinite(expiresAt) || now <= expiresAt) return;

      const target = path.join(exportDir, path.basename(entry));
      try {
        await fs.promises.unlink(target);
      } catch {
        // Best effort cleanup: ignore delete failures.
      }
    }),
  );
}

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ filename: string }> },
) {
  const { filename = '' } = await params;
  if (!filename || filename.includes('/') || filename.includes('\\')) {
    return NextResponse.json({ error: 'Invalid filename' }, { status: 400 });
  }

  const match = /^exp(\d+)-/.exec(filename);
  if (!match) {
    return NextResponse.json({ error: 'Invalid export token' }, { status: 400 });
  }

  const expiresAt = Number(match[1]);
  if (!Number.isFinite(expiresAt)) {
    return NextResponse.json({ error: 'Invalid expiration' }, { status: 400 });
  }

  const now = Math.floor(Date.now() / 1000);
  if (now > expiresAt) {
    return NextResponse.json({ error: 'Export link expired' }, { status: 410 });
  }

  const exportDir = path.join(process.cwd(), '.exports');
  const safeFilename = path.basename(filename);
  const filePath = path.join(exportDir, safeFilename);

  try {
    const fileBuffer = await fs.promises.readFile(filePath);
    const ext = path.extname(safeFilename).toLowerCase();
    const contentType =
      ext === '.csv'
        ? 'text/csv; charset=utf-8'
        : ext === '.json'
          ? 'application/json; charset=utf-8'
          : ext === '.pdf'
            ? 'application/pdf'
            : 'text/plain; charset=utf-8';

    // One-time export URL: remove file after successful read to prevent disk growth.
    // Response uses in-memory buffer, so deletion won't affect current download.
    fs.promises.unlink(filePath).catch((error) => {
      console.warn('[exports] Failed to delete file after download:', safeFilename, error);
    });
    // Also prune other expired export files on every successful download.
    cleanupExpiredExports(exportDir, now).catch((error) => {
      console.warn('[exports] Failed to cleanup expired files:', error);
    });

    return new NextResponse(fileBuffer, {
      status: 200,
      headers: {
        'Content-Type': contentType,
        'Content-Disposition': `attachment; filename="${safeFilename}"`,
        'Cache-Control': 'no-store'
      }
    });
  } catch {
    return NextResponse.json({ error: 'File not found' }, { status: 404 });
  }
}
