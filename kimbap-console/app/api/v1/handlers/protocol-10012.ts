import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getIpWhitelist, makeProxyRequestWithUserId } from '@/lib/proxy-api';

interface Request10012 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: Record<string, never>; // Empty params according to proto
}

interface IPRecord {
  id: number;
  ip: string;
}

interface Response10012Data {
  ipList: IPRecord[];
}

/**
 * Protocol 10012 - Get IP Whitelist
 * Returns IP whitelist records from ip_whitelist table
 */
export async function handleProtocol10012(body: Request10012): Promise<Response10012Data> {
  try {
    // Fetch all IP whitelist records using proxy API
    const ipWhitelistResponse = await getIpWhitelist(body.common.userid);
    console.log('ipWhitelistResponse:', ipWhitelistResponse);

    // Transform response to protocol format
    const ipList: IPRecord[] = ipWhitelistResponse.whitelist.map((ip: string, index: number) => ({
      id: index + 1,  // Use index+1 as id since whitelist returns string array
      ip: ip
    }));

    return {
      ipList
    };
    
  } catch (error) {
    console.error('Protocol 10012 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}