import { ExternalApiError, E1001, E1003, E3002, E5004, ErrorCode } from './error-codes';

export function throwCoreAdminError(message: string, notFoundCode: ErrorCode = E3002, coreCode?: number): never {
  const msg = message || 'Core admin request failed';
  const lower = msg.toLowerCase();

  if (coreCode === 1001) {
    if (lower.includes('missing required field')) {
      throw new ExternalApiError(E1001, msg);
    }
    if (lower.includes('not found')) {
      throw new ExternalApiError(notFoundCode, msg);
    }
    if (lower.includes('invalid field') || lower.includes('invalid decision') || lower.includes('invalid request')) {
      throw new ExternalApiError(E1003, msg);
    }
    throw new ExternalApiError(E5004, msg);
  }

  if (lower.includes('missing required field')) {
    throw new ExternalApiError(E1001, msg);
  }

  if (lower.includes('invalid field') || lower.includes('invalid decision') || lower.includes('invalid request')) {
    throw new ExternalApiError(E1003, msg);
  }

  if (lower.includes('not found')) {
    throw new ExternalApiError(notFoundCode, msg);
  }

  throw new ExternalApiError(E5004, msg);
}
