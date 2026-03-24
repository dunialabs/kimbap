import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getUserAvailableServersCapabilities, getServers, getUsers } from '@/lib/proxy-api';
import { getTokenMetadataMap } from '@/lib/token-metadata';

interface Request10007 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    proxyId: number; // Filter users by proxy ID
  };
}

interface Tool {
  name: string;
  description: string;
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

interface AccessToken {
  tokenId: string;
  name: string;
  role: number; // 1-owner, 2-admin, 3-member
  notes: string;
  lastUsed: number;
  createAt: number;
  expireAt: number;
  rateLimit: number;
  toolList: Tool[];
  namespace: string;
  tags: string[];
}

interface Response10007Data {
  tokenList: AccessToken[]; // Array of access tokens matching proto
}

async function getUserCapabilities(
  userId: string,
  requestUserId: string | undefined,
  serverNameMap: { [serverId: string]: string }
): Promise<Tool[]> {
  try {
    // Get user capabilities from proxy API  
    const userCapabilities = await getUserAvailableServersCapabilities(userId, requestUserId);
    
    const toolList: Tool[] = [];
    
    // Process each server capability returned from proxy API
    for (const [serverId, serverCapabilities] of Object.entries(userCapabilities)) {
      // Only include servers that belong to this proxyId
      if (!serverNameMap[serverId]) {
        continue;
      }
      
      // Extract tool functions
      const toolFuncs: Array<{ funcName: string; enabled: boolean }> = [];
      if (serverCapabilities.tools && typeof serverCapabilities.tools === 'object') {
        for (const [funcName, config] of Object.entries(serverCapabilities.tools)) {
          toolFuncs.push({
            funcName,
            enabled: config.enabled || false
          });
        }
      }
      
      // Extract tool resources
      const toolResources: Array<{ uri: string; enabled: boolean }> = [];
      if (serverCapabilities.resources && typeof serverCapabilities.resources === 'object') {
        for (const [uri, config] of Object.entries(serverCapabilities.resources)) {
          toolResources.push({
            uri,
            enabled: config.enabled || false
          });
        }
      }
      
      // Create Tool object
      const tool: Tool = {
        name: serverNameMap[serverId],
        description: serverNameMap[serverId], // Use serverName as description
        toolFuncs,
        toolResources,
        lastUsed: 0, // TODO: Implement lastUsed tracking
        enabled: serverCapabilities.enabled || false,
        toolId: serverId
      };
      
      toolList.push(tool);
    }
    
    return toolList;
  } catch (error) {
    console.error('Failed to get user capabilities:', error);
    return [];
  }
}

export async function handleProtocol10007(body: Request10007): Promise<Response10007Data> {
  const { proxyId } = body.params || {};
  
  // Validate required proxyId
  if (!proxyId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'proxyId' });
  }
  
  try {
    const { servers } = await getServers({ proxyId }, body.common.userid);
    const serverNameMap: { [serverId: string]: string } = {};
    servers.forEach((server) => {
      serverNameMap[server.serverId] = server.serverName;
    });

    // Get all users, including owners, filtered by proxyId, ordered by createAt ascending
    const { users } = await getUsers({
      proxyId: proxyId
    }, body.common.userid);
    
    // Sort by createdAt asc to match original ordering
    users.sort((a, b) => (a.createdAt || 0) - (b.createdAt || 0));

    const userIds = users.map(user => user.userId);
    const metadataMap = await getTokenMetadataMap(proxyId, userIds);

    // Map users to response format
    const tokenList: AccessToken[] = await Promise.all(
      users.map(async (user) => {
        // Get user capabilities from proxy API
        const toolList = await getUserCapabilities(user.userId, body.common.userid, serverNameMap);
        const metadata = metadataMap.get(user.userId) || { namespace: 'default', tags: [] };
        
        return {
          tokenId: user.userId,
          name: user.name,
          role: user.role,
          notes: (user as any).notes || '', // Temporary cast while Prisma types catch up
          lastUsed: 0, // TODO: Implement lastUsed tracking
          createAt: user.createdAt || 0, // Already Unix timestamp
          expireAt: user.expiresAt || 0, // Already Unix timestamp
          rateLimit: user.ratelimit,
          toolList,
          namespace: metadata.namespace,
          tags: metadata.tags
        };
      })
    );
    
    return {
      tokenList: tokenList
    };
    
  } catch (error) {
    console.error('Protocol 10007 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
