import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10059 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId?: string;
  };
}

interface Response10059Data {
  policySets: Array<{
    id: string;
    serverId: string | null;
    version: number;
    status: string;
    dsl: {
      rules: any[];
    };
    createdAt: string;
    updatedAt: string;
  }>;
}

/**
 * Protocol 10059 - Get Effective Policy
 * Returns only the active policy sets that apply to a given server (server-specific + global)
 * Forwards request to Core's GET_EFFECTIVE_POLICY (action 9105)
 */
export async function handleProtocol10059(body: Request10059): Promise<Response10059Data> {
  const { serverId } = body.params || {};
  const userid = body.common?.userid;

  console.log('[Protocol 10059] Get effective policy request:', { serverId, userid });

  try {
    const response = await makeProxyRequestWithUserId<Response10059Data>(
      AdminActionType.GET_EFFECTIVE_POLICY,
      { serverId },
      userid
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { message: response.error?.message || 'Failed to get effective policy' }
      );
    }

    console.log('[Protocol 10059] Effective policy retrieved successfully:', response.data?.policySets?.length || 0);

    return {
      policySets: response.data?.policySets || []
    };
  } catch (error) {
    console.error('[Protocol 10059] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
