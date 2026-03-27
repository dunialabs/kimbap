/**
 * External API Error Codes
 *
 * Error Code Categories:
 * - E1xxx: Validation Errors (request parameter validation failures)
 * - E2xxx: Authentication Errors (authentication and authorization failures)
 * - E3xxx: Resource Errors (resource not found or state errors)
 * - E4xxx: Business Logic Errors (business rule violations)
 * - E5xxx: System Errors (internal server errors)
 */

// Validation Errors (E1xxx)
export const E1001 = 'E1001'; // Missing required field
export const E1002 = 'E1002'; // Invalid field type
export const E1003 = 'E1003'; // Invalid field value
export const E1004 = 'E1004'; // Invalid JSON format
export const E1005 = 'E1005'; // Invalid URL format
export const E1006 = 'E1006'; // Invalid IP address format
export const E1007 = 'E1007'; // Invalid email format
export const E1008 = 'E1008'; // Field length exceeded
export const E1009 = 'E1009'; // Field length too short
export const E1010 = 'E1010'; // Invalid enum value

// Authentication Errors (E2xxx)
export const E2001 = 'E2001'; // Access token is required
export const E2002 = 'E2002'; // Invalid access token
export const E2003 = 'E2003'; // Permission denied (not owner)
export const E2004 = 'E2004'; // Invalid master password
export const E2005 = 'E2005'; // Insufficient permissions
export const E2006 = 'E2006'; // Owner permission required
export const E2007 = 'E2007'; // Admin or owner permission required
export const E2008 = 'E2008'; // Rate limit exceeded

// Resource Errors (E3xxx)
export const E3001 = 'E3001'; // Proxy not found
export const E3002 = 'E3002'; // Tool not found
export const E3003 = 'E3003'; // Token not found
export const E3004 = 'E3004'; // Template not found
export const E3005 = 'E3005'; // DNS record not found
export const E3006 = 'E3006'; // IP whitelist entry not found
export const E3007 = 'E3007'; // Proxy already initialized
export const E3008 = 'E3008'; // Tool already enabled
export const E3009 = 'E3009'; // Tool already disabled
export const E3010 = 'E3010'; // Tunnel already enabled
export const E3011 = 'E3011'; // Tunnel already disabled

// Business Logic Errors (E4xxx)
export const E4001 = 'E4001'; // Tool limit reached
export const E4002 = 'E4002'; // Token limit reached/exceeded
export const E4003 = 'E4003'; // License expired
export const E4004 = 'E4004'; // Invalid license format
export const E4005 = 'E4005'; // License fingerprint mismatch
export const E4006 = 'E4006'; // Cannot delete owner token
export const E4007 = 'E4007'; // Cannot create owner role
export const E4008 = 'E4008'; // Tool start failed
export const E4009 = 'E4009'; // Tool connection failed
export const E4010 = 'E4010'; // Tunnel creation failed
export const E4011 = 'E4011'; // Invalid proxy key
export const E4012 = 'E4012'; // Backup decryption failed
export const E4013 = 'E4013'; // Invalid backup data
export const E4014 = 'E4014'; // Kimbap Core not available
export const E4015 = 'E4015'; // Kimbap Core validation failed

// System Errors (E5xxx)
export const E5001 = 'E5001'; // Database error
export const E5002 = 'E5002'; // Encryption error
export const E5003 = 'E5003'; // Internal server error
export const E5004 = 'E5004'; // External service error
export const E5005 = 'E5005'; // Service temporarily unavailable

/**
 * Error code type
 */
export type ErrorCode =
  | typeof E1001
  | typeof E1002
  | typeof E1003
  | typeof E1004
  | typeof E1005
  | typeof E1006
  | typeof E1007
  | typeof E1008
  | typeof E1009
  | typeof E1010
  | typeof E2001
  | typeof E2002
  | typeof E2003
  | typeof E2004
  | typeof E2005
  | typeof E2006
  | typeof E2007
  | typeof E2008
  | typeof E3001
  | typeof E3002
  | typeof E3003
  | typeof E3004
  | typeof E3005
  | typeof E3006
  | typeof E3007
  | typeof E3008
  | typeof E3009
  | typeof E3010
  | typeof E3011
  | typeof E4001
  | typeof E4002
  | typeof E4003
  | typeof E4004
  | typeof E4005
  | typeof E4006
  | typeof E4007
  | typeof E4008
  | typeof E4009
  | typeof E4010
  | typeof E4011
  | typeof E4012
  | typeof E4013
  | typeof E4014
  | typeof E4015
  | typeof E5001
  | typeof E5002
  | typeof E5003
  | typeof E5004
  | typeof E5005;

/**
 * HTTP status codes for error codes
 */
export const ErrorHttpStatus: Record<ErrorCode, number> = {
  // Validation errors - 400
  [E1001]: 400,
  [E1002]: 400,
  [E1003]: 400,
  [E1004]: 400,
  [E1005]: 400,
  [E1006]: 400,
  [E1007]: 400,
  [E1008]: 400,
  [E1009]: 400,
  [E1010]: 400,
  // Authentication errors - 401/403/429
  [E2001]: 401,
  [E2002]: 401,
  [E2003]: 403,
  [E2004]: 401,
  [E2005]: 403,
  [E2006]: 403,
  [E2007]: 403,
  [E2008]: 429,
  // Resource errors - 404/409
  [E3001]: 404,
  [E3002]: 404,
  [E3003]: 404,
  [E3004]: 404,
  [E3005]: 404,
  [E3006]: 404,
  [E3007]: 409,
  [E3008]: 409,
  [E3009]: 409,
  [E3010]: 409,
  [E3011]: 409,
  // Business logic errors - 400/422
  [E4001]: 422,
  [E4002]: 422,
  [E4003]: 422,
  [E4004]: 422,
  [E4005]: 422,
  [E4006]: 422,
  [E4007]: 422,
  [E4008]: 400,
  [E4009]: 400,
  [E4010]: 400,
  [E4011]: 400,
  [E4012]: 400,
  [E4013]: 400,
  [E4014]: 422,
  [E4015]: 422,
  // System errors - 500/503
  [E5001]: 500,
  [E5002]: 500,
  [E5003]: 500,
  [E5004]: 500,
  [E5005]: 503,
};

/**
 * Get HTTP status code for an error code
 */
export function getHttpStatus(code: ErrorCode): number {
  return ErrorHttpStatus[code] || 500;
}

/**
 * API Error class for external API
 */
export class ExternalApiError extends Error {
  constructor(
    public code: ErrorCode,
    public override message: string
  ) {
    super(message);
    this.name = 'ExternalApiError';
  }

  get httpStatus(): number {
    return getHttpStatus(this.code);
  }
}
