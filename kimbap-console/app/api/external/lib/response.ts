import { NextResponse } from 'next/server';
import { ErrorCode, getHttpStatus, ExternalApiError } from './error-codes';

/**
 * Success response format
 */
interface SuccessResponse<T = unknown> {
  success: true;
  data: T;
}

/**
 * Error response format
 */
interface ErrorResponse {
  success: false;
  error: {
    code: ErrorCode;
    message: string;
  };
}

/**
 * API Response helper for external API
 */
export class ApiResponse {
  private static readonly cacheHeaders = {
    'Cache-Control': 'private, no-store',
    'Vary': 'Authorization',
  };

  static success<T>(data: T, status: number = 200): NextResponse<SuccessResponse<T>> {
    return NextResponse.json(
      {
        success: true as const,
        data,
      },
      { status, headers: this.cacheHeaders }
    );
  }

  /**
   * Return an error response with specific error code
   */
  static error(code: ErrorCode, message: string): NextResponse<ErrorResponse> {
    const status = getHttpStatus(code);
    return NextResponse.json(
      {
        success: false as const,
        error: {
          code,
          message,
        },
      },
      { status, headers: this.cacheHeaders }
    );
  }

  static handleError(error: unknown): NextResponse<ErrorResponse> {
    // If it's our custom ExternalApiError
    if (error instanceof ExternalApiError) {
      return this.error(error.code, error.message);
    }

    // Log unexpected errors
    console.error('Unexpected error:', error);

    // Return generic internal error
    return this.error('E5003', 'Internal server error');
  }
}
