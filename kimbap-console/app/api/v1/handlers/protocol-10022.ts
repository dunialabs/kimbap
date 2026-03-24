import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import {
  normalizeHostForDatabase,
  addProtocolIfMissing,
  validateHostForEnvironment,
} from '@/lib/url-utils';
import { isRunningInDocker } from '@/lib/docker-utils';
import { invalidateProxyAdminUrlCache } from '@/lib/proxy-api';

const ALLOWED_PRIVATE_PATTERNS = [
  /^localhost$/i,
  /^127\.\d+\.\d+\.\d+$/,
  /^10\.\d+\.\d+\.\d+$/,
  /^172\.(1[6-9]|2\d|3[01])\.\d+\.\d+$/,
  /^192\.168\.\d+\.\d+$/,
  /^::1$/,
  /^host\.docker\.internal$/i,
  /^kimbap[-_]?core$/i,
];

interface Request10022 {
  common: {
    cmdId: number;
    userid?: string;
  };
  params: {
    host: string;
    port?: number;
  };
}

interface Response10022Data {
  isValid: number; // 1-validation successful and saved, 2-host/port invalid, 3-port is running but not kimbap core service
  message: string; // Operation result description
}

/**
 * Protocol 10022: Custom KIMBAP Core host and port configuration
 * Allows manual setting of KIMBAP Core host and port with validation
 *
 * Validates the provided host and port by:
 * 1. Checking if the port is responding (via HTTPS)
 * 2. Verifying it's actually KIMBAP Core service
 * 3. Saving to config table if validation succeeds
 *
 * If port is not provided, defaults to 443 (HTTPS standard port)
 *
 * Returns:
 * - isValid: 1 = validation successful and configuration saved
 * - isValid: 2 = host/port invalid or unable to connect
 * - isValid: 3 = port responds but is not KIMBAP Core service
 */
export async function handleProtocol10022(body: Request10022): Promise<Response10022Data> {
  const { host, port } = body.params;

  // 验证输入参数
  if (!host || !host.trim()) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
      field: 'host',
      details: 'Host cannot be empty',
    });
  }

  // 验证端口参数（如果提供）
  if (port !== undefined && (port <= 0 || port > 65535)) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, {
      field: 'port',
      details: 'Port must be a valid port number between 1-65535',
    });
  }

  const trimmedHost = host.trim();
  console.log(`🔍 Validating KIMBAP Core configuration: ${trimmedHost}${port ? ':' + port : ''}`);

  // ========================================
  // STEP 0: Validate environment compatibility
  // ========================================
  // Check if the host configuration is appropriate for current environment
  const isDockerEnv = isRunningInDocker();
  const envValidation = validateHostForEnvironment(trimmedHost, isDockerEnv);

  if (!envValidation.isValid) {
    console.error(`❌ Environment validation failed: ${envValidation.error}`);
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, {
      field: 'host',
      details: envValidation.error,
      suggestion: envValidation.suggestion,
      currentEnvironment: isDockerEnv ? 'Docker' : 'Local',
    });
  }

  console.log(`✓ Environment validation passed (${isDockerEnv ? 'Docker' : 'Local'} environment)`);

  // ========================================
  // STEP 1: Normalize BEFORE validation
  // ========================================
  // IMPORTANT: Normalize host and prepare URL BEFORE sending request
  // This ensures we validate the same URL that will be saved to database

  // 1.1 Add protocol if missing
  let normalizedHost: string;
  if (trimmedHost.startsWith('http://') || trimmedHost.startsWith('https://')) {
    // Host already has protocol
    normalizedHost = trimmedHost;
  } else {
    // Host doesn't have protocol, add appropriate protocol
    normalizedHost = addProtocolIfMissing(trimmedHost);
  }

  // 1.2 Normalize (minimal, only for kimbap-core protocol)
  normalizedHost = normalizeHostForDatabase(normalizedHost);

  // 1.2.1 Restrict to private/loopback addresses only (prevents DNS rebinding)
  {
    const hostOnly = normalizedHost
      .replace(/^https?:\/\//, '')
      .split(':')[0]
      .split('/')[0];
    const isPrivate = ALLOWED_PRIVATE_PATTERNS.some((p) => p.test(hostOnly));
    if (!isPrivate) {
      throw new ApiError(ErrorCode.INVALID_PARAMS, 400);
    }
  }

  // 1.3 Determine port for validation and database
  let portForValidation: number;
  if (port) {
    // Port explicitly provided
    portForValidation = port;
  } else {
    // Port not provided, extract from URL or use default
    try {
      const urlObj = new URL(normalizedHost);
      portForValidation = urlObj.port
        ? parseInt(urlObj.port)
        : urlObj.protocol === 'https:'
          ? 443
          : 80;
    } catch {
      // Fallback to default port
      portForValidation = normalizedHost.startsWith('https://') ? 443 : 80;
    }
  }

  // 1.4 Build validation URL (from normalized host)
  const validationUrl = port ? `${normalizedHost}:${port}` : normalizedHost;
  console.log(`🔍 Normalized validation URL: ${validationUrl}`);
  console.log(`ℹ  Will save to DB - Host: ${normalizedHost}, Port: ${portForValidation}`);

  // ========================================
  // STEP 2: Validate with normalized URL
  // ========================================
  try {
    // 尝试连接到指定的KIMBAP Core服务
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000); // 5秒超时

    const fetchOptions: RequestInit = {
      method: 'GET',
      signal: controller.signal,
      headers: {
        Accept: 'application/json',
      },
    };

    const response = await fetch(validationUrl, fetchOptions);
    clearTimeout(timeoutId);

    // 检查响应是否成功
    if (response.ok) {
      try {
        const data = await response.json();

        // 验证是否为KIMBAP Core服务
        if (data.service === 'Kimbap Core') {
          console.log(`✓ KIMBAP Core service validation successful: ${validationUrl}`);

          // ========================================
          // STEP 3: Save validated configuration
          // ========================================
          // Save the validated URL (already normalized)
          try {
            // 查找是否已有配置记录
            const existingConfig = await prisma.config.findFirst();

            const dbData = {
              kimbap_core_host: normalizedHost,
              kimbap_core_prot: portForValidation,
            };

            if (existingConfig) {
              // Update existing configuration
              await prisma.config.update({
                where: { id: existingConfig.id },
                data: dbData,
              });
              console.log(
                `✓ Updated KIMBAP Core configuration: ${normalizedHost}:${portForValidation}`,
              );
            } else {
              // Create new configuration record
              await prisma.config.create({
                data: dbData,
              });
              console.log(
                `✓ Created KIMBAP Core configuration: ${normalizedHost}:${portForValidation}`,
              );
            }

            // Invalidate proxy admin URL cache to ensure new configuration takes effect immediately
            invalidateProxyAdminUrlCache();

            return {
              isValid: 1,
              message: `KIMBAP Core configuration validation successful and saved: ${normalizedHost}:${portForValidation}`,
            };
          } catch (dbError) {
            // Database operation failed
            console.error('Failed to save KIMBAP Core configuration to database:', dbError);
            return {
              isValid: 1, // Validation successful, but save failed
              message: `KIMBAP Core configuration validation successful, but failed to save to database: ${validationUrl}`,
            };
          }
        } else {
          // Port responds but is not KIMBAP Core service
          console.log(
            `✗ ${validationUrl} responds but is not KIMBAP Core service (service: ${data.service || 'unknown'})`,
          );
          return {
            isValid: 3,
            message: `${validationUrl} is running other service, not KIMBAP Core: ${data.service || 'unknown service'}`,
          };
        }
      } catch (parseError) {
        // Response is not valid JSON
        console.log(`✗ ${validationUrl} returned invalid JSON response`);
        return {
          isValid: 3,
          message: `${validationUrl} returned invalid response format, not a KIMBAP Core service`,
        };
      }
    } else {
      // HTTPS response not successful
      console.log(`✗ ${validationUrl} response failed: HTTPS ${response.status}`);
      return {
        isValid: 2,
        message: `Cannot connect to ${validationUrl} (HTTPS ${response.status})`,
      };
    }
  } catch (error: any) {
    // Connection failed
    if (error.name === 'AbortError') {
      console.log(`✗ Connection to ${validationUrl} timed out`);
      return {
        isValid: 2,
        message: `Connection to ${validationUrl} timed out, please check network connection and service status`,
      };
    } else {
      console.log(`✗ Connection to ${validationUrl} failed:`, error.message);
      return {
        isValid: 2,
        message: `Cannot connect to ${validationUrl}: ${error.message}`,
      };
    }
  }
}
