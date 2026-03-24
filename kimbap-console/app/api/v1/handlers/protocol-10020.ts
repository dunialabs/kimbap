import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { LicenseCrypto } from '@/license-system/license-crypto';
import { countServers, countUsers } from '@/lib/proxy-api';
import type { License } from '@/lib/types/license';




interface Response10020Data {
  licenseList: License[];
}

/**
 * Protocol 10020 - History License
 * Returns the history of all licenses (active, inactive, expired)
 * Decrypts all licenses to show full information
 */
export async function handleProtocol10020(): Promise<Response10020Data> {
  try {
    // Get all licenses from database, ordered by most recent first
    const licenses = await prisma.license.findMany({
      orderBy: { addtime: 'desc' }
    });

    // Get current usage counts only for active licenses
    let toolCount = 0;
    let tokenCount = 0;

    // Only calculate current usage if there's an active license
    const hasActiveLicense = licenses.some(license => license.status === 100);
    if (hasActiveLicense) {
      // Don't pass userid - use system-level access for counting
      // This avoids the "User not found in local database" warning when userid is invalid
      try {
        const toolCountResult = await countServers({});
        toolCount = toolCountResult.count;

        const tokenCountResult = await countUsers({
          excludeRole: 1 // Exclude owner role (1-owner, 2-admin, 3-member)
        });
        tokenCount = tokenCountResult.count;
      } catch (countError) {
        console.error('Failed to get usage counts (KIMBAP Core may be offline or not responding):', countError);
        // Continue with 0 counts if KIMBAP Core is not available
        // This allows license data to be displayed even if Core has issues
        toolCount = 0;
        tokenCount = 0;
      }
    }

    // Transform database records to License format
    const licenseList: License[] = [];

    // Get master password for decryption
    const masterPassword = process.env.LICENSE_MASTER_PASSWORD;
    if (!masterPassword) {
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        message: 'License master password not configured'
      });
    }

    for (const dbLicense of licenses) {
      if (dbLicense.status === 100) {
        // Active license - decrypt to show full information
        try {
          const decryptedLicense = LicenseCrypto.decryptLicense(dbLicense.licenseStr, masterPassword);

          if (decryptedLicense) {
            const license: License = {
              plan: decryptedLicense.planLevel || 'free',
              status: dbLicense.status,
              expiresAt: Math.floor(decryptedLicense.expiresAt / 1000), // Convert to seconds
              createdAt: dbLicense.addtime,
              customerEmail: decryptedLicense.customerEmail || '',
              licenseStr: dbLicense.licenseStr,
              maxToolCreations: decryptedLicense.maxToolCreations,
              maxAccessTokens: decryptedLicense.maxAccessTokens,
              currentToolCreations: toolCount,
              currentAccessTokens: tokenCount
            };

            licenseList.push(license);
          } else {
            // Failed to decrypt - return minimal info
            const license: License = {
              plan: 'active',
              status: dbLicense.status,
              expiresAt: 0,
              createdAt: dbLicense.addtime,
              customerEmail: '',
              licenseStr: dbLicense.licenseStr,
              maxToolCreations: 0,
              maxAccessTokens: 0,
              currentToolCreations: toolCount,
              currentAccessTokens: tokenCount
            };

            licenseList.push(license);
          }
        } catch (error) {
          console.error('Failed to decrypt active license:', error);
          // Return minimal info on decryption error
          const license: License = {
            plan: 'active',
            status: dbLicense.status,
            expiresAt: 0,
            createdAt: dbLicense.addtime,
            customerEmail: '',
            licenseStr: dbLicense.licenseStr,
            maxToolCreations: 0,
            maxAccessTokens: 0,
            currentToolCreations: toolCount,
            currentAccessTokens: tokenCount
          };

          licenseList.push(license);
        }
      } else {
        // Inactive or expired license - try to decrypt for history display
        try {
          const decryptedLicense = LicenseCrypto.decryptLicense(dbLicense.licenseStr, masterPassword);

          if (decryptedLicense) {
            const license: License = {
              plan: decryptedLicense.planLevel || '',
              status: dbLicense.status,
              expiresAt: Math.floor(decryptedLicense.expiresAt / 1000), // Convert to seconds
              createdAt: dbLicense.addtime,
              customerEmail: decryptedLicense.customerEmail || '',
              licenseStr: dbLicense.licenseStr,
              maxToolCreations: decryptedLicense.maxToolCreations,
              maxAccessTokens: decryptedLicense.maxAccessTokens,
              currentToolCreations: 0, // Don't show current counts for inactive licenses
              currentAccessTokens: 0
            };

            licenseList.push(license);
          } else {
            // Failed to decrypt - return minimal info
            const license: License = {
              plan: '',
              status: dbLicense.status,
              expiresAt: 0,
              createdAt: dbLicense.addtime,
              customerEmail: '',
              licenseStr: dbLicense.licenseStr,
              maxToolCreations: 0,
              maxAccessTokens: 0,
              currentToolCreations: 0,
              currentAccessTokens: 0
            };

            licenseList.push(license);
          }
        } catch (error) {
          console.error('Failed to decrypt inactive/expired license:', error);
          // Return minimal info on decryption error
          const license: License = {
            plan: '',
            status: dbLicense.status,
            expiresAt: 0,
            createdAt: dbLicense.addtime,
            customerEmail: '',
            licenseStr: dbLicense.licenseStr,
            maxToolCreations: 0,
            maxAccessTokens: 0,
            currentToolCreations: 0,
            currentAccessTokens: 0
          };

          licenseList.push(license);
        }
      }
    }

    return {
      licenseList
    };

  } catch (error) {
    console.error('Protocol 10020 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}