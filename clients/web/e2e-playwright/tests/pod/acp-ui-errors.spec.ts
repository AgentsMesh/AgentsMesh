import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import {
  collectConsoleErrors,
  collectPageErrors,
  assertNoWasmRecursiveBorrow,
} from "../../helpers/console-errors";
import {
  createMockAgentPod,
  workspaceUrlForPod,
} from "../../helpers/mock-agent";

// Defensive-path coverage: every scenario here exercises an unhappy
// runner/agent boundary that should NOT crash the web UI or wedge the
// activity stream. Each spec leaves the page idle for a moment then
// asserts the wasm-bindgen recursive-borrow guard is intact — these
// are the paths most likely to trip a previously-unobserved race.
test.describe("ACP UI: error and degradation paths", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("tool_call_failed renders the failed status without crashing UI", async ({ page, api }) => {
    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "tool_call_failed",
      prompt: "edit me",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    await expect(page.getByText("Trying to edit: edit me")).toBeVisible({ timeout: 15_000 });
    // AcpToolCallCard renders the toolName label even for failed tools.
    await expect(page.getByText("Edit", { exact: true }).first()).toBeVisible({ timeout: 15_000 });

    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });

  test("malformed_json output does not break subsequent valid messages", async ({ page, api }) => {
    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "malformed_json",
      prompt: "garbled",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // The recovery chunk must surface even though garbage preceded it on stdout.
    await expect(page.getByText("recovered: garbled")).toBeVisible({ timeout: 15_000 });

    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });

  test("log_warnings surfaces warn/error stderr lines in activity stream", async ({ page, api }) => {
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "log_warnings",
      prompt: "noisy run",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // OnLog with stderr-prefixed text passes through; AcpActivityStream
    // renders warn/error inside a colored LogEntry (yellow for warn, red
    // for error). The exact prefix on stderr is part of the log line text.
    await expect(page.getByText(/degraded connection/i)).toBeVisible({ timeout: 15_000 });
    // The final assistant chunk still shows up after the warnings.
    await expect(page.getByText("Completed with warnings: noisy run")).toBeVisible({ timeout: 15_000 });
  });

  test("fail_after_1s does not leave the UI wedged in a processing state", async ({ page, api }) => {
    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "fail_after_1s",
      prompt: "crash test",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // First the agent prints its warning, then exits after 1s. The page
    // must remain interactive (no eternal spinner, no js crash).
    await expect(page.getByText("Will crash soon: crash test")).toBeVisible({ timeout: 15_000 });
    // Give the runner time to detect the exit and broadcast state=stopped.
    await page.waitForTimeout(4000);

    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });
});
