import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import { setupAcpScenarioPage } from "../../helpers/acp-spec-setup";

// Scenario coverage for the universal mock agent — one spec per scenario,
// each one exercises a distinct slice of the ACP UI render path.
//
//   streaming_3              StreamingCaret + complete-flag pipeline
//   thinking_then_answer     ThinkingIndicator spinner + collapse
//   tool_call_edit           AcpToolCallCard animate-pulse → ✓ icon
//   permission_request_edit  AcpPermissionDialog full approve flow
test.describe.fixme("ACP UI: mock agent scenario matrix", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test.fixme("streaming_3 emits three chunks concatenated in the activity stream", async ({ page, api }) => {
    const ctx = await setupAcpScenarioPage(page, api, {
      mode: "acp", scenario: "streaming_3", prompt: "hello",
    });
    if (!ctx) { test.skip(); return; }

    await expect(page.getByText(/streaming: hello\s+\(done\)/)).toBeVisible({ timeout: 15_000 });
    ctx.assertWasmHealthy();
  });

  test.fixme("thinking_then_answer renders ThinkingIndicator and final content", async ({ page, api }) => {
    const ctx = await setupAcpScenarioPage(page, api, {
      mode: "acp", scenario: "thinking_then_answer", prompt: "what is 2+2",
    });
    if (!ctx) { test.skip(); return; }

    await expect(page.getByText("Thinking...", { exact: false })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("Answer to: what is 2+2")).toBeVisible({ timeout: 15_000 });
  });

  test.fixme("tool_call_edit renders AcpToolCallCard with completed status", async ({ page, api }) => {
    const ctx = await setupAcpScenarioPage(page, api, {
      mode: "acp", scenario: "tool_call_edit", prompt: "edit me",
    });
    if (!ctx) { test.skip(); return; }

    await expect(page.getByText("Edit", { exact: true }).first()).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("Editing file for: edit me")).toBeVisible({ timeout: 15_000 });
  });

  test.fixme("permission_request_edit shows permission dialog and approval completes the tool", async ({ page, api }) => {
    const ctx = await setupAcpScenarioPage(page, api, {
      mode: "acp", scenario: "permission_request_edit", prompt: "edit me carefully",
    });
    if (!ctx) { test.skip(); return; }

    await expect(page.getByText(/Tool: tc-mock-edit-perm-1/)).toBeVisible({ timeout: 15_000 });
    await page.getByRole("button", { name: /Approve/i }).first().click();
    await expect(page.getByText(/Tool: tc-mock-edit-perm-1/)).not.toBeVisible({ timeout: 10_000 });
  });
});
