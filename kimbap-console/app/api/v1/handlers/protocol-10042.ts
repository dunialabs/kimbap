import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10042 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId: string;
    skillName: string;
  };
}

interface Response10042Data {
  success: boolean;
  message: string;
}

/**
 * Protocol 10042 - Delete Skill
 * Deletes a specific skill from a Skills Server
 * Forwards request to Core
 */
export async function handleProtocol10042(body: Request10042): Promise<Response10042Data> {
  const { serverId, skillName } = body.params || {};
  const userid = body.common?.userid;

  console.log('[Protocol 10042] Delete skill request:', { serverId, skillName, userid });

  // Validate required fields
  if (!serverId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  if (!skillName) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'skillName' });
  }

  try {
    // Forward request to Core
    const response = await makeProxyRequestWithUserId<Response10042Data>(
      AdminActionType.DELETE_SKILL,
      { serverId, skillName },
      userid
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to delete skill' }
      );
    }

    console.log('[Protocol 10042] Skill deleted successfully:', skillName);

    return {
      success: true,
      message: response.data?.message || 'Skill deleted successfully'
    };
  } catch (error) {
    console.error('[Protocol 10042] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
