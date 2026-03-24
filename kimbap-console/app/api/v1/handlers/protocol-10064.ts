import { ApiError, ErrorCode } from '@/lib/error-codes';
import { purgeCacheTool } from '@/lib/proxy-api';

interface Request10064 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    serverId?: string;
    toolName?: string;
    reason?: string;
  };
}

interface Response10064Data {
  success: true;
}

export async function handleProtocol10064(body: Request10064): Promise<Response10064Data> {
  const userid = body.common?.userid;
  const { serverId, toolName, reason } = body.params || {};

  if (!serverId?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'serverId' });
  }

  if (!toolName?.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'toolName' });
  }

  await purgeCacheTool(serverId, toolName, reason, userid);
  return { success: true };
}
