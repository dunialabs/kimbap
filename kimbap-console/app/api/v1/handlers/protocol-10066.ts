import { ApiError, ErrorCode } from '@/lib/error-codes';
import { purgeCacheResource } from '@/lib/proxy-api';

interface Request10066 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId?: string;
    uri?: string;
    reason?: string;
  };
}

interface Response10066Data {
  success: true;
}

export async function handleProtocol10066(body: Request10066): Promise<Response10066Data> {
  const userid = body.common?.userid;
  const { serverId, uri, reason } = body.params || {};

  if (!serverId?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  if (!uri?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'uri' });
  }

  await purgeCacheResource(serverId, uri, reason, userid);
  return { success: true };
}
