import { NextResponse } from 'next/server';
import { ErrorCode, getErrorMessage, getErrorStatusCode, ApiError } from './error-codes';

interface CommonResponse {
  code: number;
  message: string;
  cmdId: number;
  errorCode?: string; // Add error code for easier debugging
}

interface ApiResponseData<T = any> {
  common: CommonResponse;
  data?: T;
}

export class ApiResponse {
  /**
   * Success response
   */
  static success<T = any>(cmdId: number, data?: T): NextResponse {
    const response: ApiResponseData<T> = {
      common: {
        code: 0,
        message: 'Success',
        cmdId: cmdId
      }
    };

    if (data !== undefined) {
      response.data = data;
    }

    return NextResponse.json(response);
  }

  /**
   * Error response with error code
   */
  static errorWithCode(
    cmdId: number, 
    errorCode: ErrorCode, 
    params?: Record<string, any>
  ): NextResponse {
    const statusCode = getErrorStatusCode(errorCode);
    let message = getErrorMessage(errorCode, params);
    
    // If there are details in params, append them to the message
    if (params?.details) {
      message = `${message}: ${params.details}`;
    }
    
    const response: ApiResponseData = {
      common: {
        code: statusCode,
        message: message,
        cmdId: cmdId,
        errorCode: errorCode
      }
    };

    return NextResponse.json(response, { status: statusCode });
  }

  /**
   * Error response (legacy, for custom errors)
   */
  static error(cmdId: number, code: number, message: string, status: number = 400): NextResponse {
    const response: ApiResponseData = {
      common: {
        code: code,
        message: message,
        cmdId: cmdId
      }
    };

    return NextResponse.json(response, { status });
  }

  /**
   * Handle errors automatically
   */
  static handleError(cmdId: number, error: unknown): NextResponse {
    // If it's our custom ApiError, use the error code with params
    if (error instanceof ApiError) {
      return this.errorWithCode(cmdId, error.code, error.params);
    }
    
    // Otherwise, log and return generic error
    console.error(`Error in protocol ${cmdId}:`, error);
    const message = error instanceof Error ? error.message : 'Internal server error';
    return this.errorWithCode(cmdId, ErrorCode.INTERNAL_SERVER_ERROR);
  }

  /**
   * Common error responses using error codes
   */
  static missingCmdId(): NextResponse {
    return this.errorWithCode(0, ErrorCode.MISSING_CMD_ID);
  }

  static protocolNotImplemented(cmdId: number): NextResponse {
    return this.errorWithCode(cmdId, ErrorCode.PROTOCOL_NOT_IMPLEMENTED, { cmdId });
  }

  static missingRequiredField(cmdId: number, field: string): NextResponse {
    return this.errorWithCode(cmdId, ErrorCode.MISSING_REQUIRED_FIELD, { field });
  }

  static unauthorized(cmdId: number): NextResponse {
    return this.errorWithCode(cmdId, ErrorCode.UNAUTHORIZED);
  }

  static forbidden(cmdId: number): NextResponse {
    return this.errorWithCode(cmdId, ErrorCode.FORBIDDEN);
  }
}