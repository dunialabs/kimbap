import { prisma } from '@/lib/prisma';
import { mapConnectedClientRow } from './protocol-10023.helpers';

export async function getConnectedClientsList(
  usersMap: Map<string, string>,
  proxyKey: string,
): Promise<Array<{
  id: string;
  name: string;
  token: string;
  ip: string;
  location: string;
  lastActive: string;
  requests: number;
}>> {
  if (!proxyKey) {
    return [];
  }

  try {
    const last24Hours = BigInt(Math.floor((Date.now() - 24 * 60 * 60 * 1000) / 1000));

    const groupedClients = await prisma.log.groupBy({
      by: ['userid', 'ip'],
      where: {
        proxyKey,
        addtime: { gte: last24Hours },
        userid: { not: '' },
      },
      _max: { addtime: true },
      _count: { id: true },
      orderBy: [
        {
          _max: {
            addtime: 'desc',
          },
        },
        { userid: 'asc' },
        { ip: 'asc' },
      ],
      take: 20,
    });

    const connectedClientsList: Array<{
      id: string;
      name: string;
      token: string;
      ip: string;
      location: string;
      lastActive: string;
      requests: number;
    }> = [];

    for (const client of groupedClients) {
      const mappedClient = mapConnectedClientRow(client, usersMap);
      if (!mappedClient) {
        continue;
      }
      connectedClientsList.push(mappedClient);
    }

    return connectedClientsList;
  } catch (error) {
    console.error('Failed to get connected clients list:', error);
    return [];
  }
}
