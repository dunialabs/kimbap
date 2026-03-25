import crypto from 'crypto';
import { prisma } from '@/lib/prisma';

export const hashToken = (token: string): string => {
  return crypto.createHash('sha256').update(token).digest('hex');
};

export const maskToken = (token: string): string => {
  if (token.length <= 10) return token;
  return `${token.substring(0, 8)}****...${token.substring(token.length - 4)}`;
};

export const getUserFromRequest = async (request: Request) => {
  const authHeader = request.headers.get('authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return null;
  }

  const token = authHeader.substring(7);
  try {
    const user = await prisma.user.findFirst({
      where: { accessTokenHash: hashToken(token) },
    });
    return user;
  } catch {
    return null;
  }
};
