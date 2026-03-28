import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10051 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    id: string;
  };
}

interface Response10051Data {
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
 * Protocol 10051 - Get Tool Policy
 * Gets a single policy set by ID
 * Forwards request to Core
 */
export async function handleProtocol10051(body: Request10051): Promise<Response10051Data> {
  const rawID = body.params?.id;
  const id = typeof rawID === 'string' ? rawID.trim() : '';
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10051] Get tool policy request:', { id, userid });

  if (!id) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'id' });
  }

  try {
    const response = await makeProxyRequestWithUserId<Response10051Data>(
      AdminActionType.GET_TOOL_POLICY,
      { id },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to get tool policy' }
      );
    }

    console.log('[Protocol 10051] Tool policy retrieved successfully:', response.data?.id);

    return response.data!;
  } catch (error) {
    console.error('[Protocol 10051] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
