import { ApiError, ErrorCode } from '@/lib/error-codes';
import { CryptoUtils } from '@/lib/crypto';
import { disableUser, countUsers, getUsers, createUser, updateUser, deleteUser, getProxy, getUserAvailableServersCapabilities } from '@/lib/proxy-api';
import { LicenseService } from '@/license-system';
import { prisma } from '@/lib/prisma';
import {
  upsertTokenMetadata,
  deleteTokenMetadata,
  validateMetadataInput,
  normalizeNamespace,
  normalizeTags,
  applyTagsOperation,
  getTokenMetadataMap,
  mergeParsedPermissions,
} from '@/lib/token-metadata';
import type { TagsMode, PermissionsMode } from '@/lib/token-metadata';

interface Tool {
  toolTmplId: string;
  toolType: number;
  name: string;
  description: string;
  tags: string[];
  authtags: string[];
  credentials: Array<{
    name: string;
    description: string;
    dataType: number;
    key: string;
    value: string;
    options: Array<{ key: string; value: string }>;
    selected: { key: string; value: string };
  }>;
  toolFuncs: Array<{
    funcName: string;
    enabled: boolean;
  }>;
  toolResources: Array<{
    uri: string;
    enabled: boolean;
  }>;
  lastUsed: number;
  enabled: boolean;
  toolId: string;
}

interface Request10008 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    handleType: number;
    userid?: string; // Changed from tokenId to match proto
    userids?: string[];
    name?: string;
    role?: number; // 1-owner, 2-admin, 3-member (updated from proto)
    expireAt?: number;
    rateLimit?: number;
    notes?: string;
    permissions?: Tool[];
    permissionsMode?: string;
    tagsMode?: string;
    namespace?: string;
    tags?: string[];
    masterPwd?: string; // Added from proto
    proxyId?: number; // Added for finding owner
  };
}

interface Response10008Data {
  accessToken: string; // Updated to match proto file
  updatedCount?: number;
  failedCount?: number;
  failures?: Array<{ userid: string; error: string }>;
}

/**
 * Parse permissions from Tool array to the required structure
 * @param permissions - Array of Tool objects with permission details
 * @returns Parsed permissions object with server_id as keys
 */
function parsePermissions(permissions?: Tool[]): Record<string, any> {
  if (!permissions || permissions.length === 0) {
    return {};
  }
  
  const parsedPermissions: Record<string, any> = {};
  
  for (const tool of permissions) {
    // Use toolId as the server_id key
    const serverId = tool.toolId;
    
    // Parse tools (functions) into the required structure
    const tools: Record<string, { enabled: boolean }> = {};
    if (tool.toolFuncs && tool.toolFuncs.length > 0) {
      for (const func of tool.toolFuncs) {
        tools[func.funcName] = {
          enabled: func.enabled
        };
      }
    }
    
    // Parse resources into the required structure
    const resources: Record<string, { enabled: boolean }> = {};
    if (tool.toolResources && tool.toolResources.length > 0) {
      for (const resource of tool.toolResources) {
        resources[resource.uri] = {
          enabled: resource.enabled
        };
      }
    }
    
    // Build the permission structure for this server
    parsedPermissions[serverId] = {
      enabled: tool.enabled,
      tools: tools,
      resources: resources,
      prompts: {} // Empty prompts as requested
    };
  }
  
  return parsedPermissions;
}

export async function handleProtocol10008(body: Request10008): Promise<Response10008Data> {
  const { handleType, userid, name, role, expireAt, rateLimit, notes, permissions, namespace, tags, masterPwd, proxyId } = body.params || {};
  
  
  // Validate handleType
  if (!handleType || handleType < 1 || handleType > 4) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
  }
  
  try {
    switch (handleType) {
      case 1: // Add access token
        // Validate required fields for add operation
        if (!name) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'name' });
        }
        if (!role) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'role' });
        }
        if (!masterPwd) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'masterPwd' });
        }
        if (!proxyId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'proxyId' });
        }

        if (namespace !== undefined || tags !== undefined) {
          const metaErr = validateMetadataInput({ namespace, tags });
          if (metaErr) {
            throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'metadata', message: metaErr });
          }
        }

        // Check access token creation limit before proceeding
        try {
          // Get current access token count for this proxy
          const currentTokenCount = await countUsers(
            {
              excludeRole: 1 // Exclude owner role, only count access tokens
            },
            body.common.userid
          );
          
          // Check license limits
          const licenseService = LicenseService.getInstance();
          const limitCheck = await licenseService.checkAccessTokenLimit(currentTokenCount.count);
          
          if (!limitCheck.allowed) {
            throw new ApiError(ErrorCode.ACCESS_TOKEN_LIMIT_EXCEEDED);
          }
        } catch (error) {
          
          if (error instanceof ApiError) {
            throw error;
          }
          console.error('Failed to check access token limit:', error);
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }
        
        try {
          // 1. Find the owner of the proxy server
          const { users: owners } = await getUsers({
            proxyId: proxyId,
            role: 1 // Owner role
          }, body.common.userid);
          const owner = owners.length > 0 ? owners[0] : null;
          
          if (!owner || !owner.encryptedToken) {
            throw new ApiError(ErrorCode.INVALID_REQUEST);
          }
          
          // 2. Decrypt the owner's token using master password
          let ownerToken: string;
          try {
            ownerToken = await CryptoUtils.decryptDataFromString(
              owner.encryptedToken,
              masterPwd
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
          
          // 3. Generate access token
          const accessToken = CryptoUtils.generateToken();
          
          // 4. Generate user ID from access token
          const userId = await CryptoUtils.calculateUserId(accessToken);
          
          // 5. Encrypt access token with owner's token (not master password)
          const encryptedToken = await CryptoUtils.encryptData(accessToken, ownerToken);
          
          // 6. Parse permissions into the required structure
          const parsedPermissions = parsePermissions(permissions);
          
          // 7. Create user record via proxy API
          await createUser({
            userId: userId,
            status: 1, // 1-running (default active status)
            role: role, // 1-owner, 2-admin, 3-member
            permissions: JSON.stringify(parsedPermissions), // Store parsed permissions
            serverApiKeys: JSON.stringify({}), // Default empty object
            expiresAt: expireAt || 0, // Unix timestamp (0 means no expiration)
            ratelimit: rateLimit || 10, // Fixed: use lowercase 'ratelimit' to match schema
            name: name,
            encryptedToken: JSON.stringify(encryptedToken), // Store encrypted token as JSON string
            proxyId: proxyId, // Use the provided proxy ID
            notes: notes ?? undefined
          }, ownerToken);

          // Get proxy information to get the proxyKey
          const proxyInfo = await getProxy();
          const currentProxyKey = proxyInfo?.proxyKey || '';
          
          // Save token and userid to local user table
          try {
            await prisma.user.create({
              data: {
                userid: userId,
                accessToken: accessToken, // Save plain text token
                proxyKey: currentProxyKey, // Save the proxy key
                role: role // Save the role (2-admin, 3-member)
              }
            });
            console.log('Saved user to local table:', { userid: userId, proxyKey: currentProxyKey, role: role });
          } catch (error) {
            console.error('Failed to save user to local table:', error);
            // Continue execution even if local save fails
          }

          if (namespace !== undefined || tags !== undefined) {
            const metadataInput = {
              ...(namespace !== undefined ? { namespace: normalizeNamespace(namespace) } : {}),
              ...(tags !== undefined ? { tags: normalizeTags(tags) } : {})
            };
            await upsertTokenMetadata(proxyId, userId, metadataInput);
          }
          
          const result = {
            accessToken: accessToken // Return the generated access token
          };


          return result;
        } catch (error) {
          console.error('Failed to create access token:', error);
          
          
          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }
        
      case 2: // Edit access token
        // Validate required userid
        if (!userid) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'userid' });
        }

        if (namespace !== undefined || tags !== undefined) {
          const metaErr = validateMetadataInput({ namespace, tags });
          if (metaErr) {
            throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'metadata', message: metaErr });
          }
        }
        
        try {
          // Find the existing user
          const { users } = await getUsers({ userId: userid }, body.common.userid);
          const existingUser = users.length > 0 ? users[0] : null;
          
          if (!existingUser) {
            throw new ApiError(ErrorCode.USER_NOT_FOUND, 404);
          }
          
          // Prepare update data - only update name, notes, and permissions
          const updateData: any = {};
          
          // Update name if provided
          if (name !== undefined) {
            updateData.name = name;
          }
          
          // Update notes if provided
          if (notes !== undefined) {
            updateData.notes = notes;
          }
          
          // Update permissions if provided (full replacement)
          if (permissions !== undefined && permissions !== null) {
            const parsedPermissions = parsePermissions(permissions);
            updateData.permissions = JSON.stringify(parsedPermissions);
          }
          
          // Only update if there's something to update
          if (Object.keys(updateData).length > 0) {
            await updateUser(userid, updateData, body.common.userid);
          }

          if (namespace !== undefined || tags !== undefined) {
            if (!proxyId) {
              throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'proxyId' });
            }
            const metadataInput = {
              ...(namespace !== undefined ? { namespace: normalizeNamespace(namespace) } : {}),
              ...(tags !== undefined ? { tags: normalizeTags(tags) } : {})
            };
            await upsertTokenMetadata(proxyId, userid, metadataInput);
          }
          
          
          // Return the user's access token
          // Since we don't store the plain access token, we need to return something
          // In a real implementation, this might need to be handled differently
          const result = {
            accessToken: userid // Return userid as placeholder since we can't retrieve the original token
          };


          return result;
        } catch (error) {
          console.error('Failed to edit access token:', error);
          
          
          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }
        
      case 3: // Delete access token
        // Validate required userid
        if (!userid) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'userid' });
        }
        
        try {
          // Check if user exists
          const { users } = await getUsers({ userId: userid }, body.common.userid);
          const existingUser = users.length > 0 ? users[0] : null;
          
          if (!existingUser) {
            throw new ApiError(ErrorCode.USER_NOT_FOUND, 404);
          }
          
          // Delete the user via proxy API
          await deleteUser(userid, body.common.userid);
          
          // Call proxy API to disable the user
          try {
            await disableUser(userid, body.common.userid);
          } catch (error) {
            console.error('Failed to disable user in proxy:', error);
            // Continue even if proxy disable fails since user is already deleted from DB
          }

          try {
            let metaProxyId = proxyId;
            if (!metaProxyId) {
              const proxyInfo = await getProxy();
              metaProxyId = proxyInfo?.id;
            }
            if (metaProxyId) {
              await deleteTokenMetadata(metaProxyId, userid);
            }
          } catch (error) {
            console.error('Failed to delete token metadata:', error);
          }
          try {
            await prisma.user.deleteMany({ where: { userid } });
          } catch (error) {
            console.error('Failed to delete user from local table:', error);
          }
          
          // Return the deleted userid as confirmation
          const result = {
            accessToken: userid
          };


          return result;
        } catch (error) {
          console.error('Failed to delete access token:', error);
          
          
          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
        }

      case 4: {
        const { userids: targetUserids, permissionsMode, tagsMode, namespace: bulkNamespace, tags: bulkTags } = body.params;

        const resolvedPermissionsMode: PermissionsMode = (permissionsMode as PermissionsMode) || 'replace';
        const resolvedTagsMode: TagsMode = (tagsMode as TagsMode) || 'replace';

        if (!targetUserids || !Array.isArray(targetUserids) || targetUserids.length === 0) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'userids' });
        }
        if (!proxyId) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'proxyId' });
        }

        if (permissionsMode && permissionsMode !== 'replace' && permissionsMode !== 'merge') {
          throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'permissionsMode' });
        }
        if (tagsMode && tagsMode !== 'replace' && tagsMode !== 'add' && tagsMode !== 'remove' && tagsMode !== 'clear') {
          throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'tagsMode' });
        }
        if (bulkTags !== undefined && !Array.isArray(bulkTags)) {
          throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'tags' });
        }
        if (tagsMode && tagsMode !== 'clear' && bulkTags === undefined) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'tags' });
        }

        if (bulkNamespace !== undefined || bulkTags !== undefined) {
          const metadataValidationError = validateMetadataInput({
            ...(bulkNamespace !== undefined ? { namespace: normalizeNamespace(bulkNamespace) } : {}),
            ...(bulkTags !== undefined ? { tags: normalizeTags(bulkTags) } : {})
          });
          if (metadataValidationError) {
            throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'metadata', message: metadataValidationError });
          }
        }

        let updatedCount = 0;
        let failedCount = 0;
        const failures: Array<{ userid: string; error: string }> = [];

        const { users: proxyUsers } = await getUsers({ proxyId }, body.common.userid);
        const userMap = new Map(proxyUsers.map((u) => [u.userId, u]));
        const parsedPermissions = permissions !== undefined ? parsePermissions(permissions) : null;
        const normalizedBulkTags = bulkTags !== undefined ? normalizeTags(bulkTags) : undefined;
        const metadataMap =
          resolvedTagsMode !== 'replace' && resolvedTagsMode !== 'clear' && bulkTags !== undefined
            ? await getTokenMetadataMap(proxyId, targetUserids)
            : null;

        for (const targetUserId of targetUserids) {
          try {
            const targetUser = userMap.get(targetUserId) || null;

            if (!targetUser) {
              failedCount++;
              failures.push({ userid: targetUserId, error: 'User not found' });
              continue;
            }

            if (targetUser.role === 1) {
              failedCount++;
              failures.push({ userid: targetUserId, error: 'Cannot bulk update owner token' });
              continue;
            }

            if (permissions !== undefined) {
              if (resolvedPermissionsMode === 'replace') {
                await updateUser(targetUserId, { permissions: JSON.stringify(parsedPermissions || {}) }, body.common.userid);
              } else if (resolvedPermissionsMode === 'merge') {
                const existingCaps = await getUserAvailableServersCapabilities(targetUserId, body.common.userid);
                const mergedPerms = mergeParsedPermissions(existingCaps, parsedPermissions || {});
                await updateUser(targetUserId, { permissions: JSON.stringify(mergedPerms) }, body.common.userid);
              }
            }

            if (bulkNamespace !== undefined || bulkTags !== undefined || resolvedTagsMode === 'clear') {
              const currentMeta = metadataMap?.get(targetUserId) || { namespace: 'default', tags: [] };
              const updateMeta: { namespace?: string; tags?: string[] } = {};

              if (bulkNamespace !== undefined && bulkNamespace !== '') {
                updateMeta.namespace = normalizeNamespace(bulkNamespace);
              }

              if (resolvedTagsMode !== 'clear' && normalizedBulkTags) {
                updateMeta.tags = applyTagsOperation(
                  currentMeta.tags,
                  normalizedBulkTags,
                  resolvedTagsMode
                );
              } else if (resolvedTagsMode === 'clear') {
                updateMeta.tags = [];
              }

              if (Object.keys(updateMeta).length > 0) {
                await upsertTokenMetadata(proxyId, targetUserId, updateMeta);
              }
            }

            updatedCount++;
          } catch (error) {
            failedCount++;
            failures.push({ userid: targetUserId, error: error instanceof Error ? error.message : 'Unknown error' });
          }
        }

        return {
          accessToken: 'bulk',
          updatedCount,
          failedCount,
          failures
        };
      }
        
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
    }
    
  } catch (error) {
    console.error('Protocol 10008 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
