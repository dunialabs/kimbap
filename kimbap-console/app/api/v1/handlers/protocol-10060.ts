import { getCacheHealth } from '@/lib/proxy-api';

interface Request10060 {
  common: {
    cmdId: number;
    userid: string;
  };
  params?: Record<string, never>;
}

interface Response10060Data {
  enabled: boolean;
  health: { ok: boolean; details?: string; backend: string };
  metrics?: Record<string, number>;
}

export async function handleProtocol10060(body: Request10060): Promise<Response10060Data> {
  const userid = body.common?.userid;
  return getCacheHealth(userid);
}
