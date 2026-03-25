import { ApiError, ErrorCode } from '@/lib/error-codes';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';

interface PolicyRule {
  id: string;
  priority?: number;
  match: {
    tool?: string;
    serverId?: string;
  };
  extract?: Record<string, {
    path: string;
    type: 'string' | 'number' | 'boolean' | 'url.host' | 'bytes.length';
  }>;
  when?: Array<{
    left: string;
    op: 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'not_in' | 'matches';
    right: any;
  }>;
  effect: {
    decision: 'ALLOW' | 'REQUIRE_APPROVAL' | 'DENY';
    reason?: string;
  };
}

interface Request10050 {
  common: {
    cmdId: number;
    userid: string;
    rawToken?: string;
  };
  params: {
    serverId?: string;
    dsl: {
      rules: PolicyRule[];
    };
  };
}

interface Response10050Data {
  id: string;
  serverId: string | null;
  version: number;
  status: string;
  dsl: {
    rules: PolicyRule[];
  };
  createdAt: string;
  updatedAt: string;
}

/**
 * Protocol 10050 - Create Tool Policy
 * Creates a new content-aware tool policy set
 * Forwards request to Core
 */
export async function handleProtocol10050(body: Request10050): Promise<Response10050Data> {
  const { serverId, dsl } = body.params || {};
  const userid = body.common?.userid;
  const rawToken = body.common?.rawToken;

  console.log('[Protocol 10050] Create tool policy request:', { serverId, userid });

  if (!dsl || !Array.isArray(dsl.rules)) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'dsl.rules' });
  }

  try {
    const response = await makeProxyRequestWithUserId<Response10050Data>(
      AdminActionType.CREATE_TOOL_POLICY,
      { serverId, dsl },
      userid,
      rawToken,
    );

    if (!response.success) {
      throw new ApiError(
        ErrorCode.INTERNAL_SERVER_ERROR,
        500,
        { details: response.error?.message || 'Failed to create tool policy' }
      );
    }

    console.log('[Protocol 10050] Tool policy created successfully:', response.data?.id);

    return response.data!;
  } catch (error) {
    console.error('[Protocol 10050] Error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
