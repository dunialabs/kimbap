import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10043 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId: string;
  };
}

interface Response10043Data {
  success: boolean;
  message: string;
}

/**
 * Protocol 10043 - Delete Server Skills
 * Deletes all skills for a Skills Server (used when deleting the entire Skills tool)
 * Forwards request to Core
 */
export async function handleProtocol10043(body: Request10043): Promise<Response10043Data> {
  const { serverId } = body.params || {};
  const userid = body.common?.userid;

  console.log('[Protocol 10043] Delete server skills request:', { serverId, userid });

  // Validate required fields
  if (!serverId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  try {
    // Forward request to Core
    const response = await makeProxyRequestWithUserId<Response10043Data>(
      AdminActionType.DELETE_SERVER_SKILLS,
      { serverId },
      userid
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to delete server skills' }
      );
    }

    console.log('[Protocol 10043] Server skills deleted successfully:', serverId);

    return {
      success: true,
      message: response.data?.message || 'Server skills deleted successfully'
    };
  } catch (error) {
    console.error('[Protocol 10043] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
