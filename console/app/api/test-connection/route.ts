import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/lib/prisma';
import { hashToken } from '@/lib/auth';

import {
  executeHttpRequest,
  prepareConnectionRequest
} from '@/lib/rest-api-test-client';
import type { PreparedRequest } from '@/lib/rest-api-test-client';
import type { APIDefinition } from '@/lib/rest-api-utils';
import {
  collectResponseHeaders,
  extractResponseBody,
  formatBodyPreview
} from '@/lib/http-response-utils';
import { validateRestApiConfig } from '@/lib/rest-api-utils';

export const dynamic = 'force-dynamic';
export const revalidate = 0;

export async function POST(request: NextRequest) {
  try {
    const authHeader = request.headers.get('authorization') || '';
    const token = authHeader.startsWith('Bearer ') ? authHeader.slice(7).trim() : null;
    if (!token) {
      return NextResponse.json({ success: false, error: 'Unauthorized' }, { status: 401 });
    }
    const authUser = await prisma.user.findUnique({ where: { accessTokenHash: hashToken(token) } });
    if (!authUser) {
      return NextResponse.json({ success: false, error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const apiConfig = body?.apiConfig as APIDefinition | undefined;

    if (!apiConfig) {
      return NextResponse.json(
        { success: false, error: 'Request body must include "apiConfig".' },
        { status: 400 }
      );
    }

    const validation = validateRestApiConfig(apiConfig);
    if (!validation.valid) {
      return NextResponse.json(
        { success: false, error: validation.error || 'REST API configuration is invalid.' },
        { status: 422 }
      );
    }

    let prepared: PreparedRequest;
    try {
      prepared = prepareConnectionRequest(apiConfig);
    } catch (error: any) {
      return NextResponse.json(
        { success: false, error: error?.message || 'Unable to prepare test request.' },
        { status: 400 }
      );
    }

    const methods: Array<'HEAD' | 'GET'> = ['HEAD', 'GET'];
    let lastError: string | null = null;

    for (let i = 0; i < methods.length; i++) {
      const method = methods[i];
      try {
        const { response, durationMs } = await executeHttpRequest(prepared, method);
        const statusCode = response.status;
        const statusText = response.statusText;
        const responseHeaders = collectResponseHeaders(response);
        const shouldReadBody = method === 'GET' || statusCode >= 400;
        const bodyClone = shouldReadBody ? response.clone() : null;
        const responseBody = shouldReadBody && bodyClone ? await extractResponseBody(bodyClone) : undefined;
        const bodyPreview = formatBodyPreview(responseBody);

        if (statusCode === 401 || statusCode === 403) {
          return NextResponse.json({
            success: false,
            message: 'Authentication failed. Verify your credentials and auth settings.',
            statusCode,
            method,
            responseTime: durationMs,
            request: buildRequestSummary(prepared, method),
            response: {
              statusCode,
              statusText,
              headers: responseHeaders,
              bodyPreview
            },
            auth: {
              type: apiConfig.auth?.type || 'none',
              passed: false
            }
          });
        }

        if (statusCode >= 200 && statusCode < 400) {
          return NextResponse.json({
            success: true,
            message: 'API connection successful.',
            statusCode,
            method,
            responseTime: durationMs,
            request: buildRequestSummary(prepared, method),
            response: {
              statusCode,
              statusText,
              headers: responseHeaders,
              bodyPreview
            },
            auth: {
              type: apiConfig.auth?.type || 'none',
              passed: true
            }
          });
        }

        if (method === 'HEAD' && (statusCode === 405 || statusCode === 501)) {
          continue;
        }

        if (method === 'HEAD') {
          continue;
        }

        return NextResponse.json({
          success: false,
          message: `Request failed with status ${statusCode}.`,
          statusCode,
          method,
          responseTime: durationMs,
          request: buildRequestSummary(prepared, method),
          response: {
            statusCode,
            statusText,
            headers: responseHeaders,
            bodyPreview
          },
          auth: {
            type: apiConfig.auth?.type || 'none',
            passed: false
          }
        });
      } catch (error: any) {
        lastError = error?.message || 'Request failed.';
      }
    }

    return NextResponse.json({
      success: false,
      message: lastError || 'Unable to reach API base URL.',
      request: buildRequestSummary(prepared, 'HEAD'),
      auth: {
        type: apiConfig.auth?.type || 'none',
        passed: false
      }
    });
  } catch (error: any) {
    console.error('Failed to run API connection test:', error);
    return NextResponse.json(
      { success: false, error: error?.message || 'Failed to run API connection test.' },
      { status: 500 }
    );
  }
}

function buildRequestSummary(request: PreparedRequest, method: string) {
  return {
    method,
    url: request.maskedUrl,
    headers: request.maskedHeaders
  };
}
