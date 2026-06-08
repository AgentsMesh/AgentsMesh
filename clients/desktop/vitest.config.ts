import { defineConfig } from "vitest/config";

// Pure-logic units only (shared reducer + the latest-mac.yml rewrite script):
// node env, no jsdom/React/wasm. Renderer component tests would need the web
// alias + electronAPI mocks and are out of scope here.
export default defineConfig({
  test: {
    globals: true,
    environment: "node",
    include: ["src/**/*.test.ts", "scripts/**/*.test.ts"],
    reporters: ["default", "junit"],
    outputFile: { junit: "./report.xml" },
  },
});
