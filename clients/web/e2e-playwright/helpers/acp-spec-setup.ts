import type { Page } from "@playwright/test";
import type { ApiFixture } from "../fixtures/api.fixture";
import {
  collectConsoleErrors,
  collectPageErrors,
  assertNoWasmRecursiveBorrow,
} from "./console-errors";
import {
  createMockAgentPod,
  workspaceUrlForPod,
  type CreateMockPodOptions,
  type MockAgentPod,
} from "./mock-agent";

// SetupAcpScenarioResult bundles the per-spec context so each test can
// reach for exactly what it needs without a fixture explosion.
export interface SetupAcpScenarioResult {
  pod: MockAgentPod;
  consoleErrors: () => string[];
  pageErrors: () => string[];
  /** Asserts the wasm-bindgen recursive-borrow guard is intact. */
  assertWasmHealthy: () => void;
}

// setupAcpScenarioPage encapsulates the 4-step prologue every ACP UI spec
// repeats: spawn mock pod → wire console/error collectors → navigate to the
// workspace → wait for network idle. Returns null when no runner is online
// so the caller can `test.skip()`.
//
// The returned object holds the live error collectors as functions; calling
// them returns the current snapshot, which lets specs assert the recursive-
// borrow guard at any point without recomputing the wiring.
export async function setupAcpScenarioPage(
  page: Page,
  api: ApiFixture,
  opts: CreateMockPodOptions,
): Promise<SetupAcpScenarioResult | null> {
  const consoleErrors = collectConsoleErrors(page);
  const pageErrors = collectPageErrors(page);

  const pod = await createMockAgentPod(api, opts);
  if (!pod) return null;

  await page.goto(workspaceUrlForPod(pod.podKey));
  await page.waitForLoadState("networkidle");

  return {
    pod,
    consoleErrors: () => consoleErrors,
    pageErrors: () => pageErrors,
    assertWasmHealthy: () => {
      assertNoWasmRecursiveBorrow(consoleErrors);
      assertNoWasmRecursiveBorrow(pageErrors);
    },
  };
}
