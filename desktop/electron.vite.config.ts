import { resolve } from "path";
import { defineConfig, externalizeDepsPlugin } from "electron-vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const desktopSrc = resolve(__dirname, "src/renderer");
const webSrc = resolve(__dirname, "../web/src");
const desktopModules = resolve(__dirname, "node_modules");

export default defineConfig({
  main: {
    plugins: [externalizeDepsPlugin()],
    resolve: {
      alias: { "@": resolve(__dirname, "src/main") },
    },
    build: {
      rollupOptions: {
        external: ["@agentsmesh/node-bridge"],
      },
    },
  },
  preload: {
    plugins: [externalizeDepsPlugin()],
    resolve: {
      alias: { "@": resolve(__dirname, "src/preload") },
    },
  },
  renderer: {
    plugins: [react(), tailwindcss()],
    define: {
      "process.env": JSON.stringify({}),
    },
    resolve: {
      alias: [
        { find: "react", replacement: resolve(desktopModules, "react") },
        { find: "react-dom", replacement: resolve(desktopModules, "react-dom") },
        { find: /^@\/lib\/wasm-core$/, replacement: resolve(desktopSrc, "shims/service-shim") },
        { find: /^@\/lib\/wasm-getters$/, replacement: resolve(desktopSrc, "shims/service-shim") },
        { find: "@/stores", replacement: resolve(webSrc, "stores") },
        { find: "@/hooks", replacement: resolve(webSrc, "hooks") },
        { find: "@/components", replacement: resolve(webSrc, "components") },
        // env.ts must resolve to the desktop-specific version so it picks up
        // the preload-exposed apiUrl instead of `window.location.origin`.
        { find: /^@\/lib\/env$/, replacement: resolve(desktopSrc, "lib/env") },
        { find: "@/lib", replacement: resolve(webSrc, "lib") },
        { find: "@/messages", replacement: resolve(webSrc, "messages") },
        // `@/app/...` is the Next.js app-router path; some shared components
        // (e.g. InfraRepositoryDetail) import ./components co-located with the
        // route file. Fall back to the web source tree so those imports resolve.
        { find: "@/app", replacement: resolve(webSrc, "app") },
        { find: "@/providers", replacement: resolve(webSrc, "providers") },
        { find: "@", replacement: desktopSrc },
        { find: "next/navigation", replacement: resolve(desktopSrc, "shims/next-navigation") },
        { find: "next/link", replacement: resolve(desktopSrc, "shims/next-link") },
        { find: "next-intl", replacement: resolve(desktopSrc, "shims/next-intl") },
        { find: "next-themes", replacement: resolve(desktopSrc, "shims/next-themes") },
        { find: "@tauri-apps/plugin-shell", replacement: resolve(desktopSrc, "shims/electron-shell") },
        { find: "@tauri-apps/api/core", replacement: resolve(desktopSrc, "shims/electron-ipc") },
        { find: "@tauri-apps/api", replacement: resolve(desktopSrc, "shims/electron-ipc") },
      ],
    },
  },
});
