import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';
import { ExternalApiError, E2001, E2002, E2003 } from './error-codes';

export interface AuthUser {
  userid: string;
  role: number; // 1=owner, 2=admin, 3=member
  accessToken: string;
  proxyKey: string;
}

/**
 * Extract and validate Bearer token from request
 * Returns the authenticated user info
 */
export async function authenticate(request: NextRequest): Promise<AuthUser> {
  const authHeader = request.headers.get('Authorization');

  if (!authHeader) {
    throw new ExternalApiError(E2001, 'Access token is required');
  }

  // Check Bearer token format
  if (!authHeader.startsWith('Bearer ')) {
    throw new ExternalApiError(E2002, 'Invalid access token');
  }

  const token = authHeader.slice(7); // Remove 'Bearer ' prefix

  if (!token) {
    throw new ExternalApiError(E2002, 'Invalid access token');
  }

  // Find user by access token
  const user = await prisma.user.findFirst({
    where: { accessToken: token },
  });

  if (!user) {
    throw new ExternalApiError(E2002, 'Invalid access token');
  }

  if (user.role !== 1 && user.role !== 2) {
    throw new ExternalApiError(E2003, 'Permission denied: only owner or admin can access external API');
  }

  return {
    userid: user.userid,
    role: user.role,
    accessToken: user.accessToken,
    proxyKey: user.proxyKey,
  };
}
