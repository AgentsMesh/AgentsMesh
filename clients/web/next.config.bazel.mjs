import createNextIntlPlugin from "next-intl/plugin";
import path from "node:path";
import { fileURLToPath } from "node:url";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

const here = path.dirname(fileURLToPath(import.meta.url));
const monorepoRoot = path.resolve(here, "../..");

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",

  // pnpm + monorepo: NFT must walk the virtual store at the repo root,
  // otherwise transitively-imported deps disappear from .next/standalone/.
  // See vercel/next.js#33895, #48017.
  outputFileTracingRoot: monorepoRoot,

  outputFileTracingIncludes: {
    "/blog/[slug]": ["./src/content/blog/**/*.md"],
    "/blog": ["./src/content/blog/**/*.md"],
  },

  typescript: { ignoreBuildErrors: true },

  transpilePackages: [
    "@agentsmesh/service-runtime",
    "@agentsmesh/service-interface",
  ],

  webpack: (config, { isServer }) => {
    config.experiments = { ...config.experiments, asyncWebAssembly: true };
    config.output.webassemblyModuleFilename = isServer
      ? "./../static/wasm/[modulehash].wasm"
      : "static/wasm/[modulehash].wasm";
    return config;
  },

  allowedDevOrigins: process.env.ALLOWED_DEV_ORIGINS
    ? process.env.ALLOWED_DEV_ORIGINS.split(",")
    : [],

  turbopack: { root: monorepoRoot },

  env: {
    NEXT_PUBLIC_PRIMARY_DOMAIN:
      process.env.PRIMARY_DOMAIN || "__PRIMARY_DOMAIN__",
    NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS || "__USE_HTTPS__",
    NEXT_PUBLIC_POSTHOG_KEY: process.env.POSTHOG_KEY || "__POSTHOG_KEY__",
    NEXT_PUBLIC_POSTHOG_HOST: process.env.POSTHOG_HOST || "__POSTHOG_HOST__",
  },

  async rewrites() {
    const proxyTarget = process.env.API_PROXY_TARGET;
    if (process.env.NODE_ENV === "development" && proxyTarget) {
      console.log(`[Next.js] API proxy enabled: /api/* -> ${proxyTarget}/api/*`);
      return [
        { source: "/api/:path*", destination: `${proxyTarget}/api/:path*` },
        { source: "/health", destination: `${proxyTarget}/health` },
      ];
    }
    return [];
  },
};

export default withNextIntl(nextConfig);
