import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';
import { invalidateProxyAdminUrlCache } from '@/lib/proxy-api';
import { invalidateMcpGatewayValidationCache, validateAndCacheMcpGatewayUrl } from '@/lib/mcp-gateway-cache';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003, E1005, E4014 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

async function getKimbapCoreConnection(request: NextRequest) {
  await authenticate(request);

  const config = await prisma.config.findFirst();

  if (!config || !config.kimbap_core_host) {
    throw new ExternalApiError(E4014, 'Kimbap Core not configured');
  }

  const port: number | null = config.kimbap_core_port || null;

  return ApiResponse.success({
    host: config.kimbap_core_host,
    port,
    connected: true,
  }, 200, request);
}

interface ConnectInput {
  host?: unknown;
  port?: unknown;
}

interface NormalizedTarget {
  host: string;
  port: number;
  baseUrl: string;
}

function defaultPortForProtocol(protocol: string): number {
  return protocol === 'https:' ? 443 : 80;
}

function normalizeConnectTarget(input: ConnectInput): NormalizedTarget {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new ExternalApiError(E1001, 'Invalid request body');
  }

  if (typeof input.host !== 'string' || input.host.trim() === '') {
    throw new ExternalApiError(E1001, 'Missing required field: host');
  }
  const rawHost = input.host.trim();

  let explicitPort: number | undefined;
  if (input.port !== undefined && input.port !== null) {
    if (typeof input.port !== 'number' || !Number.isInteger(input.port)) {
      throw new ExternalApiError(E1003, 'Invalid field value: port must be an integer');
    }
    explicitPort = input.port;
    if (explicitPort < 1 || explicitPort > 65535) {
      throw new ExternalApiError(E1003, 'Invalid field value: port must be between 1 and 65535');
    }
  }

  const hasProtocol = rawHost.startsWith('http://') || rawHost.startsWith('https://');
  const candidate = hasProtocol
    ? rawHost.replace(/\/+$/, '')
    : `${explicitPort === 443 ? 'https' : 'http'}://${rawHost}`;

  let parsed: URL;
  try {
    parsed = new URL(candidate);
  } catch {
    throw new ExternalApiError(E1005, 'Invalid URL format for host');
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new ExternalApiError(E1005, 'Invalid URL format for host');
  }
  if (!parsed.hostname) {
    throw new ExternalApiError(E1005, 'Invalid URL format for host');
  }
  if (parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new ExternalApiError(E1003, 'Invalid field value: host must not include path, query, or hash');
  }

  const resolvedPort = explicitPort ?? (parsed.port ? parseInt(parsed.port, 10) : defaultPortForProtocol(parsed.protocol));
  const host = `${parsed.protocol}//${parsed.hostname}`;
  const baseUrl = resolvedPort === defaultPortForProtocol(parsed.protocol) ? host : `${host}:${resolvedPort}`;

  return { host, port: resolvedPort, baseUrl };
}

async function connectKimbapCore(request: NextRequest) {
  await authenticate(request);

  let body: ConnectInput;
  try {
    body = await request.json();
  } catch {
    throw new ExternalApiError(E1001, 'Invalid request body');
  }

  const target = normalizeConnectTarget(body);
  invalidateMcpGatewayValidationCache();
  const validation = await validateAndCacheMcpGatewayUrl(target.baseUrl);
  if (!validation.isValid) {
    throw new ExternalApiError(E4014, validation.errorMessage || 'Kimbap Core not available');
  }

  const existing = await prisma.config.findFirst();
  if (existing) {
    await prisma.config.update({
      where: { id: existing.id },
      data: {
        kimbap_core_host: target.host,
        kimbap_core_port: target.port,
      },
    });
  } else {
    await prisma.config.create({
      data: {
        kimbap_core_host: target.host,
        kimbap_core_port: target.port,
      },
    });
  }

  invalidateProxyAdminUrlCache();

  return ApiResponse.success({
    host: target.host,
    port: target.port,
    connected: true,
    isValid: 1,
  }, 200, request);
}

export async function POST(request: NextRequest) {
  try {
    return await connectKimbapCore(request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    return await getKimbapCoreConnection(request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}
