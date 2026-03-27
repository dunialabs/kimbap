import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10055 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    userId?: string;
    serverId?: string;
    toolName?: string;
    status?: string;
    page?: number;
    pageSize?: number;
  };
}

interface Response10055Data {
  page: number;
  pageSize: number;
  hasMore: boolean;
  requests: Array<{
    id: string;
    resumeToken?: string;
    userId: string;
    serverId: string | null;
    toolName: string;
    canonicalArgs: Record<string, any>;
    redactedArgs: Record<string, any>;
    requestHash: string;
    status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'EXPIRED' | 'EXECUTING' | 'EXECUTED' | 'FAILED';
    reason?: string | null;
    decisionReason?: string;
    decidedByUserId?: string | null;
    decidedByRole?: number | null;
    decisionChannel?: 'admin_api' | 'socket' | null;
    executedAt?: string | null;
    executionError?: string | null;
    executionResultAvailable?: boolean;
    policyVersion: number;
    uniformRequestId?: string | null;
    createdAt: string;
    expiresAt: string;
    decidedAt?: string;
  }>;
}

/**
 * Protocol 10055 - List Approval Requests
 * Lists approval requests with optional filters
 * Forwards request to Core
 */
export async function handleProtocol10055(body: Request10055): Promise<Response10055Data> {
  const { userId, serverId, toolName, status, page, pageSize } = body.params || {};
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10055] List approval requests:', {
    userId,
    serverId,
    toolName,
    status,
    page,
    pageSize,
    userid,
  });

  try {
    const response = await makeProxyRequestWithUserId<Response10055Data>(
      AdminActionType.LIST_APPROVAL_REQUESTS,
      { userId, serverId, toolName, status, page, pageSize },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        message: response.error?.message || 'Failed to list approval requests',
      });
    }

    console.log(
      '[Protocol 10055] Approval requests listed successfully:',
      response.data?.requests?.length || 0,
    );

    return {
      page: response.data?.page || 1,
      pageSize: response.data?.pageSize || 20,
      hasMore: response.data?.hasMore || false,
      requests: response.data?.requests || [],
    };
  } catch (error) {
    console.error('[Protocol 10055] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
