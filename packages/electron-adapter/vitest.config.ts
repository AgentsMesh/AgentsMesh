import { defineConfig } from "vitest/config";

// Pure-logic units: node env, no jsdom/wasm. Verifies the electron service
// provider derives the correct backend Connect (service, method) path for
// every facade — guarding the multi-service routing + name-override config.
export default defineConfig({
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
});
