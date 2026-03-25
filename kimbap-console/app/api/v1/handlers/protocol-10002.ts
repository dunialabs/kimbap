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

export async function handleProtocol10002(_body: Request10002): Promise<Response10002Data> {
  // Get the proxy record (there should only be one per system)
  const proxy = await getProxy();

  return {
    proxyId: proxy.id,
    proxyKey: proxy.proxyKey || '',
    proxyName: proxy.name,
    status: 1,
    createdAt: proxy.addtime,
    fingerprint: '',
  };
}