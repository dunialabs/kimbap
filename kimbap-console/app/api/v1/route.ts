import { NextRequest } from 'next/server';
import { ApiResponse } from '@/lib/api-response';
import { getProtocolHandler } from './handlers';
import { prisma } from '@/lib/prisma';

export const dynamic = 'force-dynamic';
export const revalidate = 0;
export const runtime = 'nodejs';

const PUBLIC_CMD_IDS = new Set([10015]);

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
    const body = await request.json();
    
    // Extract cmdId from request
    cmdId = body.common?.cmdId || 0;
    
    if (!cmdId) {
      return ApiResponse.missingCmdId();
    }

    // Get the appropriate handler for this cmdId
    const handler = getProtocolHandler(cmdId);
    
    if (!handler) {
      return ApiResponse.protocolNotImplemented(cmdId);
    }

    if (!PUBLIC_CMD_IDS.has(cmdId)) {
      const token = getBearerToken(request);
      if (!token) {
        return ApiResponse.unauthorized(cmdId);
      }

      const user = await prisma.user.findFirst({
        where: { accessToken: token }
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
