import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10056 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    id: string;
  };
}

interface Response10056Data {
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
  executedAt?: string | null;
  executionError?: string | null;
  executionResultAvailable?: boolean;
  policyVersion: number;
  uniformRequestId?: string | null;
  createdAt: string;
  expiresAt: string;
  decidedAt?: string;
}

/**
 * Protocol 10056 - Get Approval Request
 * Gets a single approval request by ID
 * Forwards request to Core
 */
export async function handleProtocol10056(body: Request10056): Promise<Response10056Data> {
  const rawID = body.params?.id;
  const id = typeof rawID === 'string' ? rawID.trim() : '';
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10056] Get approval request:', { id, userid });

  if (!id) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'id' });
  }

  try {
    const response = await makeProxyRequestWithUserId<Response10056Data>(
      AdminActionType.GET_APPROVAL_REQUEST,
      { id },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        message: response.error?.message || 'Failed to get approval request',
      });
    }

    console.log('[Protocol 10056] Approval request retrieved successfully:', response.data?.id);

    return response.data!;
  } catch (error) {
    console.error('[Protocol 10056] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
