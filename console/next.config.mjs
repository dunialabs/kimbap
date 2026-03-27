/** @type {import('next').NextConfig} */
const allowedOrigins = (() => {
  const origins = []

  // 添加 Cloudflare tunnel 域名（支持所有 *.p-mcp.com）
  origins.push('https://*.p-mcp.com')

  // 动态添加本地端口
  const frontendPort = process.env.FRONTEND_PORT || process.env.PORT || '3000'
  const backendPort = process.env.BACKEND_PORT || '3002'

  // 支持 localhost 和 127.0.0.1
  origins.push(`http://localhost:${frontendPort}`)
  origins.push(`http://localhost:${backendPort}`)
  origins.push(`http://127.0.0.1:${frontendPort}`)
  origins.push(`http://127.0.0.1:${backendPort}`)

  // 支持 HTTPS（如果启用）
  if (process.env.ENABLE_HTTPS === 'true') {
    origins.push(`https://localhost:${frontendPort}`)
    origins.push(`https://localhost:${backendPort}`)
    origins.push(`https://127.0.0.1:${frontendPort}`)
    origins.push(`https://127.0.0.1:${backendPort}`)
  }

  // 如果有自定义域名环境变量
  if (process.env.CUSTOM_DOMAIN) {
    origins.push(`https://${process.env.CUSTOM_DOMAIN}`)
    origins.push(`http://${process.env.CUSTOM_DOMAIN}`)
  }

  return origins
})()

const nextConfig = {
  // 独立部署配置 - 支持API routes和SSR
  output: 'standalone',

  // 允许的开发源（解决跨域警告）- Next.js 15 顶层配置
  allowedDevOrigins: allowedOrigins,

  // 图片配置
  images: {
    unoptimized: true,
    formats: ['image/webp', 'image/avif'],
    minimumCacheTTL: 60,
    dangerouslyAllowSVG: true,
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;"
  },

  // 实验性功能和性能优化
  experimental: {
    optimizeCss: false,
    optimizePackageImports: ['@radix-ui/react-*', 'lucide-react'],
    // Server Actions 的跨域白名单
    serverActions: {
      allowedOrigins: allowedOrigins
    }
  },

  // TypeScript 配置
  typescript: {
    ignoreBuildErrors: true
  },

  // ESLint 配置
  eslint: {
    ignoreDuringBuilds: false
  },

  // 启用压缩
  compress: true,

  // 启用严格模式
  reactStrictMode: true,

  // 移除 Next.js 标识
  poweredByHeader: false,

  // 安全头部配置
  headers: async () => [
    {
      source: '/(.*)',
      headers: [
        {
          key: 'Content-Security-Policy',
          value: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' http://localhost:* ws://localhost:*; object-src 'none'; base-uri 'self'; frame-ancestors 'none';"
        },
        {
          key: 'X-Frame-Options',
          value: 'DENY'
        },
        {
          key: 'X-Content-Type-Options',
          value: 'nosniff'
        },
        {
          key: 'Referrer-Policy',
          value: 'strict-origin-when-cross-origin'
        },
        {
          key: 'X-XSS-Protection',
          value: '1; mode=block'
        }
      ]
    }
  ],

  // Webpack 配置优化
  webpack: (config, { dev, isServer }) => {
    // Node.js 环境下的 fallback 配置
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        net: false,
        tls: false
      }
    }

    // 生产环境优化
    if (!dev && !isServer) {
      config.optimization.splitChunks = {
        chunks: 'all',
        cacheGroups: {
          default: false,
          vendor: {
            name: 'vendor',
            chunks: 'all',
            test: /node_modules/,
            priority: 10
          },
          common: {
            minChunks: 2,
            priority: -10,
            reuseExistingChunk: true
          }
        }
      }
    }

    return config
  }
}

export default nextConfig
