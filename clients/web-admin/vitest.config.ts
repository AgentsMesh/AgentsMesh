import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { resolve } from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "jsdom",
    // `threads` (worker_threads) shares the Node module cache across
    // parallel test files. Default `forks` spawns one process per file,
    // and each fresh process re-walks pnpm's virtual store, picking up
    // a different `node_modules/.aspect_rules_js/.../node_modules/react`
    // copy for every peer-dep that pulls react in — Bazel sandbox
    // amplifies this and breaks vi.mock identity for `@/...` aliases
    // (mocked module and the import inside the SUT resolve to different
    // pnpm replicas). Threads share the resolved module instance, so
    // both alias and relative import end up at the same identity.
    pool: "threads",
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      reportsDirectory: "./coverage",
      exclude: [
        "node_modules/",
        "src/test/",
        "**/*.d.ts",
        "**/*.config.*",
        "**/types/**",
      ],
    },
  },
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src"),
      "@proto": resolve(__dirname, "../../proto/gen/ts"),
    },
    // Force a single copy of React across the workspace — same reason
    // documented in clients/web/vitest.config.ts: pnpm's virtual store
    // hands out per-peer-dep replicas otherwise.
    dedupe: ["react", "react-dom"],
  },
});
