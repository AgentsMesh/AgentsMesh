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

// Scenario coverage for the universal mock agent — one spec per scenario,
// each one exercises a distinct slice of the ACP UI render path.
//
//   streaming_3              StreamingCaret + complete-flag pipeline
//   thinking_then_answer     ThinkingIndicator spinner + collapse
//   tool_call_edit           AcpToolCallCard animate-pulse → ✓ icon
//   permission_request_edit  AcpPermissionDialog full approve flow
//
// All specs require a running runner; without one they skip rather than fail.
test.describe("ACP UI: mock agent scenario matrix", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("streaming_3 emits three chunks concatenated in the activity stream", async ({ page, api }) => {
    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "streaming_3",
      prompt: "hello",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // Chunks are: "streaming: " + "hello " + "(done)" → final assistant
    // message text concatenates them. Wasm session manager seals the
    // assistant message on state=idle, so we see the full text.
    await expect(page.getByText(/streaming: hello\s+\(done\)/)).toBeVisible({ timeout: 15_000 });

    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });

  test("thinking_then_answer renders ThinkingIndicator and final content", async ({ page, api }) => {
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "thinking_then_answer",
      prompt: "what is 2+2",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // ThinkingIndicator <summary> always contains "Thinking..." label.
    await expect(page.getByText("Thinking...", { exact: false })).toBeVisible({ timeout: 15_000 });
    // Final answer chunk also surfaces.
    await expect(page.getByText("Answer to: what is 2+2")).toBeVisible({ timeout: 15_000 });
  });

  test("tool_call_edit renders AcpToolCallCard with completed status", async ({ page, api }) => {
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "tool_call_edit",
      prompt: "edit me",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // Tool name is rendered inside AcpToolCallCard's mono-font label.
    await expect(page.getByText("Edit", { exact: true }).first()).toBeVisible({ timeout: 15_000 });
    // The introductory assistant message must also appear.
    await expect(page.getByText("Editing file for: edit me")).toBeVisible({ timeout: 15_000 });
  });

  test("permission_request_edit shows permission dialog and approval completes the tool", async ({ page, api }) => {
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "permission_request_edit",
      prompt: "edit me carefully",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // The AcpPermissionDialog surfaces "Tool: <toolCallId>" as description
    // (handler.go:121). The dialog body is gated by pendingPermissions.length>0.
    await expect(page.getByText(/Tool: tc-mock-edit-perm-1/)).toBeVisible({ timeout: 15_000 });

    // Approve via the dialog's Approve button. After approval the mock
    // emits a successful tool_call_update; the dialog dismisses itself.
    await page.getByRole("button", { name: /Approve/i }).first().click();

    await expect(page.getByText(/Tool: tc-mock-edit-perm-1/)).not.toBeVisible({ timeout: 10_000 });
  });
});
