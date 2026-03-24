import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getServersCapabilities, makeProxyRequestWithUserId } from '@/lib/proxy-api';

interface Request10010 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    toolId: string; // Server ID
  };
}

interface ToolFunction {
  funcName: string;
  enabled: boolean;
  dangerLevel: number; // 0: No validation, 1: hint only, 2: validation required
  description: string; // Description for the function
}

interface ToolResource {
  uri: string;
  enabled: boolean;
}

interface Response10010Data {
  functions: ToolFunction[];
  resources: ToolResource[];
}

export async function handleProtocol10010(body: Request10010): Promise<Response10010Data> {
  const { toolId } = body.params || {};
  
  // Validate required toolId
  if (!toolId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'toolId' });
  }
  
  try {
    // Call proxy API to get server capabilities
    const capabilities = await getServersCapabilities(toolId, body.common.userid);
    
    // Transform capabilities to the response format
    const functions: ToolFunction[] = [];
    const resources: ToolResource[] = [];
    
    // Convert tools to functions array
    if (capabilities.tools) {
      for (const [funcName, config] of Object.entries(capabilities.tools)) {
        functions.push({
          funcName,
          enabled: config.enabled || false,
          dangerLevel: config.dangerLevel !== undefined ? config.dangerLevel : 0, // Default to 0 if not specified
          description: config.description || '' // Include description from capabilities
        });
      }
    }
    
    // Convert resources to resources array
    if (capabilities.resources) {
      for (const [uri, config] of Object.entries(capabilities.resources)) {
        resources.push({
          uri,
          enabled: config.enabled || false
        });
      }
    }
    
    // Note: We're not including prompts in the response as per the proto definition
    
    return {
      functions,
      resources
    };
    
  } catch (error) {
    console.error('Failed to get server capabilities:', error);
    
    // Re-throw the error to properly handle proxy configuration issues
    if (error instanceof Error) {
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { message: error.message });
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500);
  }
}