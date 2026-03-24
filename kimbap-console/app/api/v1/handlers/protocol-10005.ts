import { ApiError, ErrorCode } from '@/lib/error-codes';
import { randomUUID } from 'crypto';
import { CryptoUtils } from '@/lib/crypto';
import {
  startMCPServer,
  stopMCPServer,
  updateServer,
  getUsers,
  countServers,
  createServer,
  getServers,
  deleteServer,
  makeProxyRequestWithUserId,
} from '@/lib/proxy-api';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { validateHttpsBaseUrl } from '@/lib/rest-api-utils';
import { LicenseService } from '@/license-system';
import { log } from 'console';
import { ServerAuthType } from '@/types/api';
import {
  extractOAuthEndpointOverridesFromAuthConf,
  resolveOAuthEndpoints,
} from '@/lib/oauth-endpoint-overrides';

interface Request10005 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    handleType: number;
    proxyId?: number;
    toolId?: string;
    toolTmplId?: string;
    masterPwd?: string;
    allowUserInput?: number; // 1-True, else-False
    category?: number; // 1-Template, 2-Custom Remote HTTP, 3-REST API, 4-Skills, 5-Custom Stdio
    serverName?: string; // Server name (for category=4 Skills)
    customRemoteConfig?: {
      url: string;
      headers: Record<string, string>;
    }; // Custom MCP URL configuration (for category=2)
    stdioConfig?: {
      command: string;
      args?: string[];
      env?: Record<string, string>;
      cwd?: string;
    };
    restApiConfig?: string; // REST API configuration JSON string (for category=3)
    authConf?: Array<{
      key: string;
      value: string;
      dataType: number;
    }>;
    functions?: Array<{
      funcName: string;
      enabled: boolean;
      dangerLevel?: number; // 0: No validation, 1: hint only, 2: validation required
      description?: string; // description for the function
    }>;
    resources?: Array<{
      uri: string;
      enabled: boolean;
    }>;
    cachePolicies?: {
      tools?: Record<string, any>;
      prompts?: Record<string, any>;
      resources?: {
        exact?: Record<string, any>;
        patterns?: any[];
      };
    };
    lazyStartEnabled?: boolean; // Enable lazy loading for this server
    publicAccess?: boolean; // Public access for this server, default is false.
    anonymousAccess?: boolean; // Enable anonymous access for this server
    anonymousRateLimit?: number; // Rate limit for anonymous access (req/min per IP)
  };
}

function normalizeArgsForComparison(args?: string[]): string {
  return JSON.stringify([...(args ?? [])].sort());
}

function normalizeOptionalCwd(cwd?: string): string | undefined {
  const trimmed = cwd?.trim();
  return trimmed ? trimmed : undefined;
}

/**
 * Process and encrypt launch configuration
 * @param launchConfig - The configuration to process
 * @param authConf - Auth configuration array to apply
 * @param proxyId - Proxy ID for finding owner
 * @param masterPwd - Master password for decryption
 * @param userid - User ID for token authentication
 * @returns Processed and optionally encrypted config as JSON string
 */
async function processLaunchConfig(
  launchConfig: any,
  authConf?: Array<{ key: string; value: string; dataType: number }>,
  proxyId?: number,
  masterPwd?: string,
  userid?: string,
): Promise<string> {
  let processedConfig = JSON.parse(JSON.stringify(launchConfig)); // Deep clone

  // Apply authConf changes if provided
  // Note: OAuth code (YOUR_OAUTH_CODE) and redirect URI (YOUR_OAUTH_REDIRECT_URL)
  // are passed through authConf and will be replaced like other credentials
  if (authConf && authConf.length > 0) {
    const escapeRegExp = (value: string): string => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

    const replacementEntries: Array<{ key: string; value: string }> = [];
    const seenKeys = new Set<string>();

    for (const auth of authConf) {
      if (auth.dataType !== 1) {
        continue;
      }

      // Validate input
      if (
        !auth.key ||
        typeof auth.key !== 'string' ||
        !auth.value ||
        typeof auth.value !== 'string' ||
        auth.value === ''
      ) {
        console.warn('Invalid auth configuration:', { key: auth.key, value: auth.value });
        continue;
      }

      // Keep first occurrence to preserve current behavior for duplicate keys.
      if (seenKeys.has(auth.key)) {
        continue;
      }

      seenKeys.add(auth.key);
      replacementEntries.push({ key: auth.key, value: auth.value });
    }

    if (replacementEntries.length > 0) {
      const replacementMap = new Map<string, string>(
        replacementEntries.map((entry) => [entry.key, entry.value]),
      );
      const sortedKeys = replacementEntries
        .map((entry) => entry.key)
        .sort((a, b) => b.length - a.length);
      const replacementPattern = new RegExp(
        sortedKeys.map((key) => escapeRegExp(key)).join('|'),
        'g',
      );

      const configStr = JSON.stringify(processedConfig);
      const updatedConfigStr = configStr.replace(
        replacementPattern,
        (matchedKey) => replacementMap.get(matchedKey) ?? matchedKey,
      );

      // Validate the result is still valid JSON
      try {
        processedConfig = JSON.parse(updatedConfigStr);
      } catch (error) {
        console.error('Failed to parse config after variable replacement:', error);
        throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
          details: 'Configuration became invalid after variable replacement',
        });
      }
    }
  }

  // Encrypt if proxyId and masterPwd are provided
  if (proxyId && masterPwd) {
    try {
      // Find the owner of the proxy
      const ownerResult = await getUsers(
        {
          proxyId: proxyId,
          role: 1, // Owner role
        },
        userid,
      );
      const owner = ownerResult.users[0];

      if (!owner || !owner.encryptedToken) {
        throw new ApiError(ErrorCode.INVALID_REQUEST);
      }

      // Decrypt the owner token
      let ownerToken: string;
      try {
        ownerToken = await CryptoUtils.decryptDataFromString(owner.encryptedToken, masterPwd);

        // Validate decrypted token is not empty
        if (!ownerToken) {
          throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
        }
      } catch (error) {
        // If decryption fails, it's likely due to invalid master password
        console.error('Failed to decrypt owner token:', error);
        throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
      }

      // Encrypt the config with the owner token
      const encryptedConfig = await CryptoUtils.encryptData(
        JSON.stringify(processedConfig),
        ownerToken,
      );

      // Return encrypted config as JSON string
      return JSON.stringify(encryptedConfig);
    } catch (error) {
      console.error('Failed to encrypt launch config:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
    }
  }

  // Return unencrypted config as JSON string
  return JSON.stringify(processedConfig);
}

export async function handleProtocol10005(body: Request10005): Promise<any> {
  const {
    handleType,
    proxyId,
    toolId,
    toolTmplId,
    masterPwd,
    allowUserInput,
    category,
    serverName: inputServerName,
    customRemoteConfig,
    stdioConfig,
    restApiConfig,
    authConf,
    functions,
    resources,
    cachePolicies,
    lazyStartEnabled,
    publicAccess,
    anonymousAccess,
    anonymousRateLimit,
  } = body.params || {};

  // Log received parameters for debugging
  const logParams = {
    ...body.params,
  };
  delete logParams.masterPwd;
  delete logParams.restApiConfig;
  delete logParams.authConf;
  delete logParams.customRemoteConfig;
  delete logParams.stdioConfig;
  console.log('Protocol 10005 received params:', logParams);

  // Validate handleType
  if (!handleType || handleType < 1 || handleType > 6) {
    throw new ApiError(ErrorCode.INVALID_REQUEST);
  }

  try {
    switch (handleType) {
      case 1: {
        // Add tool
        try {
          // Check tool creation limit before proceeding
          // Get current tool count for this proxy
          const currentToolCountResult = await countServers(
            {
              proxyId: proxyId || 0,
            },
            body.common.userid,
          );
          const currentToolCount = currentToolCountResult.count;

          // Check license limits
          const licenseService = LicenseService.getInstance();
          const limitCheck = await licenseService.checkToolCreationLimit(currentToolCount);

          if (!limitCheck.allowed) {
            throw new ApiError(ErrorCode.TOOL_CREATION_LIMIT_EXCEEDED);
          }
        } catch (error) {
          if (error instanceof ApiError) {
            throw error;
          }
          console.error('Failed to check tool creation limit:', error);
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

        const serverId = randomUUID().replace(/-/g, ''); // Remove hyphens to get 32-character ID
        let serverName = '';
        let launchConfig = {};
        let authType = 0; // Default authType
        let template: any = null; // Store template for configTemplate
        const toolCategory = category || 1; // Default to category 1

        if (toolCategory === 1) {
          // Category 1: Template-based tool
          if (!toolTmplId) {
            throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
              field: 'toolTmplId',
              details: 'toolTmplId is required for category=1 (template tools)',
            });
          }

          try {
            const kimbapCloudApi = new KimbapCloudApiService();
            const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();
            template = templates.find((t) => t.toolTmplId === toolTmplId);

            if (!template) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                details: 'Tool template not found',
              });
            }

            // Clone template to avoid mutating shared in-memory objects.
            template = JSON.parse(JSON.stringify(template));

            // OAuth endpoint overrides are passed through authConf.
            // They must be persisted into configTemplate so Core can read them later.
            if (template.oAuthConfig && authConf && authConf.length > 0) {
              const endpointOverrides = extractOAuthEndpointOverridesFromAuthConf(authConf);
              if (
                endpointOverrides.authorizationUrl ||
                endpointOverrides.tokenUrl ||
                endpointOverrides.baseUrl
              ) {
                try {
                  const resolvedEndpoints = resolveOAuthEndpoints(
                    template.oAuthConfig,
                    endpointOverrides,
                  );
                  template.oAuthConfig.authorizationUrl = resolvedEndpoints.authorizationUrl;
                  template.oAuthConfig.tokenUrl = resolvedEndpoints.tokenUrl;
                } catch (error: any) {
                  throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                    field: 'authConf',
                    details: error?.message || 'Invalid OAuth endpoint configuration',
                  });
                }
              }
            }

            serverName = template.name;
            if (allowUserInput === 1) {
              serverName = template.name + ' Personal';
            }

            launchConfig = template.mcpJsonConf;
            authType = template.authType || 0; // Extract authType from template
          } catch (error) {
            console.error('Failed to fetch tool templates:', error);
            throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
          }
        } else if (toolCategory === 2) {
          if (customRemoteConfig && customRemoteConfig.url) {
            const urlIssue = validateHttpsBaseUrl(customRemoteConfig.url);
            if (urlIssue) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'customRemoteConfig.url',
                details: urlIssue,
              });
            }
            const configHeaders = customRemoteConfig.headers ?? {};

            launchConfig = {
              url: customRemoteConfig.url,
              headers: configHeaders,
            };

            serverName = `Custom MCP Server (${new URL(customRemoteConfig.url).hostname})`;

            if (allowUserInput === 1) {
              serverName = serverName + ' Personal';
              template = {
                url: customRemoteConfig.url,
                headers: configHeaders,
              };
            }

            authType = 1;
          } else {
            throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
              field: 'customRemoteConfig',
              details: 'customRemoteConfig with url is required for category=2 (custom MCP tools)',
            });
          }
        } else if (toolCategory === 5) {
          if (!stdioConfig || !stdioConfig.command) {
            throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
              field: 'stdioConfig',
              details:
                'stdioConfig with command is required for category=5 (custom stdio MCP tools)',
            });
          }

          if (typeof stdioConfig.command !== 'string' || !stdioConfig.command.trim()) {
            throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
              field: 'stdioConfig.command',
              details: 'stdioConfig.command is required and must be a non-empty string',
            });
          }

          if (stdioConfig.args !== undefined) {
            if (!Array.isArray(stdioConfig.args)) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'stdioConfig.args',
                details: 'args must be an array of strings',
              });
            }
            if (stdioConfig.args.some((arg: unknown) => typeof arg !== 'string')) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'stdioConfig.args',
                details: 'All args must be strings',
              });
            }
          }

          if (stdioConfig.env !== undefined) {
            if (
              typeof stdioConfig.env !== 'object' ||
              stdioConfig.env === null ||
              Array.isArray(stdioConfig.env)
            ) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'stdioConfig.env',
                details: 'env must be an object with string values',
              });
            }
            for (const [envKey, envValue] of Object.entries(stdioConfig.env)) {
              if (typeof envValue !== 'string') {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: `stdioConfig.env.${envKey}`,
                  details: 'All environment variable values must be strings',
                });
              }
            }
          }

          if (stdioConfig.cwd !== undefined && typeof stdioConfig.cwd !== 'string') {
            throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
              field: 'stdioConfig.cwd',
              details: 'cwd must be a string',
            });
          }

          const normalizedCwd = normalizeOptionalCwd(stdioConfig.cwd);

          launchConfig = {
            command: stdioConfig.command.trim(),
            args: stdioConfig.args ?? [],
            env: stdioConfig.env ?? {},
            ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
          };

          const commandName =
            stdioConfig.command.trim().split('/').pop()?.split('\\').pop() ||
            stdioConfig.command.trim();
          serverName = `Custom MCP Server (${commandName})`;

          if (allowUserInput === 1) {
            serverName = serverName + ' Personal';
            template = {
              command: stdioConfig.command.trim(),
              args: stdioConfig.args ?? [],
              env: stdioConfig.env ?? {},
              ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
            };
          }

          authType = 1;
        } else if (toolCategory === 3) {
          // Category 3: REST API tool
          if (!restApiConfig) {
            throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
              field: 'restApiConfig',
              details: 'restApiConfig is required for category=3 (REST API tools)',
            });
          }

          try {
            // Parse REST API configuration
            const restApi = JSON.parse(restApiConfig);

            const baseUrl = restApi.baseUrl?.trim() ?? '';
            if (baseUrl === '') {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'restApiConfig.baseUrl',
                details: 'Base URL is required for REST API tools',
              });
            }
            const baseUrlIssue = validateHttpsBaseUrl(baseUrl);
            if (baseUrlIssue) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'restApiConfig.baseUrl',
                details: baseUrlIssue,
              });
            }

            const tools = restApi.tools ?? [];
            if (tools.length === 0) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'restApiConfig.tools',
                details: 'At least one tool is required for REST API tools',
              });
            }

            serverName = restApi.name.trim() || `REST API Tool ${Date.now()}`;
            if (allowUserInput === 1) {
              serverName = serverName + ' Personal';
            }

            // Extract and remove auth field
            const auth = restApi.auth ?? { type: 'none' };
            delete restApi.auth;

            // Build launchConfig
            launchConfig = {
              command: 'docker',
              args: [
                'run',
                '--pull=always',
                '-i',
                '--rm',
                '-e',
                'accessToken',
                '-e',
                'GATEWAY_CONFIG',
                'ghcr.io/dunialabs/mcp-servers/rest-gateway',
              ],
              auth: auth,
            };

            // Build template
            template = {
              apis: [restApi],
            };

            // Set authType
            authType = 1;
          } catch (error) {
            console.error('Failed to parse restApiConfig:', error);
            throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
              details: 'Invalid REST API configuration JSON',
            });
          }
        } else if (toolCategory === 4) {
          // Category 4: Skills tool
          // Skills are stored on Core server at skills/{serverId}/
          // The Skills MCP Server will read from this directory
          serverName = inputServerName || 'Skills MCP Server';

          // Build launch config for Skills MCP Server
          // The serverId will be used to locate skills directory on Core
          launchConfig = {
            command: 'docker',
            args: [
              'run',
              '--pull=always',
              '-i',
              '--rm',
              '-v',
              `./skills/${serverId}:/app/skills:ro`,
              '-e',
              'skills_dir=/app/skills',
              'ghcr.io/dunialabs/mcp-servers/skills:latest',
            ],
          };

          // Set template for Skills (required when allowUserInput=true)
          template = {
            type: 'skills',
            serverName: serverName,
          };

          authType = 1; // Use authType 1 (same as custom MCP) - Core requires authType >= 1
        } else {
          throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
            details:
              'Invalid category. Must be 1 (template), 2 (custom remote HTTP), 3 (REST API), 4 (Skills), or 5 (custom stdio)',
          });
        }

        // Process and encrypt launchConfig
        let finalLaunchConfig: string;
        if (allowUserInput === 1 && toolCategory === 3) {
          finalLaunchConfig = JSON.stringify(launchConfig);
        } else {
          finalLaunchConfig = await processLaunchConfig(
            launchConfig,
            authConf,
            proxyId,
            masterPwd,
            body.common.userid,
          );
        }

        // Process capabilities from functions and resources
        const capabilities: any = {
          tools: {},
          resources: {},
        };

        // Save to database - initially set enabled to false
        try {
          const serverResult = await createServer(
            {
              serverId: serverId,
              serverName: serverName,
              enabled: false, // Initially disabled until MCP server starts
              launchConfig: finalLaunchConfig, // Use encrypted or plain config as string
              capabilities: capabilities, // Store processed capabilities as object
              allowUserInput: Boolean(allowUserInput === 1), // Explicitly convert: 1 -> true, else -> false
              proxyId: proxyId || 0, // Use user-provided proxyId, default to 0 if not provided
              toolTmplId: toolTmplId || '', // Save tool template ID
              authType: authType, // Authentication type from template
              configTemplate: template ? JSON.stringify(template) : '{}', // Convert template to JSON string or empty object
              category: toolCategory, // Pass category field (1 for template, 2 for custom MCP URL)
              lazyStartEnabled: lazyStartEnabled ?? true, // Default to true when not provided
              publicAccess: publicAccess ?? false, // Default to false when not provided
              anonymousAccess: anonymousAccess ?? false,
              anonymousRateLimit: anonymousRateLimit ?? 10,
            },
            body.common.userid,
          );
          const server = serverResult.server;

          // Start the MCP server using proxy API with owner's access token
          let isStartServer = 2; // Default to failed
          try {
            if (proxyId && masterPwd) {
              // Get owner's access token if proxyId and masterPwd are provided
              const ownerResult = await getUsers(
                {
                  proxyId: proxyId,
                  role: 1, // Owner role
                },
                body.common.userid,
              );
              const owner = ownerResult.users[0];

              if (owner && owner.encryptedToken) {
                try {
                  const ownerToken = await CryptoUtils.decryptDataFromString(
                    owner.encryptedToken,
                    masterPwd,
                  );

                  if (ownerToken) {
                    await startMCPServer(server.serverId, ownerToken);
                    isStartServer = 1; // Success
                  }
                } catch (decryptError) {
                  console.error('Failed to decrypt owner token for starting server:', decryptError);
                  isStartServer = 2; // Failed
                }
              }
            }
          } catch (error) {
            console.error('Failed to start MCP server:', error);
            isStartServer = 2; // Failed
            // Server remains disabled in database since start failed
          }

          const result = {
            toolId: server.serverId,
            isStartServer: isStartServer,
          };

          return result;
        } catch (dbError) {
          console.error('Database error:', dbError);
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }
      }

      case 2: // Edit tool
        // Validate required toolId
        if (!toolId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'toolId',
            details: 'toolId is required for handleType=2 (Edit tool)',
          });
        }

        try {
          // Fetch existing server
          const existingServerResult = await getServers(
            {
              serverId: toolId,
            },
            body.common.userid,
          );
          const existingServer = existingServerResult.servers[0];

          if (!existingServer) {
            throw new ApiError(ErrorCode.SERVER_NOT_FOUND, 404);
          }

          const updateData: any = {
            updatedAt: Math.floor(Date.now() / 1000),
          };

          // Process authConf if provided - reuse the common method
          if (authConf && authConf.length > 0) {
            if (existingServer.category === 1) {
              if (
                existingServer.authType !== ServerAuthType.ApiKey ||
                existingServer.allowUserInput
              ) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'authConf',
                  details:
                    'authConf cannot be modified for API Key authentication or allowUserInput is true',
                });
              }
            }

            // Parse existing launch config to get the base configuration
            let currentConfig = JSON.parse(existingServer.launchConfig);

            // If it's encrypted, we need to decrypt it first
            if (currentConfig.data && currentConfig.iv && currentConfig.salt && currentConfig.tag) {
              // Need to decrypt first if we have the credentials
              if (proxyId && masterPwd) {
                try {
                  const ownerResult = await getUsers(
                    {
                      proxyId: proxyId,
                      role: 1, // Owner role
                    },
                    body.common.userid,
                  );
                  const owner = ownerResult.users[0];

                  if (owner && owner.encryptedToken) {
                    let ownerToken: string;
                    try {
                      ownerToken = await CryptoUtils.decryptDataFromString(
                        owner.encryptedToken,
                        masterPwd,
                      );

                      // Validate decrypted token is not empty
                      if (!ownerToken) {
                        throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
                      }
                    } catch (error) {
                      // If decryption fails, it's likely due to invalid master password
                      console.error('Failed to decrypt owner token:', error);
                      throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
                    }

                    // Decrypt current config
                    const decryptedConfig = await CryptoUtils.decryptData(
                      currentConfig,
                      ownerToken,
                    );
                    if (!decryptedConfig) {
                      throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
                    }
                    currentConfig = JSON.parse(decryptedConfig);
                  }
                } catch (error) {
                  console.error('Failed to decrypt existing config:', error);
                  throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
                }
              }
            }

            // Process and potentially re-encrypt the config
            updateData.launchConfig = await processLaunchConfig(
              currentConfig,
              authConf,
              proxyId,
              masterPwd,
              body.common.userid,
            );
          }

          if (existingServer.category === 3 && restApiConfig) {
            const oldConfigTemplate = JSON.parse(existingServer.configTemplate ?? '{}');
            const oldRestApi = oldConfigTemplate.apis?.[0] ?? {};
            const oldBaseUrl = oldRestApi.baseUrl?.trim() ?? '';

            const restApi = JSON.parse(restApiConfig);
            let newServerName: string = restApi.name?.trim() ?? '';
            if (newServerName !== '') {
              if (existingServer.allowUserInput) {
                newServerName = newServerName + ' Personal';
              }
              if (newServerName !== existingServer.serverName) {
                updateData.serverName = newServerName;
              }
            }

            const tools = restApi.tools ?? [];
            if (tools.length === 0) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'restApiConfig.tools',
                details: 'At least one tool is required for REST API tools',
              });
            }
            const baseUrl = restApi.baseUrl?.trim() ?? '';
            if (baseUrl) {
              const editBaseUrlIssue = validateHttpsBaseUrl(baseUrl);
              if (editBaseUrlIssue) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'restApiConfig.baseUrl',
                  details: editBaseUrlIssue,
                });
              }
            }
            if (existingServer.allowUserInput) {
              if (baseUrl === '' || baseUrl !== oldBaseUrl) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'restApiConfig.baseUrl',
                  details: 'Base URL is required for REST API tools and cannot be changed',
                });
              }

              const newAuth = restApi.auth ?? { type: 'none' };
              const auth = JSON.stringify(newAuth);
              const launchConfig = JSON.parse(existingServer.launchConfig);
              const oldAuth = JSON.stringify(launchConfig.auth ?? { type: 'none' });
              if (auth !== oldAuth) {
                launchConfig.auth = newAuth;
                updateData.launchConfig = JSON.stringify(launchConfig);
              }

              delete restApi.auth;
            } else {
              if (baseUrl === '') {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'restApiConfig.baseUrl',
                  details: 'Base URL is required for REST API tools',
                });
              }
              // Extract and remove auth field
              const auth = restApi.auth || {};
              delete restApi.auth;

              // Build launchConfig for REST API tool
              const restApiLaunchConfig = {
                command: 'docker',
                args: [
                  'run',
                  '--pull=always',
                  '-i',
                  '--rm',
                  '-e',
                  'accessToken',
                  '-e',
                  'GATEWAY_CONFIG',
                  'ghcr.io/dunialabs/mcp-servers/rest-gateway',
                ],
                auth: auth,
              };

              // Process and encrypt launchConfig
              const finalLaunchConfig = await processLaunchConfig(
                restApiLaunchConfig,
                authConf,
                proxyId,
                masterPwd,
                body.common.userid,
              );

              updateData.launchConfig = finalLaunchConfig;
            }

            // Build template
            const template = {
              apis: [restApi],
            };

            updateData.configTemplate = JSON.stringify(template);
          }

          if (existingServer.category === 2 && customRemoteConfig) {
            let newServerUrl: string = customRemoteConfig.url?.trim() ?? '';
            if (newServerUrl === '') {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'customRemoteConfig.url',
                details: 'URL is required for Custom MCP URL tools',
              });
            }
            const editUrlIssue = validateHttpsBaseUrl(newServerUrl);
            if (editUrlIssue) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'customRemoteConfig.url',
                details: editUrlIssue,
              });
            }
            if (existingServer.allowUserInput) {
              let index = newServerUrl.indexOf('?');
              let newUrl = newServerUrl;
              if (index !== -1) {
                newUrl = newServerUrl.substring(0, index);
              }
              const oldConfigTemplate = JSON.parse(existingServer.configTemplate ?? '{}');
              let oldUrl = oldConfigTemplate.url ?? '';
              index = oldUrl.indexOf('?');
              if (index !== -1) {
                oldUrl = oldUrl.substring(0, index);
              }
              if (newUrl !== oldUrl) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'customRemoteConfig.url',
                  details: 'URL is required for Custom MCP URL tools and cannot be changed',
                });
              }
            }

            const launchConfig = {
              url: newServerUrl,
              headers: customRemoteConfig.headers ?? {},
            };

            const finalLaunchConfig = await processLaunchConfig(
              launchConfig,
              undefined,
              proxyId,
              masterPwd,
              body.common.userid,
            );
            updateData.launchConfig = finalLaunchConfig;

            if (existingServer.allowUserInput) {
              updateData.configTemplate = JSON.stringify(launchConfig);
            }
          }

          if (existingServer.category === 5 && stdioConfig) {
            if (typeof stdioConfig.command !== 'string' || !stdioConfig.command.trim()) {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'stdioConfig.command',
                details: 'Command must be a non-empty string for stdio Custom MCP tools',
              });
            }

            if (stdioConfig.args !== undefined) {
              if (!Array.isArray(stdioConfig.args)) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'stdioConfig.args',
                  details: 'args must be an array of strings',
                });
              }
              if (stdioConfig.args.some((arg: unknown) => typeof arg !== 'string')) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'stdioConfig.args',
                  details: 'All args must be strings',
                });
              }
            }

            if (stdioConfig.env !== undefined) {
              if (
                typeof stdioConfig.env !== 'object' ||
                stdioConfig.env === null ||
                Array.isArray(stdioConfig.env)
              ) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'stdioConfig.env',
                  details: 'env must be an object with string values',
                });
              }
              for (const [envKey, envValue] of Object.entries(stdioConfig.env)) {
                if (typeof envValue !== 'string') {
                  throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                    field: `stdioConfig.env.${envKey}`,
                    details: 'All environment variable values must be strings',
                  });
                }
              }
            }

            if (stdioConfig.cwd !== undefined && typeof stdioConfig.cwd !== 'string') {
              throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                field: 'stdioConfig.cwd',
                details: 'cwd must be a string',
              });
            }

            if (existingServer.allowUserInput) {
              const oldConfigTemplate = JSON.parse(existingServer.configTemplate ?? '{}');
              if (
                oldConfigTemplate.command &&
                stdioConfig.command.trim() !== oldConfigTemplate.command
              ) {
                throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                  field: 'stdioConfig.command',
                  details: 'Command cannot be changed for personal stdio tools',
                });
              }
              if (stdioConfig.args !== undefined) {
                const oldArgs = normalizeArgsForComparison(oldConfigTemplate.args);
                const newArgs = normalizeArgsForComparison(stdioConfig.args);
                if (oldArgs !== newArgs) {
                  throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
                    field: 'stdioConfig.args',
                    details:
                      'Args cannot be changed for personal stdio tools. Only env parameters can be modified.',
                  });
                }
              }
            }

            const normalizedCwd = normalizeOptionalCwd(stdioConfig.cwd);

            const launchConfig = {
              command: stdioConfig.command.trim(),
              args: stdioConfig.args ?? [],
              env: stdioConfig.env ?? {},
              ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
            };

            const finalLaunchConfig = await processLaunchConfig(
              launchConfig,
              undefined,
              proxyId,
              masterPwd,
              body.common.userid,
            );
            updateData.launchConfig = finalLaunchConfig;

            if (existingServer.allowUserInput) {
              updateData.configTemplate = JSON.stringify(launchConfig);
            }
          }

          if (functions || resources || cachePolicies) {
            let existingCapabilities: any = {};
            try {
              existingCapabilities = existingServer.capabilities
                ? JSON.parse(existingServer.capabilities)
                : {};
            } catch {
              existingCapabilities = {};
            }

            const capabilities: any = {
              tools: { ...(existingCapabilities.tools || {}) },
              resources: { ...(existingCapabilities.resources || {}) },
              prompts: { ...(existingCapabilities.prompts || {}) },
              ...(existingCapabilities.resourceCachePolicies
                ? { resourceCachePolicies: existingCapabilities.resourceCachePolicies }
                : {}),
            };

            if (functions && functions.length > 0) {
              for (const func of functions) {
                const prevTool = capabilities.tools[func.funcName] || {};
                capabilities.tools[func.funcName] = {
                  ...prevTool,
                  enabled: func.enabled,
                  dangerLevel: func.dangerLevel !== undefined ? func.dangerLevel : 0,
                  description: func.description || '',
                };
              }
            }

            if (resources && resources.length > 0) {
              for (const resource of resources) {
                const prevResource = capabilities.resources[resource.uri] || {};
                capabilities.resources[resource.uri] = {
                  ...prevResource,
                  enabled: resource.enabled,
                };
              }
            }

            if (cachePolicies?.tools) {
              for (const [toolName, policy] of Object.entries(cachePolicies.tools)) {
                const prevTool = capabilities.tools[toolName] || {};
                capabilities.tools[toolName] = {
                  ...prevTool,
                  cache: policy,
                };
              }
            }

            if (cachePolicies?.prompts) {
              for (const [promptName, policy] of Object.entries(cachePolicies.prompts)) {
                const prevPrompt = capabilities.prompts[promptName] || {};
                capabilities.prompts[promptName] = {
                  ...prevPrompt,
                  cache: policy,
                };
              }
            }

            if (cachePolicies?.resources?.exact) {
              for (const [uri, policy] of Object.entries(cachePolicies.resources.exact)) {
                const prevResource = capabilities.resources[uri] || {};
                capabilities.resources[uri] = {
                  ...prevResource,
                  cache: policy,
                };
              }
            }

            if (cachePolicies?.resources?.patterns) {
              capabilities.resourceCachePolicies = {
                ...(capabilities.resourceCachePolicies || {}),
                patterns: cachePolicies.resources.patterns,
              };
            }

            updateData.capabilities = JSON.stringify(capabilities);
          }

          // Call appropriate proxy API methods based on what was updated
          // Use unified updateServer to update launchConfig and capabilities
          const serverUpdateData: any = {};

          // Only add fields that have been modified
          if (updateData.launchConfig && updateData.launchConfig !== existingServer.launchConfig) {
            serverUpdateData.launchConfig = updateData.launchConfig;
          }

          if (updateData.capabilities && updateData.capabilities !== existingServer.capabilities) {
            serverUpdateData.capabilities = updateData.capabilities;
          }

          if (
            updateData.configTemplate &&
            updateData.configTemplate !== existingServer.configTemplate
          ) {
            serverUpdateData.configTemplate = updateData.configTemplate;
          }

          if (updateData.serverName && updateData.serverName !== existingServer.serverName) {
            serverUpdateData.serverName = updateData.serverName;
          }

          if (
            lazyStartEnabled !== undefined &&
            lazyStartEnabled !== existingServer.lazyStartEnabled
          ) {
            serverUpdateData.lazyStartEnabled = lazyStartEnabled;
          }

          if (publicAccess !== undefined && publicAccess !== existingServer.publicAccess) {
            serverUpdateData.publicAccess = publicAccess;
          }

          if (anonymousAccess !== undefined && anonymousAccess !== existingServer.anonymousAccess) {
            serverUpdateData.anonymousAccess = anonymousAccess;
          }

          if (
            anonymousRateLimit !== undefined &&
            anonymousRateLimit !== existingServer.anonymousRateLimit
          ) {
            serverUpdateData.anonymousRateLimit = anonymousRateLimit;
          }

          // Only call updateServer if there are actual changes
          if (Object.keys(serverUpdateData).length > 0) {
            await updateServer(existingServer.serverId, serverUpdateData, body.common.userid);
          }

          const result = {
            toolId: toolId,
            // No isStartServer field for edit operation
          };

          return result;
        } catch (error) {
          console.error('Failed to update tool:', error);

          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

      case 3: // Enable server
        // Validate required toolId
        if (!toolId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'toolId',
            details: 'toolId is required for handleType=3 (Enable server)',
          });
        }

        try {
          // Check current server status
          const existingServerResult = await getServers(
            {
              serverId: toolId,
            },
            body.common.userid,
          );
          const existingServer = existingServerResult.servers[0];

          if (!existingServer) {
            throw new ApiError(ErrorCode.SERVER_NOT_FOUND, 404);
          }

          // If already enabled, no need to do anything
          if (existingServer.enabled === true) {
            return {
              toolId: toolId,
              isStartServer: 1, // Already running, consider it as success
            };
          }

          // Start the MCP server first before updating database
          let isStartServer = 2; // Default to failed
          try {
            // Get owner's access token if proxyId and masterPwd are provided
            if (proxyId && masterPwd) {
              const ownerResult = await getUsers(
                {
                  proxyId: Number(proxyId),
                  role: 1, // Owner role
                },
                body.common.userid,
              );
              const owner = ownerResult.users[0];

              if (owner && owner.encryptedToken) {
                const ownerToken = await CryptoUtils.decryptDataFromString(
                  owner.encryptedToken,
                  masterPwd,
                );

                if (ownerToken) {
                  await startMCPServer(existingServer.serverId, ownerToken);
                  isStartServer = 1; // Success
                } else {
                  throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
                }
              } else {
                throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, {
                  details: 'Owner user not found',
                });
              }
            } else {
              throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
                field: 'proxyId and masterPwd required for enabling server',
              });
            }
          } catch (error) {
            console.error('Failed to start MCP server:', error);
            isStartServer = 2; // Failed
            if (error instanceof ApiError) {
              throw error;
            }
            throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
              details: `Failed to start MCP server: ${error instanceof Error ? error.message : 'Unknown error'}`,
            });
          }

          return {
            toolId: toolId,
            isStartServer: isStartServer,
          };
        } catch (error) {
          console.error('Failed to enable server:', error);
          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

      case 4: // Disable server
        // Validate required toolId
        if (!toolId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'toolId',
            details: 'toolId is required for handleType=4 (Disable server)',
          });
        }

        // Validate required proxyId and masterPwd for security
        if (!proxyId || !masterPwd) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'proxyId and masterPwd',
            details:
              'proxyId and masterPwd are required for handleType=4 (Disable server) for security verification',
          });
        }

        try {
          // Verify master password is valid before proceeding
          try {
            const ownerResult = await getUsers(
              {
                proxyId: Number(proxyId),
                role: 1, // Owner role
              },
              body.common.userid,
            );
            const owner = ownerResult.users[0];

            if (!owner || !owner.encryptedToken) {
              throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, {
                details: 'Owner user not found or no encrypted token',
              });
            }

            // Attempt to decrypt owner token to verify master password
            const ownerToken = await CryptoUtils.decryptDataFromString(
              owner.encryptedToken,
              masterPwd,
            );

            if (!ownerToken) {
              throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401, {
                details: 'Invalid master password',
              });
            }

            // Master password is valid, proceed with disabling server
          } catch (error) {
            if (error instanceof ApiError) {
              throw error;
            }
            console.error('Failed to verify master password:', error);
            throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401, {
              details: 'Failed to verify master password',
            });
          }

          // Check current server status
          const existingServerResult = await getServers(
            {
              serverId: toolId,
            },
            body.common.userid,
          );
          const existingServer = existingServerResult.servers[0];

          if (!existingServer) {
            throw new ApiError(ErrorCode.SERVER_NOT_FOUND, 404);
          }

          // If already disabled, no need to do anything
          if (existingServer.enabled === false) {
            return {
              toolId: toolId,
              // No isStartServer field for disable operation
            };
          }

          // Stop the MCP server first before updating database
          try {
            await stopMCPServer(existingServer.serverId, body.common.userid);
          } catch (error) {
            console.error('Failed to stop MCP server:', error);
            throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
              details: `Failed to stop MCP server: ${error instanceof Error ? error.message : 'Unknown error'}`,
            });
          }

          return {
            toolId: toolId,
            // No isStartServer field for disable operation
          };
        } catch (error) {
          console.error('Failed to disable server:', error);
          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

      case 5: // Delete server
        // Validate required toolId
        if (!toolId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'toolId',
            details: 'toolId is required for handleType=5 (Delete server)',
          });
        }

        try {
          // Check if server exists
          const existingServerResult = await getServers(
            {
              serverId: toolId,
            },
            body.common.userid,
          );
          const existingServer = existingServerResult.servers[0];

          if (!existingServer) {
            throw new ApiError(ErrorCode.SERVER_NOT_FOUND, 404);
          }

          // Only delete from database if proxy server call succeeds
          await deleteServer(toolId, body.common.userid);

          const result = {
            toolId: toolId,
            // No isStartServer field for delete operation
          };

          return result;
        } catch (error) {
          console.error('Failed to delete server:', error);

          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

      case 6: // Start server
        // Validate required toolId
        if (!toolId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'toolId',
            details: 'toolId is required for handleType=6 (Start server)',
          });
        }

        // Validate required proxyId and masterPwd
        if (!proxyId || !masterPwd) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
            field: 'proxyId and masterPwd',
            details: 'proxyId and masterPwd are required for handleType=6 (Start server)',
          });
        }

        try {
          // Check if server exists
          const existingServerResult = await getServers(
            {
              serverId: toolId,
            },
            body.common.userid,
          );
          const existingServer = existingServerResult.servers[0];

          if (!existingServer) {
            throw new ApiError(ErrorCode.SERVER_NOT_FOUND, 404);
          }

          // Get owner's access token for starting server
          let isStartServer = 2; // Default to failed
          try {
            const ownerResult = await getUsers(
              {
                proxyId: proxyId,
                role: 1, // Owner role
              },
              body.common.userid,
            );
            const owner = ownerResult.users[0];

            if (!owner || !owner.encryptedToken) {
              throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, {
                details: 'Owner user not found or no encrypted token',
              });
            }

            // Decrypt owner token
            const ownerToken = await CryptoUtils.decryptDataFromString(
              owner.encryptedToken,
              masterPwd,
            );

            if (!ownerToken) {
              throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401);
            }

            // Start the MCP server
            await startMCPServer(existingServer.serverId, ownerToken);
            isStartServer = 1; // Success
          } catch (error) {
            console.error('Failed to start MCP server:', error);
            if (error instanceof ApiError) {
              throw error;
            }
            isStartServer = 2; // Failed
          }

          const result = {
            toolId: toolId,
            isStartServer: isStartServer,
          };

          return result;
        } catch (error) {
          console.error('Failed to start server:', error);

          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

      default:
        throw new ApiError(ErrorCode.INVALID_REQUEST);
    }
  } catch (error) {
    console.error('Protocol 10005 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
