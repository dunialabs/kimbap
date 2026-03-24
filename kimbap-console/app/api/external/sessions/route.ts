import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';

export const dynamic = 'force-dynamic';

interface SessionsInput {
  limit?: number;
}

export async function POST(request: NextRequest) {
  try {
    await authenticate(request);

    let body: SessionsInput = {};
    try {
      body = await request.json();
    } catch {
      body = {};
    }

    const limit = typeof body.limit === 'number' && body.limit > 0 ? Math.min(Math.floor(body.limit), 200) : 50;
    const fiveMinutesAgo = Math.floor(Date.now() / 1000) - 5 * 60;

    const recentSessions = await prisma.log.findMany({
      where: {
        addtime: {
          gte: BigInt(fiveMinutesAgo),
        },
        sessionId: {
          not: '',
        },
      },
      select: {
        sessionId: true,
        userid: true,
        serverId: true,
        addtime: true,
        ip: true,
      },
      distinct: ['sessionId'],
      orderBy: {
        addtime: 'desc',
      },
      take: limit,
    });

    const sessions = recentSessions.map((s) => ({
      sessionId: s.sessionId,
      userId: s.userid,
      serverId: s.serverId,
      lastActivity: Number(s.addtime),
      ip: s.ip,
    }));

    return ApiResponse.success({ sessions, count: sessions.length });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}
