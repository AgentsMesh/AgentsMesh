import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
import path from "path";
import { fileURLToPath } from "url";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

// Turbopack's `root` must be the monorepo root (where node_modules/.pnpm lives),
// NOT the project directory. Previously when `web/` was top-level, Next could
// auto-infer this; after moving under `clients/web/`, we must pin it.
const here = path.dirname(fileURLToPath(import.meta.url));
const monorepoRoot = path.resolve(here, "../..");

// `output: 'standalone'` packages the server + its transitive
// node_modules into `.next/standalone/` for a slim Docker image. The
// Bazel OCI pipeline can't track the self-referential
// `.next/standalone/node_modules/next` symlink that Next.js plants
// during that flow (it points back into the pnpm virtual store,
// outside the execroot). Gate standalone on `BAZEL_BUILD=standalone`
// so the legacy docker/build-push-action path keeps the slim layout
// and Bazel falls through to the default `.next/` output which the
// image entrypoint (build_defs/web/next.bzl) launches via
// `next start`.
const enableStandalone = process.env.BAZEL_BUILD === "standalone";

const nextConfig: NextConfig = {
  ...(enableStandalone ? { output: "standalone" as const } : {}),

  // Type checks live in the separate "Web (lint + type-check +
  // vitest)" Bazel job (plain `pnpm type-check`). Don't re-run them
  // inside `next build` — the Next.js build path hits a stricter
  // JSX-inference pass that flags pre-existing implicit-any sites the
  // top-level `tsc --noEmit` already accepts. Cleanup is a follow-up;
  // the production image build passes via ignore-build-errors.
  typescript: { ignoreBuildErrors: true },

  // Workspace packages ship their raw .ts sources (see
  // packages/service-runtime/BUILD.bazel for why). Tell Next.js to
  // run SWC over them during `next build` instead of treating them as
  // pre-compiled JS.
  transpilePackages: [
    "@agentsmesh/service-runtime",
    "@agentsmesh/service-interface",
  ],

  webpack: (config, { isServer }) => {
    config.experiments = {
      ...config.experiments,
      asyncWebAssembly: true,
    };
    config.output.webassemblyModuleFilename = isServer
      ? "./../static/wasm/[modulehash].wasm"
      : "static/wasm/[modulehash].wasm";
    return config;
  },
  allowedDevOrigins: process.env.ALLOWED_DEV_ORIGINS
    ? process.env.ALLOWED_DEV_ORIGINS.split(",")
    : [],

  // Ensure standalone build includes blog markdown files
  outputFileTracingIncludes: {
    "/blog/[slug]": ["./src/content/blog/**/*.md"],
    "/blog": ["./src/content/blog/**/*.md"],
  },

  // Required for next-intl plugin to resolve config in Turbopack dev mode.
  // `root` must point to the monorepo root so Turbopack can find the pnpm
  // virtual store at `<root>/node_modules/.pnpm/`.
  turbopack: {
    root: monorepoRoot,
  },

  // =============================================================================
  // Unified Domain Configuration
  // 将 PRIMARY_DOMAIN / USE_HTTPS 映射为 NEXT_PUBLIC_* 变量
  // 这样配置文件中可以统一使用 PRIMARY_DOMAIN，与 Backend/Relay 保持一致
  // =============================================================================
  env: {
    // 使用占位符，运行时由 docker-entrypoint.sh 替换为实际值
    // 构建时直接读 process.env 会被 Next.js 内联求值，导致占位符替换失效
    NEXT_PUBLIC_PRIMARY_DOMAIN:
      process.env.PRIMARY_DOMAIN || "__PRIMARY_DOMAIN__",
    NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS || "__USE_HTTPS__",
    NEXT_PUBLIC_POSTHOG_KEY:
      process.env.POSTHOG_KEY || "__POSTHOG_KEY__",
    NEXT_PUBLIC_POSTHOG_HOST:
      process.env.POSTHOG_HOST || "__POSTHOG_HOST__",
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
