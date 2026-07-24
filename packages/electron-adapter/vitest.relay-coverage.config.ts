import { defineConfig } from "vitest/config";

import baseConfig from "./vitest.config";

export default defineConfig({
  ...baseConfig,
  test: {
    ...baseConfig.test,
    include: ["src/electron-relay-manager.test.ts"],
    coverage: {
      enabled: true,
      provider: "v8",
      reporter: ["text", "json-summary"],
      reportsDirectory: "./coverage-relay",
      include: [
        "src/electron-relay-manager.ts",
        "src/electron-relay-output-routes.ts",
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
