import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { isRunningInDocker, getDefaultHostsByEnvironment } from '@/lib/docker-utils';
import { normalizeHostForDatabase, addProtocolIfMissing } from '@/lib/url-utils';
import { validateAndCacheMcpGatewayUrl } from '@/lib/mcp-gateway-cache';
import { invalidateProxyAdminUrlCache } from '@/lib/proxy-api';

interface Request10021 {
  common: {
    cmdId: number;
    userid?: string;
  };
  params: Record<string, never>; // No parameters needed
}

interface Response10021Data {
  isAvailable: number; // 1-available, 2-port not started, 3-port started but not kimbap core service, 4-already saved config
  kimbapCoreHost?: string; // Only return when isAvailable=1 or 4
  kimbapCorePort?: number; // Only return when isAvailable=1 or 4
}

/**
 * Protocol 10021: Detect KIMBAP Core service
 * New 3-tier priority system:
 * 1. Database config (user manual configuration) - if exists, return it
 * 2. MCP_GATEWAY_URL environment variable - validate but DO NOT save
 * 3. Auto-detection - scan possible hosts and SAVE if found
 *
 * Environment Detection (for auto-detection only):
 * - Uses filesystem-based detection (/.dockerenv, /proc/1/cgroup, KUBERNETES_SERVICE_HOST)
 * - Local Development: Prioritizes localhost/127.0.0.1
 * - Docker Environment: Prioritizes Docker service names (kimbap-core)
 *
 * Returns:
 * - isAvailable: 1 = KIMBAP Core found via MCP_GATEWAY_URL or auto-detection
 * - isAvailable: 2 = No KIMBAP Core found (ports not responding)
 * - isAvailable: 3 = Port responding but not KIMBAP Core service
 * - isAvailable: 4 = Already configured in database
 */
export async function handleProtocol10021(_body: Request10021): Promise<Response10021Data> {
  const DEFAULT_PORT = 3002;

  // ========================================
  // STEP 1: Check database for existing config
  // ========================================
  try {
    const prismaClient = prisma as any;
    const existingConfig = await prismaClient.config.findFirst();

    if (existingConfig && existingConfig.kimbap_core_host && existingConfig.kimbap_core_prot) {
      const savedHost = existingConfig.kimbap_core_host;
      const savedPort = existingConfig.kimbap_core_prot;

      console.log(`✓ [STEP 1] Found existing KIMBAP Core config in database: ${savedHost}:${savedPort}`);
      return {
        isAvailable: 4, // Already saved config
        kimbapCoreHost: savedHost,
        kimbapCorePort: savedPort,
      };
    }

    console.log(`ℹ [STEP 1] No config found in database, continuing to Step 2...`);
  } catch (dbError) {
    console.error('[STEP 1] Failed to check database config:', dbError);
    // Continue to Step 2
  }

  // ========================================
  // STEP 2: Validate MCP_GATEWAY_URL environment variable
  // ========================================
  const mcpGatewayUrl = process.env.MCP_GATEWAY_URL;
  if (mcpGatewayUrl) {
    console.log(`ℹ [STEP 2] Found MCP_GATEWAY_URL: ${mcpGatewayUrl}, validating...`);

    const validation = await validateAndCacheMcpGatewayUrl(mcpGatewayUrl);

    if (validation.isValid && validation.host && validation.port) {
      // MCP_GATEWAY_URL is valid and KIMBAP Core is accessible
      // DO NOT SAVE to database - keep environment variable flexible
      console.log(`✓ [STEP 2] MCP_GATEWAY_URL validation successful: ${validation.host}:${validation.port}`);
      console.log(`ℹ [STEP 2] Using MCP_GATEWAY_URL without saving to database (keeps env var flexible)`);

      return {
        isAvailable: 1, // KIMBAP Core is available
        kimbapCoreHost: validation.host,
        kimbapCorePort: validation.port,
      };
    } else {
      // Validation failed, log and continue to auto-detection
      console.warn(`✗ [STEP 2] MCP_GATEWAY_URL validation failed: ${validation.errorMessage}`);
      console.log(`ℹ [STEP 2] Continuing to auto-detection (Step 3)...`);
    }
  } else {
    console.log(`ℹ [STEP 2] MCP_GATEWAY_URL not set, continuing to auto-detection (Step 3)...`);
  }

  // ========================================
  // STEP 3: Auto-detection
  // ========================================
  console.log(`ℹ [STEP 3] Starting auto-detection...`);

  // Determine environment and get prioritized host list
  const isDocker = isRunningInDocker();
  const possibleHosts = getDefaultHostsByEnvironment();

  console.log(`ℹ [STEP 3] Environment: ${isDocker ? 'Docker' : 'Local'}`);
  console.log(`ℹ [STEP 3] Host detection priority order:`, possibleHosts);

  // Try each possible host
  for (const host of possibleHosts) {
    // IMPORTANT: Normalize host BEFORE validation
    // This ensures we validate the same URL that will be saved to database
    let normalizedHost = normalizeHostForDatabase(host);
    normalizedHost = addProtocolIfMissing(normalizedHost);

    console.log(`ℹ [STEP 3] Trying host: ${host} (normalized: ${normalizedHost})`);

    // Build validation URL from normalized host
    const url = `${normalizedHost}:${DEFAULT_PORT}`;

    try {
      // Try to connect to the KIMBAP Core service
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 5000); // 5 second timeout

      const response = await fetch(url, {
        method: 'GET',
        signal: controller.signal,
        headers: {
          'Accept': 'application/json',
        },
      });

      clearTimeout(timeoutId);

      // Check if the response is successful
      if (response.ok) {
        try {
          const data = await response.json();

          // Check if this is actually KIMBAP Core by verifying the service name
          if (data.service === 'Kimbap Core') {
            console.log(`✓ [STEP 3] KIMBAP Core detected and validated at ${normalizedHost}:${DEFAULT_PORT}`);

            // Save the validated URL to database (already normalized)
            try {
              const prismaClient = prisma as any;
              const existingConfig = await prismaClient.config.findFirst();

              if (existingConfig) {
                // Update existing configuration
                await prismaClient.config.update({
                  where: { id: existingConfig.id },
                  data: {
                    kimbap_core_host: normalizedHost,
                    kimbap_core_prot: DEFAULT_PORT,
                  },
                });
                console.log(`✓ [STEP 3] Updated KIMBAP Core config in database: ${normalizedHost}:${DEFAULT_PORT}`);
              } else {
                // Create new configuration record
                await prismaClient.config.create({
                  data: {
                    kimbap_core_host: normalizedHost,
                    kimbap_core_prot: DEFAULT_PORT,
                  },
                });
                console.log(`✓ [STEP 3] Created KIMBAP Core config in database: ${normalizedHost}:${DEFAULT_PORT}`);
              }

              // Invalidate proxy admin URL cache to ensure new configuration takes effect immediately
              invalidateProxyAdminUrlCache();
            } catch (dbError) {
              // Database operation failed, but does not affect return
              console.error('[STEP 3] Failed to save KIMBAP Core config to database:', dbError);
            }

            return {
              isAvailable: 1, // KIMBAP Core is available
              kimbapCoreHost: normalizedHost, // Return validated normalized host
              kimbapCorePort: DEFAULT_PORT,
            };
          } else {
            // Port is responding but it's not KIMBAP Core
            console.log(`✗ [STEP 3] Service at ${normalizedHost}:${DEFAULT_PORT} is not KIMBAP Core (service: ${data.service || 'unknown'})`);
          }
        } catch (parseError) {
          // Response is not valid JSON, not KIMBAP Core
          console.log(`✗ [STEP 3] Invalid response from ${normalizedHost}:${DEFAULT_PORT} - not KIMBAP Core`);
        }
      }
    } catch (error: any) {
      // Connection failed for this host
      if (error.name === 'AbortError') {
        console.log(`✗ [STEP 3] Timeout connecting to ${normalizedHost}:${DEFAULT_PORT}`);
      } else {
        console.log(`✗ [STEP 3] Failed to connect to ${normalizedHost}:${DEFAULT_PORT}:`, error.message);
      }
    }
  }

  // ========================================
  // STEP 3 FINAL: No KIMBAP Core found
  // ========================================
  console.log(`✗ [STEP 3] Auto-detection failed - KIMBAP Core not found on any host`);
  console.log(`ℹ Tried hosts:`, possibleHosts);
  console.log(`ℹ Please ensure KIMBAP Core is running on port ${DEFAULT_PORT}`);

  return {
    isAvailable: 2, // Port is not started
    // When isAvailable=2, do not return kimbapCoreHost and kimbapCorePort
  };
}