import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10040 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId: string;
  };
}

interface SkillInfo {
  name: string;
  description: string;
  version: string;
}

interface Response10040Data {
  skills: SkillInfo[];
}

/**
 * Protocol 10040 - List Skills
 * Lists all skills for a specific Skills Server
 * Forwards request to Core
 */
export async function handleProtocol10040(body: Request10040): Promise<Response10040Data> {
  const { serverId } = body.params || {};
  const userid = body.common?.userid;

  console.log('[Protocol 10040] List skills request:', { serverId, userid });

  // Validate required fields
  if (!serverId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  try {
    // Forward request to Core
    const response = await makeProxyRequestWithUserId<Response10040Data>(
      AdminActionType.LIST_SKILLS,
      { serverId },
      userid
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to list skills' }
      );
    }

    console.log('[Protocol 10040] Skills listed successfully:', response.data?.skills?.length || 0);

    return {
      skills: response.data?.skills || []
    };
  } catch (error) {
    console.error('[Protocol 10040] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
