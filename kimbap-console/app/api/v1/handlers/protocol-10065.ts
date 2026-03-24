import { ApiError, ErrorCode } from '@/lib/error-codes';
import { purgeCachePrompt } from '@/lib/proxy-api';

interface Request10065 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId?: string;
    promptName?: string;
    reason?: string;
  };
}

interface Response10065Data {
  success: true;
}

export async function handleProtocol10065(body: Request10065): Promise<Response10065Data> {
  const userid = body.common?.userid;
  const { serverId, promptName, reason } = body.params || {};

  if (!serverId?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  if (!promptName?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'promptName' });
  }

  await purgeCachePrompt(serverId, promptName, reason, userid);
  return { success: true };
}
