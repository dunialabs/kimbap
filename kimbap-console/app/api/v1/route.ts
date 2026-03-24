import { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';
import { ApiResponse } from '@/lib/api-response';
import { getProtocolHandler } from './handlers';
import { prisma } from '@/lib/prisma';

export const dynamic = 'force-dynamic';
export const revalidate = 0;
export const runtime = 'nodejs';

const PUBLIC_CMD_IDS = new Set([10001, 10002, 10015]);

const rateLimitMap = new Map<string, { count: number; resetAt: number }>();
const RATE_LIMIT_WINDOW_MS = 60_000;
const RATE_LIMIT_MAX_PUBLIC = 20;
const RATE_LIMIT_MAX_AUTH = 120;

function checkRateLimit(ip: string, isPublic: boolean): boolean {
  const now = Date.now();
  const key = `${ip}:${isPublic ? 'pub' : 'auth'}`;
  const entry = rateLimitMap.get(key);
  const max = isPublic ? RATE_LIMIT_MAX_PUBLIC : RATE_LIMIT_MAX_AUTH;
  if (!entry || now > entry.resetAt) {
    rateLimitMap.set(key, { count: 1, resetAt: now + RATE_LIMIT_WINDOW_MS });
    return true;
  }
  entry.count++;
  return entry.count <= max;
}

function getBearerToken(request: NextRequest): string | null {
  const authHeader = request.headers.get('authorization') || request.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) return null;
  const token = authHeader.slice('Bearer '.length).trim();
  return token ? token : null;
}

// Central API endpoint that routes based on cmdId
export async function POST(request: NextRequest) {
  let cmdId = 0;

  try {
    const clientIP =
      request.headers.get('x-forwarded-for')?.split(',')[0]?.trim() ||
      request.headers.get('x-real-ip') ||
      'unknown';
    const body = await request.json();

    cmdId = body.common?.cmdId || 0;

    if (!cmdId) {
      return ApiResponse.missingCmdId();
    }

    // Get the appropriate handler for this cmdId
    const handler = getProtocolHandler(cmdId);

    if (!handler) {
      return ApiResponse.protocolNotImplemented(cmdId);
    }

    const isPublicCmd = PUBLIC_CMD_IDS.has(cmdId);
    if (!checkRateLimit(clientIP, isPublicCmd)) {
      return NextResponse.json(
        { common: { cmdId, code: 429 }, data: { message: 'Too many requests' } },
        { status: 429 },
      );
    }

    if (!isPublicCmd) {
      const token = getBearerToken(request);
      if (!token) {
        return ApiResponse.unauthorized(cmdId);
      }

      const user = await prisma.user.findFirst({
        where: { accessToken: token },
      });

      if (!user) {
        return ApiResponse.unauthorized(cmdId);
      }

      body.common = body.common || {};
      body.common.userid = user.userid;
    }

    // Execute the handler
    const responseData = await handler(body);

    // Return success response with handler data
    return ApiResponse.success(cmdId, responseData);
  } catch (error) {
    // Return error response
    return ApiResponse.handleError(cmdId, error);
  }
}
