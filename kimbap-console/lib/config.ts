export const config = {
  cloudApi: {
    baseUrl: process.env.KIMBAP_CLOUD_API_URL || 'https://kimbap-cloud.kimbap.sh',
    endpoints: {
      toolTemplates: '/tool/templates',
      checkProxyIP: '/api/check_proxy_ip',
      tunnelCreate: '/tunnel/create',
      tunnelCredentials: '/tunnel/credentials',
      tunnelDelete: '/tunnel/delete'
    }
  }
};