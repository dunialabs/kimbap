/**
 * KIMBAP Core URL validation and caching utilities
 * Uses KIMBAP_CORE_URL to validate and cache the Core service connection.
 */

interface CacheEntry {
  url: string;
  status: 'valid' | 'invalid_format' | 'unreachable';
  timestamp: number;
  host?: string;
  port?: number;
  errorMessage?: string;
}

interface ValidationResult {
  isValid: boolean;
  status: 'valid' | 'invalid_format' | 'unreachable';
  host?: string;
  port?: number;
  errorMessage?: string;
}

// In-memory cache (cleared on service restart)
let cache: CacheEntry | null = null;

// Cache TTL constants
const CACHE_TTL = {
  PERMANENT: -1, // Never expires (until service restart)
  UNREACHABLE: 60 * 1000, // 1 minute for unreachable hosts
};

/**
 * Check if cached entry is still valid
 */
function isCacheValid(entry: CacheEntry): boolean {
  // Permanent cache (valid and invalid_format)
  if (entry.status === 'valid' || entry.status === 'invalid_format') {
    return true;
  }

  // Temporary cache (unreachable)
  if (entry.status === 'unreachable') {
    const elapsed = Date.now() - entry.timestamp;
    return elapsed < CACHE_TTL.UNREACHABLE;
  }

  return false;
}

/**
 * Validate KIMBAP_CORE_URL format
 * Returns validation result without making HTTP requests
 */
function validateFormat(url: string): { isValid: boolean; errorMessage?: string } {
  // Check if URL is empty
  if (!url || !url.trim()) {
    return {
      isValid: false,
      errorMessage: 'KIMBAP_CORE_URL is empty',
    };
  }

  // Check if URL ends with /admin (forbidden)
  if (url.trim().endsWith('/admin')) {
    return {
      isValid: false,
      errorMessage: 'KIMBAP_CORE_URL should not include /admin suffix (it will be added automatically)',
    };
  }

  // Validate URL format
  try {
    const urlObj = new URL(url);

    // Check protocol
    if (urlObj.protocol !== 'http:' && urlObj.protocol !== 'https:') {
      return {
        isValid: false,
        errorMessage: `Invalid protocol "${urlObj.protocol}" - must be http:// or https://`,
      };
    }

    // Check if hostname exists
    if (!urlObj.hostname) {
      return {
        isValid: false,
        errorMessage: 'KIMBAP_CORE_URL does not contain a valid hostname',
      };
    }

    return { isValid: true };
  } catch (error: any) {
    return {
      isValid: false,
      errorMessage: `Invalid URL format: ${error.message}`,
    };
  }
}

/**
 * Validate availability of KIMBAP_CORE_URL and verify it's Kimbap Core service
 * Makes actual HTTP request to check if service is responding
 */
async function validateAvailability(url: string): Promise<ValidationResult> {
  try {
    const baseUrl = url.trim().replace(/\/+$/, '');
    const healthUrl = `${baseUrl}/health`;

    // Create fetch with 3 second timeout
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 3000);

    const response = await fetch(healthUrl, {
      method: 'GET',
      signal: controller.signal,
      headers: {
        'Accept': 'application/json',
      },
    });

    clearTimeout(timeoutId);

    // Check if response is successful
    if (response.ok) {
      try {
        const data = await response.json();
        const status = typeof data?.status === 'string'
          ? data.status
          : (typeof data?.data?.status === 'string' ? data.data.status : '');

        if (status === 'healthy' || status === 'ok') {
          // Extract host and port from URL
          const urlObj = new URL(baseUrl);
          const host = urlObj.hostname;
          const port = urlObj.port ? parseInt(urlObj.port) : (urlObj.protocol === 'https:' ? 443 : 80);

          console.log(`✓ KIMBAP_CORE_URL validation successful: ${baseUrl}`);
          return {
            isValid: true,
            status: 'valid',
            host,
            port,
          };
        } else {
          console.warn(`✗ Service at ${healthUrl} is not recognized as Kimbap Core health endpoint (status: ${status || 'unknown'})`);
          return {
            isValid: false,
            status: 'unreachable',
            errorMessage: `Service at ${healthUrl} returned unrecognized health payload (status: ${status || 'unknown'})`,
          };
        }
      } catch (parseError) {
        // Response is not valid JSON
        console.warn(`✗ Invalid JSON response from ${healthUrl}`);
        return {
          isValid: false,
          status: 'unreachable',
          errorMessage: `Service at ${healthUrl} returned invalid JSON response`,
        };
      }
    } else {
      // HTTP error response
      console.warn(`✗ HTTP error from ${healthUrl}: ${response.status}`);
      return {
        isValid: false,
        status: 'unreachable',
        errorMessage: `Service at ${healthUrl} returned HTTP ${response.status}`,
      };
    }
  } catch (error: any) {
    // Connection error
    if (error.name === 'AbortError') {
      console.warn(`✗ Connection to ${url} timed out`);
      return {
        isValid: false,
        status: 'unreachable',
        errorMessage: `Connection to ${url} timed out (3s)`,
      };
    } else {
      console.warn(`✗ Failed to connect to ${url}:`, error.message);
      return {
        isValid: false,
        status: 'unreachable',
        errorMessage: `Cannot connect to ${url}: ${error.message}`,
      };
    }
  }
}

/**
 * Validate and cache KIMBAP_CORE_URL
 *
 * Performs comprehensive validation:
 * 1. Format validation (URL syntax, protocol, no /admin suffix)
 * 2. Availability check (HTTP request with 3s timeout)
 * 3. Service verification (health response status is "healthy" or "ok")
 *
 * Caching strategy:
 * - valid: Permanent cache (until service restart)
 * - invalid_format: Permanent cache (until service restart)
 * - unreachable: 1-minute cache, then retry
 *
 * @param url - KIMBAP_CORE_URL to validate
 * @returns Validation result with host and port if successful
 */
export async function validateAndCacheMcpGatewayUrl(url: string): Promise<ValidationResult> {
  // Check cache first
  if (cache && cache.url === url && isCacheValid(cache)) {
    console.log(`[MCP CACHE] Using cached validation result for ${url} (status: ${cache.status})`);
    return {
      isValid: cache.status === 'valid',
      status: cache.status,
      host: cache.host,
      port: cache.port,
      errorMessage: cache.errorMessage,
    };
  }

  console.log(`[MCP CACHE] Validating KIMBAP_CORE_URL: ${url}`);

  // Step 1: Format validation
  const formatCheck = validateFormat(url);
  if (!formatCheck.isValid) {
    console.warn(`[MCP CACHE] Format validation failed: ${formatCheck.errorMessage}`);

    // Cache format errors permanently
    cache = {
      url,
      status: 'invalid_format',
      timestamp: Date.now(),
      errorMessage: formatCheck.errorMessage,
    };

    return {
      isValid: false,
      status: 'invalid_format',
      errorMessage: formatCheck.errorMessage,
    };
  }

  // Step 2 & 3: Availability check + Service verification
  const availabilityCheck = await validateAvailability(url);

  // Cache the result
  cache = {
    url,
    status: availabilityCheck.status,
    timestamp: Date.now(),
    host: availabilityCheck.host,
    port: availabilityCheck.port,
    errorMessage: availabilityCheck.errorMessage,
  };

  return availabilityCheck;
}


