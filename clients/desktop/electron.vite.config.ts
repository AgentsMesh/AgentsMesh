import { resolve } from "path";
import { defineConfig, externalizeDepsPlugin } from "electron-vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const desktopSrc = resolve(__dirname, "src/renderer");
const webSrc = resolve(__dirname, "../web/src");
// Generated TS proto message classes — match web/vitest-config alias
// (`@proto/*` → `../../proto/gen/ts/*`). Without this the renderer
// can't resolve the Connect-RPC pb modules cross-imported via
// clients/web/src/lib/api/.
const protoGen = resolve(__dirname, "../../proto/gen/ts");
// All deps live in the workspace root after Phase C. The thin-shell
// `clients/desktop/package.json` declares no `dependencies`, so
// `clients/desktop/node_modules/` is empty in CI; pnpm's hoisting puts
// React + everything else at the workspace root. Aliases below point
// at the root tree so vite finds `react` regardless of which CWD it
// was invoked from.
const rootModules = resolve(__dirname, "../../node_modules");

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
      // Force a single instance of these packages across the whole bundle.
      // Without this, vite resolves `react-router-dom` separately for each
      // import site (clients/desktop/src vs clients/web/src cross-imports),
      // producing two NavigationContext instances → `useNavigate may be
      // used only in the context of a <Router>` at runtime.
      //
      // `@agentsmesh/service-runtime` is critical: it holds module-scoped
      // `ready` + `i` state for getAuthManager() / markServiceReady().
      // Two instances = platform-init registers in A, RootRedirect reads
      // from B → B.ready=false → NOOP_PROXY returns `"[]"` for `_json`
      // getters → user/org parse as empty arrays → router lands on
      // `/undefined/workspace`.
      dedupe: [
        "react",
        "react-dom",
        "react-router-dom",
        "@tanstack/react-query",
        "@agentsmesh/service-runtime",
        "@agentsmesh/service-interface",
        "@agentsmesh/electron-adapter",
      ],
      alias: [
        { find: "react", replacement: resolve(rootModules, "react") },
        { find: "react-dom", replacement: resolve(rootModules, "react-dom") },
        { find: /^@\/lib\/wasm-core$/, replacement: resolve(desktopSrc, "shims/service-shim") },
        { find: /^@\/lib\/wasm-getters$/, replacement: resolve(desktopSrc, "shims/service-shim") },
        { find: /^@proto\/(.*)$/, replacement: resolve(protoGen, "$1") },
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
