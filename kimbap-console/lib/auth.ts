import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';
import crypto from 'crypto';
import { prisma } from '@/lib/prisma';

const JWT_SECRET = process.env.JWT_SECRET;

if (!JWT_SECRET) {
  throw new Error(
    'FATAL: JWT_SECRET environment variable is not set. ' +
    'The application cannot start without a secure JWT secret. ' +
    'Set JWT_SECRET in your .env.local file or environment configuration.'
  );
}

export interface JWTPayload {
  userId: string;
  email: string;
  role: string;
}

export const hashPassword = async (password: string): Promise<string> => {
  return bcrypt.hash(password, 10);
};

export const comparePassword = async (password: string, hash: string): Promise<boolean> => {
  return bcrypt.compare(password, hash);
};

export const generateToken = (payload: JWTPayload): string => {
  return jwt.sign(payload, JWT_SECRET, { expiresIn: '7d' });
};

export const verifyToken = (token: string): JWTPayload => {
  return jwt.verify(token, JWT_SECRET) as JWTPayload;
};

export const generateVerificationCode = (): string => {
  return crypto.randomInt(100000, 1000000).toString();
};

export const generateAccessToken = (): string => {
  const prefix = 'kimbap_';
  return prefix + crypto.randomBytes(24).toString('hex');
};

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
    const payload = verifyToken(token);
    const user = await prisma.user.findUnique({
      where: { userid: payload.userId }
    });
    return user;
  } catch (error) {
    return null;
  }
};

export const requireAuth = async (request: Request) => {
  const user = await getUserFromRequest(request);
  if (!user) {
    throw new Error('Unauthorized');
  }
  return user;
};
