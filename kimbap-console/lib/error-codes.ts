/**
 * Error codes and messages mapping
 * This allows for consistent error handling and easy internationalization
 */

export enum ErrorCode {
  // Common errors (1000-1999)
  MISSING_CMD_ID = 'ERR_1000',
  PROTOCOL_NOT_IMPLEMENTED = 'ERR_1001',
  INVALID_REQUEST = 'ERR_1002',
  INTERNAL_SERVER_ERROR = 'ERR_1003',
  
  // Authentication errors (2000-2999)
  MISSING_MASTER_PASSWORD = 'ERR_2000',
  INVALID_MASTER_PASSWORD = 'ERR_2001',
  UNAUTHORIZED = 'ERR_2002',
  FORBIDDEN = 'ERR_2003',
  TOKEN_EXPIRED = 'ERR_2004',
  TOKEN_INVALID = 'ERR_2005',
  INVALID_TOKEN = 'ERR_2005', // Alias for TOKEN_INVALID (same value)
  USER_DISABLED = 'ERR_2006',
  
  // Validation errors (3000-3999)
  MISSING_REQUIRED_FIELD = 'ERR_3000',
  INVALID_FIELD_FORMAT = 'ERR_3001',
  FIELD_TOO_LONG = 'ERR_3002',
  FIELD_TOO_SHORT = 'ERR_3003',
  
  // Database errors (4000-4999)
  DATABASE_CONNECTION_ERROR = 'ERR_4000',
  RECORD_NOT_FOUND = 'ERR_4001',
  DUPLICATE_RECORD = 'ERR_4002',
  TRANSACTION_FAILED = 'ERR_4003',
  
  // Business logic errors (5000-5999)
  PROXY_ALREADY_INITIALIZED = 'ERR_5000',
  SERVER_NOT_FOUND = 'ERR_5001',
  USER_NOT_FOUND = 'ERR_5002',
  PERMISSION_DENIED = 'ERR_5003',
  
  // License errors (6000-6099)
  LICENSE_ACTIVATION_FAILED = 'ERR_6000',
  LICENSE_EXPIRED = 'ERR_6001',
  HARDWARE_MISMATCH = 'ERR_6002',
  LICENSE_INVALID = 'ERR_6003',
  TOOL_CREATION_LIMIT_EXCEEDED = 'ERR_6004',
  ACCESS_TOKEN_LIMIT_EXCEEDED = 'ERR_6005',
  
  // Parameter errors
  INVALID_PARAMS = 'ERR_3004',
  
  // Configuration errors (7000-7999)
  KIMBAP_CORE_CONFIG_NOT_FOUND = 'ERR_7000',
}

// Default error messages in English
export const ErrorMessages: Record<ErrorCode, string> = {
  // Common errors
  [ErrorCode.MISSING_CMD_ID]: 'Missing cmdId in request',
  [ErrorCode.PROTOCOL_NOT_IMPLEMENTED]: 'Protocol {cmdId} not implemented',
  [ErrorCode.INVALID_REQUEST]: 'Invalid request format',
  [ErrorCode.INTERNAL_SERVER_ERROR]: 'Internal server error',
  
  // Authentication errors
  [ErrorCode.MISSING_MASTER_PASSWORD]: 'Master password is required',
  [ErrorCode.INVALID_MASTER_PASSWORD]: 'Invalid master password',
  [ErrorCode.UNAUTHORIZED]: 'Unauthorized',
  [ErrorCode.FORBIDDEN]: 'Forbidden',
  [ErrorCode.TOKEN_EXPIRED]: 'Token has expired',
  [ErrorCode.TOKEN_INVALID]: 'Invalid token',
  // [ErrorCode.INVALID_TOKEN]: 'Invalid token', // Same as TOKEN_INVALID, no need to duplicate
  [ErrorCode.USER_DISABLED]: 'User account is disabled',
  
  // Validation errors
  [ErrorCode.MISSING_REQUIRED_FIELD]: 'Invalid or missing parameter: {field}',
  [ErrorCode.INVALID_FIELD_FORMAT]: 'Invalid parameter format: {field}',
  [ErrorCode.FIELD_TOO_LONG]: 'Field {field} exceeds maximum length',
  [ErrorCode.FIELD_TOO_SHORT]: 'Field {field} is below minimum length',
  
  // Database errors
  [ErrorCode.DATABASE_CONNECTION_ERROR]: 'Database connection error',
  [ErrorCode.RECORD_NOT_FOUND]: 'Record not found',
  [ErrorCode.DUPLICATE_RECORD]: 'Record already exists',
  [ErrorCode.TRANSACTION_FAILED]: 'Transaction failed',
  
  // Business logic errors
  [ErrorCode.PROXY_ALREADY_INITIALIZED]: 'Proxy server already initialized',
  [ErrorCode.SERVER_NOT_FOUND]: 'Server not found',
  [ErrorCode.USER_NOT_FOUND]: 'User not found',
  [ErrorCode.PERMISSION_DENIED]: 'Permission denied',
  
  // License errors
  [ErrorCode.LICENSE_ACTIVATION_FAILED]: '{message}',
  [ErrorCode.LICENSE_EXPIRED]: 'License has expired',
  [ErrorCode.HARDWARE_MISMATCH]: 'License is not valid for this machine',
  [ErrorCode.LICENSE_INVALID]: 'Invalid license',
  [ErrorCode.TOOL_CREATION_LIMIT_EXCEEDED]: 'Tool creation limit exceeded. Please upgrade your plan to create more tools.',
  [ErrorCode.ACCESS_TOKEN_LIMIT_EXCEEDED]: 'Access token limit exceeded. Please upgrade your plan to create more tokens.',
  
  // Parameter errors  
  [ErrorCode.INVALID_PARAMS]: 'Invalid parameters',
  
  // Configuration errors
  [ErrorCode.KIMBAP_CORE_CONFIG_NOT_FOUND]: 'Kimbap Core configuration not found in database. Please configure Kimbap Core host and port first.',
};

/**
 * Get error message with parameter substitution
 */
export function getErrorMessage(code: ErrorCode, params?: Record<string, any>): string {
  let message = ErrorMessages[code] || 'Unknown error';
  
  if (params) {
    // Replace {param} with actual values
    Object.entries(params).forEach(([key, value]) => {
      message = message.replace(`{${key}}`, String(value));
    });
  }
  
  return message;
}

/**
 * Custom error class that includes error code
 */
export class ApiError extends Error {
  constructor(
    public code: ErrorCode,
    public statusCode: number = 400,
    public params?: Record<string, any>
  ) {
    super(getErrorMessage(code, params));
    this.name = 'ApiError';
  }
}

/**
 * HTTP status code mapping for error codes
 */
export const ErrorStatusCodes: Partial<Record<ErrorCode, number>> = {
  // 400 Bad Request
  [ErrorCode.MISSING_CMD_ID]: 400,
  [ErrorCode.INVALID_REQUEST]: 400,
  [ErrorCode.MISSING_REQUIRED_FIELD]: 400,
  [ErrorCode.INVALID_FIELD_FORMAT]: 400,
  [ErrorCode.FIELD_TOO_LONG]: 400,
  [ErrorCode.FIELD_TOO_SHORT]: 400,
  [ErrorCode.MISSING_MASTER_PASSWORD]: 400,
  
  // 401 Unauthorized
  [ErrorCode.UNAUTHORIZED]: 401,
  [ErrorCode.TOKEN_EXPIRED]: 401,
  [ErrorCode.TOKEN_INVALID]: 401,
  // [ErrorCode.INVALID_TOKEN]: 401, // Same as TOKEN_INVALID, no need to duplicate
  [ErrorCode.INVALID_MASTER_PASSWORD]: 401,
  
  // 403 Forbidden
  [ErrorCode.FORBIDDEN]: 403,
  [ErrorCode.PERMISSION_DENIED]: 403,
  [ErrorCode.USER_DISABLED]: 403,
  
  // 404 Not Found
  [ErrorCode.PROTOCOL_NOT_IMPLEMENTED]: 404,
  [ErrorCode.RECORD_NOT_FOUND]: 404,
  [ErrorCode.SERVER_NOT_FOUND]: 404,
  [ErrorCode.USER_NOT_FOUND]: 404,
  
  // 409 Conflict
  [ErrorCode.DUPLICATE_RECORD]: 409,
  [ErrorCode.PROXY_ALREADY_INITIALIZED]: 409,
  
  // 500 Internal Server Error
  [ErrorCode.INTERNAL_SERVER_ERROR]: 500,
  [ErrorCode.DATABASE_CONNECTION_ERROR]: 500,
  [ErrorCode.TRANSACTION_FAILED]: 500,
  [ErrorCode.KIMBAP_CORE_CONFIG_NOT_FOUND]: 500,
};

/**
 * Get HTTP status code for an error code
 */
export function getErrorStatusCode(code: ErrorCode): number {
  return ErrorStatusCodes[code] || 400;
}