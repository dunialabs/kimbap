import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10058 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    userId?: string;
  };
}

interface Response10058Data {
  count: number;
}

/**
 * Protocol 10058 - Count Pending Approvals
 * Gets the count of pending approval requests
 * Forwards request to Core
 */
export async function handleProtocol10058(body: Request10058): Promise<Response10058Data> {
  const { userId } = body.params || {};
  const userid = body.common?.userid;

  console.log('[Protocol 10058] Count pending approvals request:', { userId, userid });

  try {
    const response = await makeProxyRequestWithUserId<Response10058Data>(
      AdminActionType.COUNT_PENDING_APPROVALS,
      { userId },
      userid
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to count pending approvals' }
      );
    }

    console.log('[Protocol 10058] Pending approvals count:', response.data?.count || 0);

    return {
      count: response.data?.count || 0
    };
  } catch (error) {
    console.error('[Protocol 10058] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
