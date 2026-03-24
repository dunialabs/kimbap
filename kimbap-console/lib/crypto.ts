/**
 * Crypto utility functions for KIMBAP Console
 * Implements AES-GCM encryption with PBKDF2 key derivation
 */

export interface EncryptedData {
  data: string; // Base64 encoded encrypted data
  iv: string; // Base64 encoded initialization vector
  salt: string; // Base64 encoded salt
  tag: string; // Base64 encoded authentication tag
}

export interface MasterPasswordData {
  hash: string; // Password hash using random salt
  salt: string; // Base64 encoded random salt
}

/**
 * Crypto utility class for handling all encryption/decryption operations
 */
export class CryptoUtils {

  private static readonly PBKDF2_ITERATIONS = 100000;
  private static readonly KEY_LENGTH = 256; // 256 bits for AES-256
  private static readonly IV_LENGTH = 96; // 96 bits for GCM
  private static readonly SALT_LENGTH = 128; // 128 bits for random salts

  /**
   * Get crypto instance (works in both browser and Node.js)
   */
  private static getCrypto(): Crypto {
    if (typeof window !== 'undefined') {
      // Browser environment
      if (!window.crypto || !window.crypto.subtle) {
        throw new Error(
          'Web Crypto API is not available. This application requires HTTPS or localhost. ' +
          'Please access via HTTPS (https://...) or localhost (http://localhost:...). ' +
          'For more information, visit: https://docs.kimbap.io/#caddy-https-config'
        );
      }
      return window.crypto;
    }
    // Node.js environment - import crypto dynamically
    if (typeof globalThis !== 'undefined' && (globalThis as any).crypto) {
      return (globalThis as any).crypto;
    }
    throw new Error('Crypto API not available');
  }

  /**
   * Generate a random salt
   */
  static generateSalt(): Uint8Array {
    return this.getCrypto().getRandomValues(new Uint8Array(this.SALT_LENGTH / 8));
  }

  /**
   * Generate a random initialization vector
   */
  static generateIV(): Uint8Array {
    return this.getCrypto().getRandomValues(new Uint8Array(this.IV_LENGTH / 8));
  }

  /**
   * Generate a secure random token
   */
  static generateToken(length: number = 64): string {
    const randomBytes = this.getCrypto().getRandomValues(new Uint8Array(length));
    const bytes: number[] = [];
    for (let i = 0; i < randomBytes.length; i++) {
      bytes.push(randomBytes[i]);
    }
    return bytes.map((b) => b.toString(16).padStart(2, "0")).join("");
  }

  /**
   * Calculate a unique user ID from token using SHA-256 hash
   * @param token - The token to generate user ID from
   * @returns A deterministic 32-character hex string user ID
   */
  static async calculateUserId(token: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(token);

    // Use SHA-256 instead of MD5 (more secure)
    const hashBuffer = await this.getCrypto().subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));

    // Take first 32 characters to simulate MD5 length, or adjust as needed
    return hashArray
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("")
      .substring(0, 32);
  }

  /**
   * Derive key from password using PBKDF2
   * @param password - The password to derive key from
   * @param salt - Salt for key derivation
   * @param iterations - Number of PBKDF2 iterations
   * @param extractable - Whether the key can be exported using exportKey().
   *                      Set to true only when you need to export the key (e.g., for hashing).
   *                      Defaults to false for better security.
   */
  static async deriveKey(
    password: string,
    salt: Uint8Array,
    iterations: number = this.PBKDF2_ITERATIONS,
    extractable: boolean = false,
  ): Promise<CryptoKey> {
    const encoder = new TextEncoder();
    const passwordBuffer = encoder.encode(password);

    const baseKey = await this.getCrypto().subtle.importKey(
      "raw",
      passwordBuffer,
      "PBKDF2",
      false,
      ["deriveKey"],
    );

    return this.getCrypto().subtle.deriveKey(
      {
        name: "PBKDF2",
        salt: salt,
        iterations: iterations,
        hash: "SHA-256",
      },
      baseKey,
      {
        name: "AES-GCM",
        length: this.KEY_LENGTH,
      },
      extractable,
      ["encrypt", "decrypt"],
    );
  }

  /**
   * Encrypt data using AES-GCM
   */
  static async encrypt(
    data: string,
    key: CryptoKey,
    salt?: Uint8Array,
  ): Promise<EncryptedData> {
    const encoder = new TextEncoder();
    const dataBuffer = encoder.encode(data);

    const iv = this.generateIV();
    const usedSalt = salt || this.generateSalt();

    const encryptedBuffer = await this.getCrypto().subtle.encrypt(
      {
        name: "AES-GCM",
        iv: iv,
      },
      key,
      dataBuffer,
    );

    // Extract the encrypted data and authentication tag
    const encryptedArray = new Uint8Array(encryptedBuffer);
    const encryptedData = encryptedArray.slice(0, -16); // All but last 16 bytes
    const tag = encryptedArray.slice(-16); // Last 16 bytes are the tag

    return {
      data: btoa(String.fromCharCode.apply(null, Array.from(encryptedData))),
      iv: btoa(String.fromCharCode.apply(null, Array.from(iv))),
      salt: btoa(String.fromCharCode.apply(null, Array.from(usedSalt))),
      tag: btoa(String.fromCharCode.apply(null, Array.from(tag))),
    };
  }

  /**
   * Decrypt data using AES-GCM
   */
  static async decrypt(
    encryptedData: EncryptedData,
    key: CryptoKey,
  ): Promise<string> {
    const data = new Uint8Array(
      atob(encryptedData.data)
        .split("")
        .map((c) => c.charCodeAt(0)),
    );
    const iv = new Uint8Array(
      atob(encryptedData.iv)
        .split("")
        .map((c) => c.charCodeAt(0)),
    );
    const tag = new Uint8Array(
      atob(encryptedData.tag)
        .split("")
        .map((c) => c.charCodeAt(0)),
    );

    // Combine encrypted data and tag for decryption
    const combinedBuffer = new Uint8Array(data.length + tag.length);
    combinedBuffer.set(data);
    combinedBuffer.set(tag, data.length);

    const decryptedBuffer = await this.getCrypto().subtle.decrypt(
      {
        name: "AES-GCM",
        iv: iv,
      },
      key,
      combinedBuffer,
    );

    const decoder = new TextDecoder();
    return decoder.decode(decryptedBuffer);
  }

  /**
   * Encrypt data using key
   */
  static async encryptData(
    originalData: string,
    key: string,
  ): Promise<EncryptedData> {
    const salt = CryptoUtils.generateSalt();
    const tokenKey = await CryptoUtils.deriveKey(key, salt);
    return CryptoUtils.encrypt(originalData, tokenKey, salt);
  }

  /**
   * Decrypt data using key
   */
  static async decryptData(
    encryptedData: EncryptedData,
    key: string,
  ): Promise<string> {
    const salt = new Uint8Array(
      atob(encryptedData.salt)
        .split("")
        .map((c) => c.charCodeAt(0)),
    );
    const tokenKey = await this.deriveKey(key, salt);

    return CryptoUtils.decrypt(encryptedData, tokenKey);
  }

  /**
   * Decrypt data from string using key
   * @param encryptedDataString - JSON stringified EncryptedData object
   * @param key - The key to decrypt with
   * @returns Decrypted string data
   */
  static async decryptDataFromString(
    encryptedDataString: string,
    key: string,
  ): Promise<string> {
    try {
      // Validate input parameters
      if (!encryptedDataString || typeof encryptedDataString !== 'string') {
        throw new Error('Invalid encrypted data string: must be a non-empty string');
      }

      if (!key || typeof key !== 'string') {
        throw new Error('Invalid decryption key: must be a non-empty string');
      }

      // Parse the JSON string back to EncryptedData object
      let encryptedData: EncryptedData;
      try {
        encryptedData = JSON.parse(encryptedDataString);
      } catch (parseError) {
        throw new Error('Invalid JSON format in encrypted data string');
      }

      // Validate the parsed object has required fields
      if (!encryptedData || typeof encryptedData !== 'object') {
        throw new Error('Parsed data must be an object');
      }

      if (!encryptedData.data || !encryptedData.iv || !encryptedData.salt || !encryptedData.tag) {
        throw new Error('Missing required fields in encrypted data object');
      }

      // Use the existing decryptData method
      return await this.decryptData(encryptedData, key);

    } catch (error) {
      // Handle specific error types
      if (error instanceof Error) {
        // Re-throw our custom validation errors
        if (error.message.startsWith('Invalid') || 
            error.message.startsWith('Missing') || 
            error.message.startsWith('Parsed')) {
          throw error;
        }
        
        // Handle base64 decoding errors from decryptData
        if (error.name === 'InvalidCharacterError') {
          throw new Error('Invalid base64 encoding in encrypted data');
        }
        
        // Handle cryptographic errors from decrypt operation
        if (error.name === 'OperationError' || 
            error.message.includes('decrypt')) {
          throw new Error('Decryption failed: invalid key or corrupted data');
        }
        
        // Handle key derivation errors
        if (error.message.includes('deriveKey') || 
            error.message.includes('PBKDF2')) {
          throw new Error('Key derivation failed: invalid password or salt');
        }
      }
      
      // Generic error for unexpected cases
      throw new Error('Failed to decrypt data from string: ' + (error instanceof Error ? error.message : 'unknown error'));
    }
  }

  /**
   * Hash password for verification using custom salt (PBKDF2)
   */
  static async hashPasswordWithSalt(password: string, salt: Uint8Array): Promise<string> {
    const key = await this.deriveKey(
      password,
      salt,
      this.PBKDF2_ITERATIONS,
      true,
    ); // Make extractable for hashing
    const keyBuffer = await this.getCrypto().subtle.exportKey("raw", key);
    const keyArray = new Uint8Array(keyBuffer);
    return btoa(String.fromCharCode.apply(null, Array.from(keyArray)));
  }


  /**
   * Verify password against stored hash and salt
   */
  static async verifyPasswordWithSalt(
    password: string,
    storedHash: string,
    salt: Uint8Array,
  ): Promise<boolean> {
    const computedHash = await this.hashPasswordWithSalt(password, salt);
    return computedHash === storedHash;
  }
}

/**
 * Master password management with random salt for enhanced security
 */
export class MasterPasswordManager {
  private static readonly STORAGE_KEY = "kimbap_master_data";
  private static readonly CACHE_KEY = "kimbap_master_cache";
  private static readonly CACHE_DURATION = 10 * 60 * 1000; // 10 minutes in milliseconds

  /**
   * Check if master password is set
   */
  static hasMasterPassword(): boolean {
    return localStorage.getItem(this.STORAGE_KEY) !== null;
  }

  /**
   * Set master password (first time setup)
   */
  static async setMasterPassword(password: string): Promise<void> {
    // Generate random salt for better security
    const salt = CryptoUtils.generateSalt();

    // Hash password with random salt
    const hash = await CryptoUtils.hashPasswordWithSalt(password, salt);

    // Store hash and salt
    const masterData: MasterPasswordData = {
      hash: hash,
      salt: btoa(String.fromCharCode.apply(null, Array.from(salt)))
    };

    localStorage.setItem(this.STORAGE_KEY, JSON.stringify(masterData));

    // Cache the password for 10 minutes
    this.cacheMasterPassword(password);
  }

  /**
   * Verify master password
   */
  static async verifyMasterPassword(password: string): Promise<boolean> {
    const storedDataStr = localStorage.getItem(this.STORAGE_KEY);
    if (!storedDataStr) return false;

    try {
      const storedData: MasterPasswordData = JSON.parse(storedDataStr);

      // Decode salt from base64
      const salt = new Uint8Array(
        atob(storedData.salt).split('').map(c => c.charCodeAt(0))
      );

      // Verify password with stored salt
      const isValid = await CryptoUtils.verifyPasswordWithSalt(password, storedData.hash, salt);

      // If valid, cache the password
      if (isValid) {
        this.cacheMasterPassword(password);
      }

      return isValid;
    } catch (error) {
      console.error('Failed to parse master password data:', error);
      return false;
    }
  }

  /**
   * Cache master password for 10 minutes
   */
  private static cacheMasterPassword(password: string): void {
    const cacheData = {
      password: btoa(password), // Simple encoding (not for security, just obfuscation)
      timestamp: Date.now()
    };
    sessionStorage.setItem(this.CACHE_KEY, JSON.stringify(cacheData));
  }

  /**
   * Get cached master password if still valid (within 10 minutes)
   */
  static getCachedMasterPassword(): string | null {
    const cacheDataStr = sessionStorage.getItem(this.CACHE_KEY);
    if (!cacheDataStr) return null;

    try {
      const cacheData = JSON.parse(cacheDataStr);
      const now = Date.now();
      const elapsed = now - cacheData.timestamp;

      // Check if cache is still valid (within 10 minutes)
      if (elapsed < this.CACHE_DURATION) {
        return atob(cacheData.password);
      } else {
        // Cache expired, remove it
        this.clearCache();
        return null;
      }
    } catch (error) {
      console.error('Failed to parse cached master password:', error);
      return null;
    }
  }

  /**
   * Check if cached master password is available
   */
  static hasCachedMasterPassword(): boolean {
    return this.getCachedMasterPassword() !== null;
  }

  /**
   * Clear cached master password
   */
  static clearCache(): void {
    sessionStorage.removeItem(this.CACHE_KEY);
  }

  /**
   * Clear master password (for logout or reset)
   */
  static clearMasterPassword(): void {
    localStorage.removeItem(this.STORAGE_KEY);
    this.clearCache();
  }
}
