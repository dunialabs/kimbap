import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10054 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    serverId?: string;
  };
}

interface Response10054Data {
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
 * Protocol 10054 - List Tool Policies
 * Lists all policy sets, optionally filtered by serverId
 * Forwards request to Core
 */
export async function handleProtocol10054(body: Request10054): Promise<Response10054Data> {
  const { serverId } = body.params || {};
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10054] List tool policies request:', { serverId, userid });

  try {
    const response = await makeProxyRequestWithUserId<Response10054Data>(
      AdminActionType.GET_TOOL_POLICY,
      { serverId },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { details: response.error?.message || 'Failed to list tool policies' }
      );
    }

    console.log('[Protocol 10054] Tool policies listed successfully:', response.data?.policySets?.length || 0);

    return {
      policySets: response.data?.policySets || []
    };
  } catch (error) {
    console.error('[Protocol 10054] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
