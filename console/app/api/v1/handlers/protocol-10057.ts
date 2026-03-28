import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface Request10057 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    id: string;
    decision: 'APPROVED' | 'REJECTED';
    reason?: string;
  };
}

interface Response10057Data {
  id: string;
  resumeToken?: string;
  status: 'APPROVED' | 'REJECTED' | 'EXECUTING' | 'EXECUTED' | 'FAILED';
  decisionReason?: string;
  decidedAt: string;
  executionResultAvailable?: boolean;
}

/**
 * Protocol 10057 - Decide Approval Request
 * Approves or rejects a pending approval request
 * Forwards request to Core
 */
export async function handleProtocol10057(body: Request10057): Promise<Response10057Data> {
  const rawID = body.params?.id;
  const rawDecision = body.params?.decision;
  const id = typeof rawID === 'string' ? rawID.trim() : '';
  const decision = typeof rawDecision === 'string' ? rawDecision.trim() : rawDecision;
  const reason = typeof body.params?.reason === 'string' ? body.params.reason.trim() : body.params?.reason;
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10057] Decide approval request:', { id, decision, userid });

  if (!id) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'id' });
  }

  if (!decision || !['APPROVED', 'REJECTED'].includes(decision)) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'decision' });
  }

  try {
    const response = await makeProxyRequestWithUserId<Response10057Data>(
      AdminActionType.DECIDE_APPROVAL_REQUEST,
      { id, decision, reason },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        message: response.error?.message || 'Failed to decide approval request',
      });
    }

    console.log(
      '[Protocol 10057] Approval request decided successfully:',
      response.data?.id,
      response.data?.status,
    );

    return response.data!;
  } catch (error) {
    console.error('[Protocol 10057] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
