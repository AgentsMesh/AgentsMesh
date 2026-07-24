import { defineConfig } from "vitest/config";

import baseConfig from "./vitest.config";

const relayOwnerTests = [
  "src/hooks/__tests__/useAcpRelay.test.ts",
  "src/hooks/__tests__/useTerminalConnection-selfheal.test.ts",
  "src/hooks/__tests__/useTerminalInit.test.ts",
  "src/hooks/__tests__/useTerminalInit.safeFit.test.ts",
  "src/hooks/__tests__/useTerminalInit.unicode.test.ts",
  "src/hooks/__tests__/xtermUnicodeWidth.test.ts",
  "src/lib/terminal/runnerUnicodeGraphemes.test.ts",
  "src/lib/e2e/relayReadinessProbe.test.ts",
  "src/stores/__tests__/relayConnection.direct.test.ts",
  "src/stores/__tests__/relayConnection.node.test.ts",
  "src/stores/__tests__/relayConnection.test.ts",
  "src/stores/__tests__/relayConnectionEvents.test.ts",
  "src/stores/__tests__/relayEndpointSelection.test.ts",
  "src/stores/__tests__/relayPodRegistry.test.ts",
  "src/stores/__tests__/relayPodSession.test.ts",
  "src/stores/__tests__/relayProbe.test.ts",
  "src/stores/__tests__/relaySubscriptionAbort.test.ts",
];

const relayOwnerSources = [
  "src/hooks/useAcpRelay.ts",
  "src/hooks/useTerminalConnection.ts",
  "src/hooks/useTerminalInit.ts",
  "src/lib/terminal/runnerUnicodeGraphemes.ts",
  "src/lib/e2e/relayReadinessProbe.ts",
  "src/stores/relayConnection.ts",
  "src/stores/relayConnectionEvents.ts",
  "src/stores/relayEndpointSelection.ts",
  "src/stores/relayPodRegistry.ts",
  "src/stores/relayPodSession.ts",
  "src/stores/relayProbe.ts",
  "src/stores/relaySubscriptionAbort.ts",
];

export default defineConfig({
  ...baseConfig,
  test: {
    ...baseConfig.test,
    include: relayOwnerTests,
    coverage: {
      ...baseConfig.test?.coverage,
      enabled: true,
      provider: "v8",
      reporter: ["text", "json-summary"],
      reportsDirectory: "./coverage-relay",
      include: relayOwnerSources,
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
