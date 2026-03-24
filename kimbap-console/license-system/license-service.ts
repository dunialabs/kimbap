import { LicenseValidator } from './license-validator';

export interface LicenseInfo {
  valid: boolean;
  isFreeTier: boolean;
  planLevel?: string;
  expiresAt?: number;
  remainingDays?: number;
  licenseKey?: string;
  fingerprintHash?: string;
  limits?: {
    maxToolCreations: number;
    maxAccessTokens: number;
    toolCreationsRemaining: number;
    accessTokensRemaining: number;
  };
}

/**
 * License service for managing software licenses
 * Singleton pattern to ensure consistent license validation across the application
 */
export class LicenseService {
  private static instance: LicenseService;

  private constructor() {
    // Private constructor to enforce singleton pattern
  }

  /**
   * Get singleton instance
   * @returns LicenseService instance
   */
  public static getInstance(): LicenseService {
    if (!LicenseService.instance) {
      LicenseService.instance = new LicenseService();
    }
    return LicenseService.instance;
  }

  /**
   * Get hardware fingerprint
   * @returns Hardware fingerprint string
   */
  public getHardwareFingerprint(): string {
    return require('./hardware-fingerprint').HardwareFingerprint.generateFingerprint();
  }

  /**
   * Get hardware information
   * @returns Hardware information object
   */
  public getHardwareInfo() {
    return require('./hardware-fingerprint').HardwareFingerprint.getHardwareInfo();
  }

  /**
   * Validate current license status
   * @returns License information
   */
  public async validateLicense(): Promise<LicenseInfo> {
    try {
      const validation = await LicenseValidator.validateLicense();
      
      // If free tier
      if (validation.isFreeTier) {
        const freeLimits = LicenseValidator.getFreeTierLimits();
        return {
          valid: true,
          isFreeTier: true,
          planLevel: 'free',
          limits: {
            maxToolCreations: freeLimits.maxToolCreations,
            maxAccessTokens: freeLimits.maxAccessTokens,
            toolCreationsRemaining: validation.toolCreationsRemaining || 0,
            accessTokensRemaining: validation.accessTokensRemaining || 0
          }
        };
      }

      // Has license (valid or invalid)
      if (validation.valid && validation.licenseData) {
        return {
          valid: true,
          isFreeTier: false,
          planLevel: validation.planLevel,
          expiresAt: validation.licenseData.expiresAt,
          remainingDays: validation.remainingDays,
          licenseKey: validation.licenseData.licenseKey,
          fingerprintHash: validation.licenseData.fingerprintHash,
          limits: {
            maxToolCreations: validation.licenseData.maxToolCreations,
            maxAccessTokens: validation.licenseData.maxAccessTokens,
            toolCreationsRemaining: validation.toolCreationsRemaining || 0,
            accessTokensRemaining: validation.accessTokensRemaining || 0
          }
        };
      } else {
        // Invalid license - fall back to free tier
        const freeLimits = LicenseValidator.getFreeTierLimits();
        return {
          valid: true, // Still valid as free tier
          isFreeTier: true,
          planLevel: 'free',
          limits: {
            maxToolCreations: freeLimits.maxToolCreations,
            maxAccessTokens: freeLimits.maxAccessTokens,
            toolCreationsRemaining: freeLimits.maxToolCreations,
            accessTokensRemaining: freeLimits.maxAccessTokens
          }
        };
      }
    } catch (error) {
      console.error('License validation error:', error);
      // Return free tier on error
      const freeLimits = LicenseValidator.getFreeTierLimits();
      return {
        valid: true,
        isFreeTier: true,
        planLevel: 'free',
        limits: {
          maxToolCreations: freeLimits.maxToolCreations,
          maxAccessTokens: freeLimits.maxAccessTokens,
          toolCreationsRemaining: freeLimits.maxToolCreations,
          accessTokensRemaining: freeLimits.maxAccessTokens
        }
      };
    }
  }

  /**
   * Get free tier limits
   * @returns Free tier limit information
   */
  public getFreeTierLimits(): {
    maxToolCreations: number;
    maxAccessTokens: number;
  } {
    return LicenseValidator.getFreeTierLimits();
  }

  /**
   * Check if tool creation is allowed based on license limits
   * @param currentToolCount Current number of tools in database
   * @returns Object with allowed flag and limit info
   */
  public async checkToolCreationLimit(currentToolCount: number): Promise<{
    allowed: boolean;
    isFreeTier: boolean;
    maxToolCreations: number;
    currentCount: number;
    remaining: number;
  }> {
    const licenseInfo = await this.validateLicense();
    
    if (licenseInfo.valid && !licenseInfo.isFreeTier) {
      // Has valid license - check license limits
      const maxTools = licenseInfo.limits?.maxToolCreations || 0;
      const remaining = Math.max(0, maxTools - currentToolCount);
      
      return {
        allowed: currentToolCount < maxTools,
        isFreeTier: false,
        maxToolCreations: maxTools,
        currentCount: currentToolCount,
        remaining: remaining
      };
    } else {
      // No valid license - use free tier limits
      const freeLimits = this.getFreeTierLimits();
      const maxTools = freeLimits.maxToolCreations;
      const remaining = Math.max(0, maxTools - currentToolCount);
      
      return {
        allowed: currentToolCount < maxTools,
        isFreeTier: true,
        maxToolCreations: maxTools,
        currentCount: currentToolCount,
        remaining: remaining
      };
    }
  }

  /**
   * Check if access token creation is allowed based on license limits
   * @param currentTokenCount Current number of access tokens in database
   * @returns Object with allowed flag and limit info
   */
  public async checkAccessTokenLimit(currentTokenCount: number): Promise<{
    allowed: boolean;
    isFreeTier: boolean;
    maxAccessTokens: number;
    currentCount: number;
    remaining: number;
  }> {
    const licenseInfo = await this.validateLicense();
    
    if (licenseInfo.valid && !licenseInfo.isFreeTier) {
      // Has valid license - check license limits
      const maxTokens = licenseInfo.limits?.maxAccessTokens || 0;
      const remaining = Math.max(0, maxTokens - currentTokenCount);
      
      return {
        allowed: currentTokenCount < maxTokens,
        isFreeTier: false,
        maxAccessTokens: maxTokens,
        currentCount: currentTokenCount,
        remaining: remaining
      };
    } else {
      // No valid license - use free tier limits
      const freeLimits = this.getFreeTierLimits();
      const maxTokens = freeLimits.maxAccessTokens;
      const remaining = Math.max(0, maxTokens - currentTokenCount);
      
      return {
        allowed: currentTokenCount < maxTokens,
        isFreeTier: true,
        maxAccessTokens: maxTokens,
        currentCount: currentTokenCount,
        remaining: remaining
      };
    }
  }
}
