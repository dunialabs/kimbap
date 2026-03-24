import { HardwareFingerprint } from './hardware-fingerprint';
import { LicenseCrypto, LicenseData } from './license-crypto';
import { FREE_TIER_CONFIG } from './free-tier-config';
import { prisma } from '@/lib/prisma';
import { getProxy } from '@/lib/proxy-api';

export interface ValidationResult {
  valid: boolean;
  isFreeTier: boolean;  // Whether it's free tier
  planLevel?: string;  // Current plan level (defined by official website)
  reason?: string;
  licenseData?: LicenseData;
  remainingDays?: number;
  toolCreationsRemaining?: number;
  accessTokensRemaining?: number;
}

export interface UsageStats {
  toolsCreated: number;
  accessTokensCreated: number;
  lastUpdated: number;
}

export class LicenseValidator {
  private static getMasterPassword(): string {
    const password = process.env.LICENSE_MASTER_PASSWORD;
    if (!password) {
      throw new Error(
        'LICENSE_MASTER_PASSWORD environment variable is not set. Please configure it in your .env file.'
      );
    }
    return password;
  }

  // Free tier default limits
  private static readonly FREE_TIER_LIMITS = {
    maxToolCreations: FREE_TIER_CONFIG.maxToolCreations,
    maxAccessTokens: FREE_TIER_CONFIG.maxAccessTokens
  };

  /**
   * Load license from database
   * @returns License string or null if not found
   */
  private static async loadLicenseFromDB(): Promise<string | null> {
    try {
      // First get the current proxy's key
      let proxyKey = '';
      try {
        const proxyInfo = await getProxy();
        proxyKey = proxyInfo.proxyKey || '';
      } catch (error) {
        console.error('Failed to get proxy key for license validation:', error);
        // Continue with empty proxyKey if fetch fails
      }

      // Find active license with matching proxyKey
      const license = await prisma.license.findFirst({
        where: { 
          status: 100, // Active status
          proxyKey: proxyKey // Must match current proxy's key
        },
        orderBy: { addtime: 'desc' } // Get the latest active license
      });
      
      if (!license && proxyKey) {
        console.log(`No active license found for proxyKey: ${proxyKey}`);
      }
      
      return license?.licenseStr || null;
    } catch (error) {
      console.error('Failed to load license from database:', error);
      return null;
    }
  }

  /**
   * Validate a license string without saving it
   * @param encryptedLicense The encrypted license string to validate
   * @returns Validation result
   */
  public static validateLicenseString(encryptedLicense: string, usageStats?: UsageStats): ValidationResult {
    // Validate a license string without saving it
    if (!encryptedLicense) {
      return {
        valid: false,
        isFreeTier: false,
        reason: 'License string is empty'
      };
    }

    // Decrypt the license
    let licenseData: LicenseData | null;
    try {
      licenseData = LicenseCrypto.decryptLicense(encryptedLicense, this.getMasterPassword());
    } catch (error) {
      return {
        valid: false,
        isFreeTier: false,
        reason: `Failed to decrypt license: ${error instanceof Error ? error.message : 'Unknown error'}`
      };
    }
    
    if (!licenseData) {
      return {
        valid: false,
        isFreeTier: false,
        reason: 'Invalid or corrupted license'
      };
    }

    // Check hardware fingerprint
    const currentFingerprint = HardwareFingerprint.generateFingerprint();
    if (licenseData.fingerprintHash !== currentFingerprint) {
      return {
        valid: false,
        isFreeTier: false,
        reason: 'License is not valid for this machine',
        licenseData
      };
    }

    // Check expiration
    const now = Date.now();
    if (now > licenseData.expiresAt) {
      return {
        valid: false,
        isFreeTier: false,
        reason: 'License has expired',
        licenseData
      };
    }

    const usageStats_ = usageStats || { toolsCreated: 0, accessTokensCreated: 0, lastUpdated: Date.now() };
    
    // Check usage limits
    if (usageStats_.toolsCreated >= licenseData.maxToolCreations) {
      return {
        valid: false,
        isFreeTier: false,
        reason: 'Tool creation limit reached',
        licenseData,
        toolCreationsRemaining: 0
      };
    }
    
    if (usageStats_.accessTokensCreated >= licenseData.maxAccessTokens) {
      return {
        valid: false,
        isFreeTier: false,
        reason: 'Access token limit reached',
        licenseData,
        accessTokensRemaining: 0
      };
    }

    const remainingDays = Math.floor((licenseData.expiresAt - now) / (1000 * 60 * 60 * 24));
    
    return {
      valid: true,
      isFreeTier: false,
      planLevel: licenseData.planLevel,
      licenseData,
      remainingDays,
      toolCreationsRemaining: licenseData.maxToolCreations - usageStats_.toolsCreated,
      accessTokensRemaining: licenseData.maxAccessTokens - usageStats_.accessTokensCreated
    };
  }

  /**
   * Validate current license from database
   * @returns Validation result
   */
  public static async validateLicense(): Promise<ValidationResult> {
    const encryptedLicense = await this.loadLicenseFromDB();
    const usageStats = await this.getUsageStats();
    
    // If no license in database, return free tier configuration
    if (!encryptedLicense) {
      return {
        valid: true,  // Free tier is also valid
        isFreeTier: true,
        planLevel: 'free',
        reason: 'Using free tier',
        toolCreationsRemaining: this.FREE_TIER_LIMITS.maxToolCreations - usageStats.toolsCreated,
        accessTokensRemaining: this.FREE_TIER_LIMITS.maxAccessTokens - usageStats.accessTokensCreated
      };
    }

    // Validate the license from database
    return this.validateLicenseString(encryptedLicense, usageStats);
  }

  /**
   * Get free tier limits
   * @returns Free tier limit configuration
   */
  public static getFreeTierLimits() {
    return { ...this.FREE_TIER_LIMITS };
  }

  /**
   * Get usage statistics (public method)
   * @returns Current usage statistics
   */
  public static async getUsageStats(): Promise<UsageStats> {
    try {
      const { countServers, countUsers } = await import('@/lib/proxy-api');
      
      const toolCountResult = await countServers({});
      const tokenCountResult = await countUsers({
        excludeRole: 1 // Exclude owner role
      });
      
      return {
        toolsCreated: toolCountResult.count,
        accessTokensCreated: tokenCountResult.count,
        lastUpdated: Date.now()
      };
    } catch (error) {
      console.error('Failed to get usage stats:', error);
      return {
        toolsCreated: 0,
        accessTokensCreated: 0,
        lastUpdated: Date.now()
      };
    }
}
}
