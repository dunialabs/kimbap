/**
 * Docker environment detection utilities
 * Detects if the application is running inside a Docker container
 * without relying on DOCKER_ENV environment variable
 */

import * as fs from 'fs';

/**
 * Check if running in Docker by examining filesystem markers
 * Returns false on error (safe fallback to local environment)
 *
 * Detection methods:
 * 1. Check for /.dockerenv file (most reliable)
 * 2. Check /proc/1/cgroup for docker/kubepods markers (Linux only)
 * 3. Check KUBERNETES_SERVICE_HOST environment variable
 */
export function isRunningInDocker(): boolean {
  try {
    // Method 1: Check for /.dockerenv file (most reliable across platforms)
    if (fs.existsSync('/.dockerenv')) {
      console.log('[DOCKER DETECTION] Found /.dockerenv file - running in Docker');
      return true;
    }

    // Method 2: Check /proc/1/cgroup (Linux only)
    // This works when /.dockerenv is not present
    if (process.platform === 'linux') {
      try {
        const cgroup = fs.readFileSync('/proc/1/cgroup', 'utf8');
        const isDocker = cgroup.includes('docker') || cgroup.includes('kubepods');
        if (isDocker) {
          console.log('[DOCKER DETECTION] Found Docker/Kubernetes markers in /proc/1/cgroup');
          return true;
        }
      } catch (cgroupError) {
        // /proc/1/cgroup doesn't exist or is not readable
        // This is fine, just continue to next method
      }
    }

    // Method 3: Check for KUBERNETES_SERVICE_HOST (Kubernetes pod)
    if (process.env.KUBERNETES_SERVICE_HOST) {
      console.log('[DOCKER DETECTION] Found KUBERNETES_SERVICE_HOST - running in Kubernetes');
      return true;
    }

    console.log('[DOCKER DETECTION] No Docker markers found - running in local environment');
    return false;
  } catch (error) {
    // On any error, assume local environment (safe fallback)
    console.warn('[DOCKER DETECTION] Error detecting Docker environment, assuming local:', error);
    return false;
  }
}

/**
 * Get the appropriate host list based on environment
 * Returns prioritized list of hosts to try when detecting KIMBAP Core
 *
 * Docker environment: Check both Docker service names and host machine
 * Local environment: Only check localhost
 *
 * Deployment scenarios:
 * 1. Both Docker (same network): console → kimbap-core (service name)
 * 2. Console Docker, Core on host: console → host.docker.internal:3002
 * 3. Both local: console → localhost
 * 4. Console local, Core Docker: console → localhost (with port mapping)
 *
 * Rationale:
 * - Docker env:
 *   - 'kimbap-core': Same network container-to-container communication (scenario 1)
 *   - 'host.docker.internal': Host machine services (scenario 2, common in development)
 * - Local env:
 *   - 'localhost': Local development (scenario 3, 4)
 */
export function getDefaultHostsByEnvironment(): string[] {
  if (isRunningInDocker()) {
    // Docker environment: check both container and host machine
    return [
      'kimbap-core',              // Priority 1: KIMBAP Core in same Docker network (most common in production)
      'host.docker.internal',   // Priority 2: KIMBAP Core on host machine (common in development/mixed deployment)
    ];
  } else {
    // Local environment: only check localhost
    // (covers both local dev and Docker core with port mapping)
    return ['localhost'];
  }
}
