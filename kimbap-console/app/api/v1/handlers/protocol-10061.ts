import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getCachePolicy, type CachePolicyResponse } from '@/lib/proxy-api';

interface Request10061 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId?: string;
  };
}

export async function handleProtocol10061(body: Request10061): Promise<CachePolicyResponse> {
  const userid = body.common?.userid;
  const { serverId } = body.params || {};

  if (!serverId?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  return getCachePolicy(serverId, userid);
}
