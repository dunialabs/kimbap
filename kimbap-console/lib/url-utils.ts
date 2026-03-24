/**
 * URL normalization utilities
 * Minimal normalization for database storage (preserve user configuration)
 */

/**
 * Normalize host for database storage
 * Only performs minimal necessary normalization:
 * - Converts https to http for Docker internal services (kimbap-core)
 * - Does NOT replace host.docker.internal (different from kimbap-core!)
 * - Does NOT modify IP addresses or domain names
 *
 * Important distinction:
 * - host.docker.internal: Console Docker → Core on host machine
 * - kimbap-core: Console Docker → Core Docker (same network)
 * - localhost: Console local → Core local
 * - IP/domain: Cross-server deployment
 *
 * @param host - Host string (may include protocol)
 * @returns Minimally normalized host string
 */
export function normalizeHostForDatabase(host: string): string {
  if (!host) return host;

  // Only normalize protocol for Docker service name (kimbap-core)
  // host.docker.internal should keep its protocol as-is
  const hostWithoutProtocol = host.replace(/^https?:\/\//, '').replace(/\/.*$/, '');

  if (hostWithoutProtocol === 'kimbap-core') {
    // Docker service name should use HTTP (not HTTPS)
    if (host.startsWith('https://')) {
      const normalized = host.replace('https://', 'http://');
      console.log('[URL UTILS] Normalized kimbap-core: https → http for Docker service');
      return normalized;
    }
  }

  // For all other cases (host.docker.internal, localhost, IP, domain):
  // Keep as-is, do NOT modify
  return host;
}

/**
 * Validate if host configuration is appropriate for current environment
 * Prevents invalid configurations like using localhost in Docker or kimbap-core in local env
 *
 * @param host - Host string (without protocol)
 * @param isDockerEnv - Whether console is running in Docker
 * @returns Validation result with error message if invalid
 */
export function validateHostForEnvironment(host: string, isDockerEnv: boolean): {
  isValid: boolean;
  error?: string;
  suggestion?: string;
} {
  if (!host) {
    return { isValid: false, error: 'Host cannot be empty' };
  }

  // Remove protocol for comparison
  const hostWithoutProtocol = host.replace(/^https?:\/\//, '').split(':')[0].split('/')[0];

  if (isDockerEnv) {
    // Console is running in Docker
    if (hostWithoutProtocol === 'localhost' || hostWithoutProtocol === '127.0.0.1') {
      return {
        isValid: false,
        error: 'Console is running in Docker container. localhost/127.0.0.1 refers to the container itself, not the host machine.',
        suggestion: 'Use "kimbap-core" (if Core is Docker in same network) or "host.docker.internal" (if Core is on host machine) or IP/domain (if Core is on another server)'
      };
    }
  } else {
    // Console is running locally (not in Docker)
    if (hostWithoutProtocol === 'kimbap-core') {
      return {
        isValid: false,
        error: 'Console is running locally (not in Docker). Docker service name "kimbap-core" cannot be resolved.',
        suggestion: 'Use "localhost" (if Core is local) or IP/domain (if Core is on another server)'
      };
    }
    if (hostWithoutProtocol === 'host.docker.internal') {
      return {
        isValid: false,
        error: 'Console is running locally (not in Docker). "host.docker.internal" is only available inside Docker containers.',
        suggestion: 'Use "localhost" (if Core is local) or IP/domain (if Core is on another server)'
      };
    }
  }

  return { isValid: true };
}

/**
 * Add protocol to host if not present
 * IP addresses and localhost get http://, domains get https://
 * Docker service names (no dots) get http://
 *
 * @param host - Host string without protocol
 * @returns Host with protocol prefix
 */
export function addProtocolIfMissing(host: string): string {
  if (!host) return host;

  // Already has protocol
  if (host.startsWith('http://') || host.startsWith('https://')) {
    return host;
  }

  // Determine protocol based on host type
  const isIP = /^(\d{1,3}\.){3}\d{1,3}$/.test(host);
  const isLocalhost = host === 'localhost';
  const isHostDockerInternal = host === 'host.docker.internal';
  const isDockerServiceName = !host.includes('.') && !isIP && !isLocalhost && !isHostDockerInternal;

  // Use HTTP for: IP addresses, localhost, host.docker.internal, and Docker service names
  // Use HTTPS only for actual domain names (contain dots, like example.com)
  const protocol = (isIP || isLocalhost || isHostDockerInternal || isDockerServiceName) ? 'http' : 'https';

  return `${protocol}://${host}`;
}
