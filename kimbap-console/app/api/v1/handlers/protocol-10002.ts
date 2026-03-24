import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';

interface Request10002 {
  common: {
    cmdId: number;
  };
  params: {};
}

interface Response10002Data {
  proxyId: number;
  proxyKey: string;
  proxyName: string;
  status: number; // 1-running, 2-stopped
  createdAt: number; // Unix timestamp
  fingerprint: string; // Hardware fingerprint
}

export async function handleProtocol10002(body: Request10002): Promise<Response10002Data> {
  // Get the proxy record (there should only be one per system)
  const proxy = await getProxy();

  if (!proxy) {
    throw new ApiError(ErrorCode.RECORD_NOT_FOUND, 404);
  }

  // TODO: Get the actual MCP server running status
  // This should check if the MCP server process is running
  // The command/method to check status is not yet defined
  // Example: checkMCPServerStatus() or similar
  let serverStatus = 1; // 1-running (hardcoded for now)

  // TODO: Implement server status check logic here
  // try {
  //   const isRunning = await checkMCPServerStatus(proxy.id);
  //   serverStatus = isRunning ? 1 : 2; // 1-running, 2-stopped
  // } catch (error) {
  //   console.error('Failed to check MCP server status:', error);
  //   serverStatus = 2; // assume stopped on error
  // }

  const fingerprint = '';

  // Return proxy server info
  return {
    proxyId: proxy.id,
    proxyKey: proxy.proxyKey || '', // Return the proxyKey
    proxyName: proxy.name,
    status: serverStatus, // Server running status
    createdAt: proxy.addtime, // Unix timestamp
    fingerprint: fingerprint // Hardware fingerprint
  };
}