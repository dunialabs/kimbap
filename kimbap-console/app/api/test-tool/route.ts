import { NextRequest, NextResponse } from 'next/server';

import { executeHttpRequest, prepareToolRequest } from '@/lib/rest-api-test-client';
import type { APIDefinition, ToolDefinition } from '@/lib/rest-api-utils';
import {
  ToolDefinitionSchema,
  applyResponseTransform,
  validateRestApiConfig,
} from '@/lib/rest-api-utils';
import {
  collectResponseHeaders,
  extractResponseBody,
  formatBodyPreview,
} from '@/lib/http-response-utils';
import { verifyToken } from '@/lib/auth';

export const dynamic = 'force-dynamic';
export const revalidate = 0;

export async function POST(request: NextRequest) {
  try {
    const authHeader = request.headers.get('authorization') || '';
    if (!authHeader.startsWith('Bearer ') || !authHeader.slice(7).trim()) {
      return NextResponse.json(
        { success: false, error: 'Authentication required' },
        { status: 401 },
      );
    }
    let tokenPayload;
    try {
      tokenPayload = verifyToken(authHeader.slice(7).trim());
    } catch {
      return NextResponse.json(
        { success: false, error: 'Invalid or expired token' },
        { status: 401 },
      );
    }
    if (tokenPayload.role !== 'admin' && tokenPayload.role !== 'owner') {
      return NextResponse.json({ success: false, error: 'Admin access required' }, { status: 403 });
    }

    const body = await request.json();
    const apiConfig = body?.apiConfig as APIDefinition | undefined;
    const toolInput = body?.tool as ToolDefinition | undefined;
    const testParams = (body?.testParams || {}) as Record<string, any>;

    if (!apiConfig || !toolInput) {
      return NextResponse.json(
        { success: false, error: 'Request body must include "apiConfig" and "tool".' },
        { status: 400 },
      );
    }

    const configValidation = validateRestApiConfig(apiConfig);
    if (!configValidation.valid) {
      return NextResponse.json(
        { success: false, error: configValidation.error || 'REST API configuration is invalid.' },
        { status: 422 },
      );
    }

    const toolValidation = ToolDefinitionSchema.safeParse(toolInput);
    if (!toolValidation.success) {
      const issue = toolValidation.error.issues[0];
      const message = `${issue?.path?.join('.') || 'tool'}: ${issue?.message || 'Invalid tool definition'}`;
      return NextResponse.json({ success: false, error: message }, { status: 422 });
    }

    const prepared = prepareToolRequest(apiConfig, toolValidation.data, testParams);
    if (
      prepared.missingParams.length > 0 ||
      prepared.invalidParams.length > 0 ||
      !prepared.request
    ) {
      return NextResponse.json(
        {
          success: false,
          error: 'Tool parameters are incomplete or invalid.',
          missingParams: prepared.missingParams,
          invalidParams: prepared.invalidParams,
        },
        { status: 422 },
      );
    }

    const { response, durationMs } = await executeHttpRequest(
      prepared.request,
      prepared.request.method,
    );
    const responseClone = response.clone();
    const responseBody = await extractResponseBody(responseClone);
    const responseHeaders = collectResponseHeaders(response);
    const bodyPreview = formatBodyPreview(responseBody);
    const transformed = applyResponseTransform(responseBody, toolValidation.data.response);

    return NextResponse.json({
      success: response.ok,
      message: response.ok
        ? 'Tool executed successfully.'
        : `Request failed with status ${response.status}.`,
      statusCode: response.status,
      statusText: response.statusText,
      responseTime: durationMs,
      request: {
        method: prepared.request.method,
        url: prepared.request.maskedUrl,
        headers: prepared.request.maskedHeaders,
        body: formatBodyPreview(prepared.request.body, 4000),
      },
      response: {
        statusCode: response.status,
        statusText: response.statusText,
        headers: responseHeaders,
        body: responseBody,
        bodyPreview,
      },
      transformedResponse: transformed,
      warnings: prepared.pathWarnings,
      missingParams: prepared.missingParams,
      invalidParams: prepared.invalidParams,
    });
  } catch (error: any) {
    console.error('Failed to execute tool test:', error);
    return NextResponse.json(
      { success: false, error: error?.message || 'Failed to execute tool test.' },
      { status: 500 },
    );
  }
}
