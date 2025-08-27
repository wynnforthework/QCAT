/** @type {import('next').NextConfig} */
const nextConfig = {
  // 启用严格模式以帮助发现潜在问题
  reactStrictMode: true,

  // 配置实验性功能 (如果需要的话)
  experimental: {
    // 可以在这里添加实验性功能
  },

  // Turbopack 配置已移除 - 使用默认配置
  
  // 配置环境变量
  env: {
    CUSTOM_KEY: 'my-value',
  },
  
  // 配置重定向 - 整合重复的策略页面
  async redirects() {
    return [
      // 策略管理页面重定向到统一页面
      {
        source: '/strategies',
        destination: '/strategies-unified?view=management',
        permanent: true,
      },
      // 策略池管理重定向
      {
        source: '/strategy-pool',
        destination: '/strategies-unified?view=pool',
        permanent: true,
      },
      // 策略工作流重定向
      {
        source: '/strategy-workflow',
        destination: '/strategies-unified?view=workflow',
        permanent: true,
      },
      // 双闭环概览重定向
      {
        source: '/dual-loop-overview',
        destination: '/strategies-unified?view=overview',
        permanent: true,
      },
      // 交易执行监控重定向
      {
        source: '/trading-execution',
        destination: '/strategies-unified?view=execution',
        permanent: true,
      },
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
