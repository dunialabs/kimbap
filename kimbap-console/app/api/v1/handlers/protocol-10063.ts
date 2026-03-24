import { ApiError, ErrorCode } from '@/lib/error-codes';
import { purgeCacheServer } from '@/lib/proxy-api';

interface Request10063 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId?: string;
    reason?: string;
  };
}

interface Response10063Data {
  success: true;
}

export async function handleProtocol10063(body: Request10063): Promise<Response10063Data> {
  const userid = body.common?.userid;
  const { serverId, reason } = body.params || {};

  if (!serverId?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  await purgeCacheServer(serverId, reason || 'kimbap-console request', userid);
  return { success: true };
}
