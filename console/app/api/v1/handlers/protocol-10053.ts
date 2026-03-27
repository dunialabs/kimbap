import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10053 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    id: string;
  };
}

interface Response10053Data {
  success: boolean;
}

/**
 * Protocol 10053 - Delete Tool Policy
 * Deletes a policy set by ID
 * Forwards request to Core
 */
export async function handleProtocol10053(body: Request10053): Promise<Response10053Data> {
  const { id } = body.params || {};
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10053] Delete tool policy request:', { id, userid });

  if (!id) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'id' });
  }

  try {
    const response = await makeProxyRequestWithUserId<Response10053Data>(
      AdminActionType.DELETE_TOOL_POLICY,
      { id },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to delete tool policy' }
      );
    }

    console.log('[Protocol 10053] Tool policy deleted successfully:', id);

    return { success: true };
  } catch (error) {
    console.error('[Protocol 10053] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
