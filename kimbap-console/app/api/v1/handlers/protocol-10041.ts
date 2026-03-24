import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10041 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId: string;
    data: string; // ZIP file content as base64 encoded string
  };
}

interface Response10041Data {
  success: boolean;
  message: string;
  skillName?: string;
}

/**
 * Protocol 10041 - Upload Skills Bundle
 * Uploads a ZIP file containing skills to a specific Skills Server
 * The ZIP will be extracted and skill directories (containing SKILL.md) will be identified
 * Forwards request to Core
 */
export async function handleProtocol10041(body: Request10041): Promise<Response10041Data> {
  const { serverId, data } = body.params || {};
  const userid = body.common?.userid;

  console.log('[Protocol 10041] Upload skills request:', {
    serverId,
    userid,
    dataSize: data?.length || 0
  });

  // Validate required fields
  if (!serverId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  if (!data || typeof data !== 'string' || data.length === 0) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'data' });
  }

  try {
    // Forward request to Core
    const response = await makeProxyRequestWithUserId<Response10041Data>(
      AdminActionType.UPLOAD_SKILLS,
      { serverId, data },
      userid,
      undefined,
      120000 // 2 minute timeout for large uploads
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to upload skills' }
      );
    }

    console.log('[Protocol 10041] Skills uploaded successfully');

    return {
      success: true,
      message: response.data?.message || 'Skills uploaded successfully',
      skillName: response.data?.skillName
    };
  } catch (error) {
    console.error('[Protocol 10041] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
