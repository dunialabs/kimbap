import { ApiError, ErrorCode } from '@/lib/error-codes';
import { purgeCacheExact } from '@/lib/proxy-api';

interface Request10067 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    operation?: string;
    serverId?: string;
    entityId?: string;
    policy?: Record<string, unknown>;
    scopeContext?: Record<string, unknown>;
    requestParams?: unknown;
    reason?: string;
  };
}

interface Response10067Data {
  success: true;
}

export async function handleProtocol10067(body: Request10067): Promise<Response10067Data> {
  const userid = body.common?.userid;
  const { operation, serverId, entityId, policy, scopeContext, requestParams, reason } =
    body.params || {};

  if (operation !== undefined && typeof operation !== 'string') {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'operation' });
  }
  if (serverId !== undefined && typeof serverId !== 'string') {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'serverId' });
  }
  if (entityId !== undefined && typeof entityId !== 'string') {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'entityId' });
  }

  const normalizedOperation = operation?.trim();
  const normalizedServerId = serverId?.trim();
  const normalizedEntityId = entityId?.trim();

  if (!normalizedOperation) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'operation' });
  }

  if (!['tool', 'resource', 'prompt'].includes(normalizedOperation)) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'operation' });
  }

  if (!normalizedServerId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  if (!normalizedEntityId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'entityId' });
  }

  if (policy === undefined) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'policy' });
  }

  if (!policy || typeof policy !== 'object' || Array.isArray(policy)) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'policy' });
  }

  await purgeCacheExact(
    normalizedOperation,
    normalizedServerId,
    normalizedEntityId,
    policy,
    scopeContext,
    requestParams,
    reason,
    userid,
  );
  return { success: true };
}
