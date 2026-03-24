import { createCipheriv, createDecipheriv, randomBytes, createHash, pbkdf2Sync } from 'crypto';

export interface LicenseData {
  fingerprintHash: string;
  createdAt: number;
  expiresAt: number;
  licenseKey: string;
  maxToolCreations: number;
  maxAccessTokens: number;
  customerEmail?: string;
  planLevel?: string;  // Plan level identifier (defined by official website, e.g., lv1, lv2, etc.)
}

export class LicenseCrypto {
  private static readonly ALGORITHM = 'aes-256-gcm';
  private static readonly SALT_LENGTH = 32;
  private static readonly IV_LENGTH = 16;
  private static readonly TAG_LENGTH = 16;
  private static readonly ITERATIONS = 100000;
  private static readonly KEY_LENGTH = 32;

  private static deriveKey(password: string, salt: Buffer): Buffer {
    return pbkdf2Sync(password, salt, this.ITERATIONS, this.KEY_LENGTH, 'sha256');
  }

  public static encryptLicense(
    licenseData: LicenseData,
    masterPassword: string
  ): string {
    const salt = randomBytes(this.SALT_LENGTH);
    const iv = randomBytes(this.IV_LENGTH);
    const key = this.deriveKey(masterPassword, salt);

    const cipher = createCipheriv(this.ALGORITHM, key, iv);
    
    const jsonData = JSON.stringify(licenseData);
    const encrypted = Buffer.concat([
      cipher.update(jsonData, 'utf8'),
      cipher.final()
    ]);
    
    const authTag = cipher.getAuthTag();
    
    const combined = Buffer.concat([
      salt,
      iv,
      authTag,
      encrypted
    ]);
    
    return combined.toString('base64');
  }

  public static decryptLicense(
    encryptedLicense: string,
    masterPassword: string
  ): LicenseData | null {
    try {
      const combined = Buffer.from(encryptedLicense, 'base64');
      
      const salt = combined.subarray(0, this.SALT_LENGTH);
      const iv = combined.subarray(this.SALT_LENGTH, this.SALT_LENGTH + this.IV_LENGTH);
      const authTag = combined.subarray(
        this.SALT_LENGTH + this.IV_LENGTH,
        this.SALT_LENGTH + this.IV_LENGTH + this.TAG_LENGTH
      );
      const encrypted = combined.subarray(this.SALT_LENGTH + this.IV_LENGTH + this.TAG_LENGTH);
      
      const key = this.deriveKey(masterPassword, salt);
      
      const decipher = createDecipheriv(this.ALGORITHM, key, iv);
      decipher.setAuthTag(authTag);
      
      const decrypted = Buffer.concat([
        decipher.update(encrypted),
        decipher.final()
      ]);
      
      const licenseData: LicenseData = JSON.parse(decrypted.toString('utf8'));
      
      return licenseData;
    } catch (error) {
      console.error('Failed to decrypt license:', error);
      return null;
    }
  }

  public static generateLicenseKey(
    fingerprintHash: string,
    customerEmail: string,
    planType: string
  ): string {
    const data = `${fingerprintHash}:${customerEmail}:${planType}:${Date.now()}`;
    const hash = createHash('sha256').update(data).digest('hex');
    const key = hash.substring(0, 32).toUpperCase();
    
    const formatted = `${key.substring(0, 8)}-${key.substring(8, 16)}-${key.substring(16, 24)}-${key.substring(24, 32)}`;
    return formatted;
  }

  public static verifyLicenseKey(
    licenseKey: string,
    fingerprintHash: string
  ): boolean {
    const cleanKey = licenseKey.replace(/-/g, '');
    
    return cleanKey.length === 32 && /^[A-F0-9]+$/.test(cleanKey);
  }
}