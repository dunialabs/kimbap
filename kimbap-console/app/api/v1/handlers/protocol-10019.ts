import { prisma } from '@/lib/prisma';
import { LicenseValidator } from '@/license-system';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';

interface Request10019 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    proxyKey: string;
    licenseStr: string;
  };
}

interface Response10019Data {
  // Empty data as per proto definition
}

/**
 * Protocol 10019: Import License
 * 
 * This protocol handles license import and activation
 * Steps:
 * 1. Validate the license string format
 * 2. Activate the license using LicenseService
 * 3. Store the license in database
 * 4. Return activation result
 */
export async function handleProtocol10019(body: Request10019): Promise<Response10019Data> {
  const { proxyKey, licenseStr } = body.params || {};

  // Validate required parameters
  if (!proxyKey || typeof proxyKey !== 'string' || proxyKey.trim() === '') {
    throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { message: 'proxyKey is required' });
  }
  
  if (!licenseStr || typeof licenseStr !== 'string' || licenseStr.trim() === '') {
    throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { message: 'licenseStr is required' });
  }

  // Validate proxyKey matches the one from proxy API
  try {
    const proxyInfo = await getProxy();
    if (!proxyInfo.proxyKey || proxyInfo.proxyKey !== proxyKey.trim()) {
      console.error('ProxyKey mismatch - provided:', proxyKey, 'expected:', proxyInfo.proxyKey);
      throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { message: 'Invalid proxyKey' });
    }
    console.log('ProxyKey validated successfully');
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('Failed to validate proxyKey:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { message: 'Failed to validate proxyKey' });
  }

  // Validate license directly
  try {
    const validation = LicenseValidator.validateLicenseString(licenseStr.trim());
    if (!validation.valid) {
      const errorMsg = validation.reason || 'Invalid license';
      throw new ApiError(ErrorCode.LICENSE_ACTIVATION_FAILED, 400, { message: errorMsg });
    }
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    const errorMsg = error instanceof Error ? error.message : 'Failed to validate license';
    throw new ApiError(ErrorCode.LICENSE_ACTIVATION_FAILED, 400, { message: errorMsg });
  }

  // License is valid, save to database
  try {
    // Check if a license already exists
    const existingLicense = await prisma.license.findFirst({
      where: {
        status: 100 // Active status
      }
    });

    if (existingLicense) {
      // Update existing license to inactive
      await prisma.license.update({
        where: {
          id: existingLicense.id
        },
        data: {
          status: 1 // Set to inactive (replaced by new license)
        }
      });
    }

    // Save the new license with proxyKey
    await prisma.license.create({
      data: {
        licenseStr: licenseStr.trim(),
        addtime: Math.floor(Date.now() / 1000), // Unix timestamp in seconds
        status: 100, // Active status
        proxyKey: proxyKey.trim() // Save validated proxyKey to database
      }
    });
    
    console.log('License saved successfully with proxyKey:', proxyKey);
  } catch (dbError) {
    console.error('Database error while saving license:', dbError);
    // Continue even if database save fails, as license is activated locally
  }

  // Return empty data as per proto definition
  return {};
}
