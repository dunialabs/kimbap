/** @type {import('next').NextConfig} */
const nextConfig = {
  // 独立部署配置 - 支持API routes和SSR
  output: 'standalone',

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
    optimizePackageImports: ['@radix-ui/react-*', 'lucide-react']
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