import type { NextConfig } from "next";

// `output: 'standalone'` collapses the server + its transitive
// node_modules into `.next/standalone/`. It's great for a hand-rolled
// Dockerfile, but aspect_rules_js can't track the self-referential
// `.next/standalone/node_modules/next` symlink that Next.js plants
// during that pipeline (the link points back into the pnpm virtual
// store, which lives outside the Bazel execroot). So leave standalone
// off by default; the legacy Dockerfile flow can flip it back on via
// `BAZEL_BUILD=standalone`.
const enableStandalone = process.env.BAZEL_BUILD === "standalone";

const nextConfig: NextConfig = {
  ...(enableStandalone ? { output: "standalone" as const } : {}),

  // =============================================================================
  // Unified Domain Configuration
  // 将 PRIMARY_DOMAIN / USE_HTTPS 映射为 NEXT_PUBLIC_* 变量
  // 这样配置文件中可以统一使用 PRIMARY_DOMAIN，与 Backend/Relay 保持一致
  // =============================================================================
  env: {
    // 使用占位符，运行时由 entrypoint.mjs 替换为实际值
    // 构建时直接读 process.env 会被 Next.js 内联求值，导致占位符替换失效
    NEXT_PUBLIC_PRIMARY_DOMAIN:
      process.env.PRIMARY_DOMAIN || "__PRIMARY_DOMAIN__",
    NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS || "__USE_HTTPS__",
  },

  // Proxy API requests to the backend in development to avoid CORS issues
  async rewrites() {
    const primaryDomain = process.env.PRIMARY_DOMAIN;
    const useHttps = process.env.USE_HTTPS === "true";
    const protocol = useHttps ? "https" : "http";
    const backendUrl = primaryDomain
      ? `${protocol}://${primaryDomain}`
      : "http://localhost:10000";

    return [
      {
        source: "/api/:path*",
        destination: `${backendUrl}/api/:path*`,
      },
    ];
  },

  // Allow images from any source during development
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "**",
      },
      {
        protocol: "http",
        hostname: "**",
      },
    ],
  },
};

export default nextConfig;
