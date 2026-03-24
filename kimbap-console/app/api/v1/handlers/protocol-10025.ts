import { ApiError, ErrorCode } from '@/lib/error-codes';
import { LicenseService } from '@/license-system';
import { countServers, countUsers, getProxy } from '@/lib/proxy-api';

interface Request10025 {
  common: {
    cmdId: number;
    userid?: number;
  };
  params: {
    proxyKey?: string;
  };
}

interface Response10025Data {
  isFreeTier: boolean;
  expiresAt: number;
  maxToolCreations: number;
  currentToolCount: number;
  remainingToolCount: number;
  maxAccessTokens: number;
  currentAccessTokenCount: number;
  remainingAccessTokenCount: number;
  licenseKey: string;
  fingerprintHash: string;
}

export async function handleProtocol10025(body: Request10025): Promise<Response10025Data> {
  try {
    // Get proxy information
    const proxy = await getProxy();
    if (!proxy) {
      throw new ApiError(ErrorCode.RECORD_NOT_FOUND, 404);
    }

    const requestProxyKey = body.params?.proxyKey?.trim();
    if (requestProxyKey && requestProxyKey !== proxy.proxyKey) {
      throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { message: 'Invalid proxyKey' });
    }

    const proxyId = proxy.id;

    // Get current tool (server) count
    const toolCountResult = await countServers({
      proxyId: proxyId
    }, body.common.userid?.toString());
    const currentToolCount = toolCountResult.count;

    // Get current access token count (exclude owners - role 1)
    const tokenCountResult = await countUsers({
      excludeRole: 1 // Exclude owner role (1-owner, 2-admin, 3-member)
    }, body.common.userid?.toString());
    const currentAccessTokenCount = tokenCountResult.count;

    // Get license service and check limits
    const licenseService = LicenseService.getInstance();
    const licenseInfo = await licenseService.validateLicense();

    // Check tool creation limit
    const toolLimitCheck = await licenseService.checkToolCreationLimit(currentToolCount);

    // Check access token limit
    const tokenLimitCheck = await licenseService.checkAccessTokenLimit(currentAccessTokenCount);

    // Return combined data
    return {
      isFreeTier: toolLimitCheck.isFreeTier, // Both should be the same
      expiresAt: licenseInfo.isFreeTier ? 0 : (licenseInfo.expiresAt || 0),
      maxToolCreations: toolLimitCheck.maxToolCreations,
      currentToolCount: currentToolCount,
      remainingToolCount: toolLimitCheck.remaining,
      maxAccessTokens: tokenLimitCheck.maxAccessTokens,
      currentAccessTokenCount: currentAccessTokenCount,
      remainingAccessTokenCount: tokenLimitCheck.remaining,
      licenseKey: licenseInfo.isFreeTier ? '' : (licenseInfo.licenseKey || ''),
      fingerprintHash: licenseInfo.isFreeTier ? '' : (licenseInfo.fingerprintHash || '')
    };
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('Failed to get operate limit:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
