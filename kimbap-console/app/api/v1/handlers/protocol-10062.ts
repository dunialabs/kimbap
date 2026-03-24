import { purgeCacheGlobal } from '@/lib/proxy-api';

interface Request10062 {
  common: {
    cmdId: number;
    userid: string;
  };
  params?: {
    reason?: string;
  };
}

interface Response10062Data {
  success: true;
}

export async function handleProtocol10062(body: Request10062): Promise<Response10062Data> {
  const userid = body.common?.userid;
  const reason = body.params?.reason || 'kimbap-console request';

  await purgeCacheGlobal(reason, userid);
  return { success: true };
}
