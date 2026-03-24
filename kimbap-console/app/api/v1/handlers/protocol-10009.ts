import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getAvailableServersCapabilities, getServers, makeProxyRequestWithUserId } from '@/lib/proxy-api';

interface Request10009 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    proxyId: number;
  };
}

interface ToolFunction {
  funcName: string;
  enabled: boolean;
}

interface ToolResource {
  uri: string;
  enabled: boolean;
}

interface Tool {
  toolId: string;
  name: string;
  description: string;
  enabled: boolean;
  toolFuncs: ToolFunction[];
  toolResources: ToolResource[];
}

interface Response10009Data {
  scopes: Tool[];
}

export async function handleProtocol10009(body: Request10009): Promise<Response10009Data> {
  const { proxyId } = body.params || {};
  
  console.log('[Protocol 10009] Request received with proxyId:', proxyId);
  
  // Validate required proxyId
  if (!proxyId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'proxyId' });
  }
  
  try {
    console.log('[Protocol 10009] Getting available servers capabilities...');
    // Get all available servers capabilities from proxy API
    const allCapabilities = await getAvailableServersCapabilities(body.common.userid);
    console.log('[Protocol 10009] All capabilities:', JSON.stringify(allCapabilities, null, 2));
    
    // Map capabilities to Tool format
    const scopes: Tool[] = [];
    console.log('[Protocol 10009] Processing capabilities for scopes...');
    
    for (const [serverId, serverCapabilities] of Object.entries(allCapabilities)) {
      console.log(`[Protocol 10009] Processing server ${serverId}:`, {
        serverName: serverCapabilities.serverName,
        enabled: serverCapabilities.enabled,
        toolsCount: serverCapabilities.tools ? Object.keys(serverCapabilities.tools).length : 0,
        resourcesCount: serverCapabilities.resources ? Object.keys(serverCapabilities.resources).length : 0
      });
      
      let toolFuncs: ToolFunction[] = [];
      let toolResources: ToolResource[] = [];
      
      // Extract enabled tool functions from capabilities.tools
      if (serverCapabilities.tools && typeof serverCapabilities.tools === 'object') {
        toolFuncs = Object.entries(serverCapabilities.tools)
          .filter(([, config]) => config && config.enabled === true)
          .map(([funcName]) => ({
            funcName,
            enabled: true
          }));
      }
      
      // Extract enabled tool resources from capabilities.resources
      if (serverCapabilities.resources && typeof serverCapabilities.resources === 'object') {
        toolResources = Object.entries(serverCapabilities.resources)
          .filter(([, config]) => config && config.enabled === true)
          .map(([uri]) => ({
            uri,
            enabled: true
          }));
      }
      
      scopes.push({
        toolId: serverId, // Add serverId as toolId
        name: serverCapabilities.serverName,
        description: serverCapabilities.serverName, // Use serverName as description
        enabled: serverCapabilities.enabled,
        toolFuncs,
        toolResources
      });
    }
    
    console.log('[Protocol 10009] Final scopes result:', {
      scopesCount: scopes.length,
      scopes: scopes.map(s => ({
        toolId: s.toolId,
        name: s.name,
        toolFuncsCount: s.toolFuncs.length,
        toolResourcesCount: s.toolResources.length
      }))
    });
    
    return {
      scopes
    };
    
  } catch (error) {
    console.error('Protocol 10009 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}