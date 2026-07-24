import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    include: ["src/main/relay_node_bridge.integration.test.ts"],
    testTimeout: 20_000,
  },
});
