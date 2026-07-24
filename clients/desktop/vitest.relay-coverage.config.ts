import { defineConfig } from "vitest/config";

import baseConfig from "./vitest.config";

export default defineConfig({
  ...baseConfig,
  test: {
    ...baseConfig.test,
    include: [
      "src/main/relay.test.ts",
      "src/main/relay_listener_events.test.ts",
      "src/main/relay_listener_wiring.test.ts",
      "src/main/relay_output_subscriptions.test.ts",
      "src/preload/relay_push_api.test.ts",
    ],
    coverage: {
      enabled: true,
      provider: "v8",
      reporter: ["text", "json-summary"],
      reportsDirectory: "./coverage-relay",
      include: [
        "src/main/relay.ts",
        "src/main/relay_listener_events.ts",
        "src/main/relay_listener_wiring.ts",
        "src/main/relay_output_subscriptions.ts",
        "src/preload/relay_push_api.ts",
      ],
      thresholds: {
        perFile: true,
        lines: 95,
        statements: 95,
        functions: 95,
        branches: 95,
      },
    },
  },
});
