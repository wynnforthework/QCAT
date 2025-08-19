import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  eslint: {
    // 在构建时忽略 ESLint 错误，专注于功能测试
    ignoreDuringBuilds: true,
  },
  typescript: {
    // 在构建时忽略 TypeScript 错误，专注于功能测试
    ignoreBuildErrors: true,
  },
};

export default nextConfig;
