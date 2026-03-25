import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10052 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    id: string;
    dsl?: {
      rules: any[];
    };
    status?: string;
  };
}

interface Response10052Data {
  id: string;
  serverId: string | null;
  version: number;
  status: string;
  dsl: {
    rules: any[];
  };
  createdAt: string;
  updatedAt: string;
}

/**
 * Protocol 10052 - Update Tool Policy
 * Updates an existing policy set
 * Forwards request to Core
 */
export async function handleProtocol10052(body: Request10052): Promise<Response10052Data> {
  const { id, dsl, status } = body.params || {};
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10052] Update tool policy request:', { id, userid });

  if (!id) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'id' });
  }

  if (dsl && !Array.isArray(dsl.rules)) {
    throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { field: 'dsl.rules' });
  }

  try {
    const response = await makeProxyRequestWithUserId<Response10052Data>(
      AdminActionType.UPDATE_TOOL_POLICY,
      { id, dsl, status },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to update tool policy' }
      );
    }

    console.log('[Protocol 10052] Tool policy updated successfully:', response.data?.id);

    return response.data!;
  } catch (error) {
    console.error('[Protocol 10052] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
