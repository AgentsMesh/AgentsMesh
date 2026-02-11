import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

const nextConfig: NextConfig = {
  output: "standalone",
  allowedDevOrigins: ["192.168.100.189"],

  // =============================================================================
  // Unified Domain Configuration
  // 将 PRIMARY_DOMAIN / USE_HTTPS 映射为 NEXT_PUBLIC_* 变量
  // 这样配置文件中可以统一使用 PRIMARY_DOMAIN，与 Backend/Relay 保持一致
  // =============================================================================
  env: {
    NEXT_PUBLIC_PRIMARY_DOMAIN: process.env.PRIMARY_DOMAIN || "",
    NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS || "false",
  },

  // 本地开发时代理 API 请求，避免跨域问题
  // API_PROXY_TARGET 由 dev.sh 生成到 .env.local
  // 前端使用相对路径 /api/*，Next.js rewrites 代理到后端
  async rewrites() {
    // API_PROXY_TARGET 是服务端变量（不带 NEXT_PUBLIC_ 前缀）
    const proxyTarget = process.env.API_PROXY_TARGET;

    // 仅在本地开发且配置了代理目标时启用
    if (process.env.NODE_ENV === "development" && proxyTarget) {
      console.log(`[Next.js] API proxy enabled: /api/* → ${proxyTarget}/api/*`);
      return [
        {
          source: "/api/:path*",
          destination: `${proxyTarget}/api/:path*`,
        },
        {
          source: "/health",
          destination: `${proxyTarget}/health`,
        },
      ];
    }

    return [];
  },
};

export default withNextIntl(nextConfig);
