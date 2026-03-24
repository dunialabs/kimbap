import { prisma } from './prisma'

export const dbService = {
  users: {
    create: async (userid: string, accessToken: string, proxyKey: string, role: number) => {
      return await prisma.user.create({
        data: { userid, accessToken, proxyKey, role }
      })
    },

    findByUserId: async (userid: string) => {
      return await prisma.user.findUnique({
        where: { userid }
      })
    },

    findByAccessToken: async (accessToken: string) => {
      return await prisma.user.findFirst({
        where: { accessToken }
      })
    },

    list: async (limit = 100, offset = 0) => {
      return await prisma.user.findMany({
        take: limit,
        skip: offset
      })
    },

    count: async () => {
      return await prisma.user.count()
    },

    update: async (userid: string, data: { accessToken?: string; proxyKey?: string; role?: number }) => {
      return await prisma.user.update({
        where: { userid },
        data
      })
    },

    delete: async (userid: string) => {
      return await prisma.user.delete({
        where: { userid }
      })
    }
  },

  logs: {
    create: async (data: {
      action: number;
      userid?: string;
      sessionId?: string;
      upstreamRequestId?: string;
      uniformRequestId?: string;
      ip?: string;
      ua?: string;
      eventType?: number;
      tokenMask?: string;
      requestParams?: string;
      responseResult?: string;
      error?: string;
    }) => {
      return await prisma.log.create({
        data: {
          addtime: BigInt(Math.floor(Date.now() / 1000)),
          action: data.action,
          userid: data.userid || '',
          sessionId: data.sessionId || '',
          upstreamRequestId: data.upstreamRequestId || '',
          uniformRequestId: data.uniformRequestId || '',
          ip: data.ip || '',
          ua: data.ua || '',
          tokenMask: data.tokenMask || '',
          requestParams: data.requestParams || '',
          responseResult: data.responseResult || '',
          error: data.error || ''
        }
      })
    },

    findById: async (id: number) => {
      return await prisma.log.findUnique({
        where: { id }
      })
    },

    list: async (limit = 100, offset = 0, eventType?: number) => {
      return await prisma.log.findMany({
        where: eventType !== undefined ? { eventType } : undefined,
        take: limit,
        skip: offset,
        orderBy: { addtime: 'desc' }
      })
    },

    count: async (eventType?: number) => {
      return await prisma.log.count({
        where: eventType !== undefined ? { eventType } : undefined
      })
    }
  },

  apiRequests: {
    create: async (data: {
      method: string
      url: string
      statusCode?: number
      responseTime?: number
      userId?: number
    }) => {
      return await prisma.apiRequest.create({ data })
    },

    list: async (limit = 100, offset = 0) => {
      return await prisma.apiRequest.findMany({
        take: limit,
        skip: offset,
        orderBy: { createdAt: 'desc' },
        include: { user: true }
      })
    },

    getStats: async () => {
      const [totalRequests, avgResponseTime, requestsByMethod] = await Promise.all([
        prisma.apiRequest.count(),
        prisma.apiRequest.aggregate({
          _avg: { responseTime: true }
        }),
        prisma.apiRequest.groupBy({
          by: ['method'],
          _count: true
        })
      ])

      return {
        totalRequests,
        avgResponseTime: avgResponseTime._avg.responseTime || 0,
        requestsByMethod: requestsByMethod.map(item => ({
          method: item.method,
          count: item._count
        }))
      }
    }
  }
}