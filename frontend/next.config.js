/** @type {import('next').NextConfig} */
const nextConfig = {
  // 启用严格模式以帮助发现潜在问题
  reactStrictMode: true,

  // 配置实验性功能 (如果需要的话)
  experimental: {
    // 可以在这里添加实验性功能
  },

  // Turbopack 配置 (用于开发模式)
  turbo: {
    // 可以在这里添加 Turbopack 特定的配置
  },
  
  // 配置环境变量
  env: {
    CUSTOM_KEY: 'my-value',
  },
  
  // 配置重定向
  async redirects() {
    return [
      // 可以在这里添加重定向规则
    ]
  },
  
  // 配置重写
  async rewrites() {
    // 从环境变量获取API URL，优先级：NEXT_PUBLIC_API_URL > API_URL > 默认值
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || process.env.API_URL || 'http://localhost:8082';

    return [
      // API 代理配置
      {
        source: '/api/:path*',
        destination: `${apiUrl}/:path*`,
      },
    ]
  },
  
  // 配置头部
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-XSS-Protection',
            value: '1; mode=block',
          },
        ],
      },
    ]
  },
  
  // 配置图片优化
  images: {
    domains: ['localhost'],
    formats: ['image/webp', 'image/avif'],
  },
  
  // 配置 webpack (仅在非 Turbopack 模式下使用)
  webpack: (config, { buildId, dev, isServer, defaultLoaders, webpack }) => {
    // 自定义 webpack 配置
    // 注意：当使用 Turbopack 时，这个配置不会被使用
    return config
  },
}

module.exports = nextConfig
