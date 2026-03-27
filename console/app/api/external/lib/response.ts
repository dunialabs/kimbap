import { NextResponse, type NextRequest } from 'next/server';
import { ErrorCode, getHttpStatus, ExternalApiError } from './error-codes';

interface SuccessResponse<T = unknown> {
  success: true;
  data: T;
  request_id?: string;
}

interface ErrorResponse {
  success: false;
  error: {
    code: ErrorCode;
    message: string;
    retryable: boolean;
  };
  request_id?: string;
}

const RETRYABLE_CODES = new Set<string>(['E5004', 'E5005', 'E2008']);

export class ApiResponse {
  private static readonly cacheHeaders = {
    'Cache-Control': 'private, no-store',
    'Vary': 'Authorization',
  };

  private static getRequestId(request?: NextRequest): string | undefined {
    const raw = request?.headers.get('x-request-id');
    if (!raw) return undefined;
    const id = raw.trim();
    if (id.length === 0 || id.length > 128) return undefined;
    if (!/^[a-zA-Z0-9._-]+$/.test(id)) return undefined;
    return id;
  }

  static success<T>(data: T, status: number = 200, request?: NextRequest): NextResponse<SuccessResponse<T>> {
    const requestId = this.getRequestId(request);
    return NextResponse.json(
      {
        success: true as const,
        data,
        ...(requestId && { request_id: requestId }),
      },
      { status, headers: this.cacheHeaders }
    );
  }

  static error(code: ErrorCode, message: string, request?: NextRequest): NextResponse<ErrorResponse> {
    const status = getHttpStatus(code);
    const requestId = this.getRequestId(request);
    return NextResponse.json(
      {
        success: false as const,
        error: {
          code,
          message,
          retryable: RETRYABLE_CODES.has(code),
        },
        ...(requestId && { request_id: requestId }),
      },
      { status, headers: this.cacheHeaders }
    );
  }

  static handleError(error: unknown, request?: NextRequest): NextResponse<ErrorResponse> {
    if (error instanceof ExternalApiError) {
      return this.error(error.code, error.message, request);
    }

    console.error('Unexpected error:', error);
    return this.error('E5003', 'Internal server error', request);
  }
}
