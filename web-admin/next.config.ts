import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
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
