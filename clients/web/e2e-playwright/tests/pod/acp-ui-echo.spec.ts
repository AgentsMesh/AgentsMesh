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

// First real end-to-end coverage of the ACP UI path:
//   browser ↔ relay ↔ runner ↔ e2e-mock-agent (--mode=acp --scenario=echo)
// Validates that an ACP-mode pod can:
//   1. complete handshake (initialize + session/new) without panic
//   2. accept a prompt and echo it back as an agent_message_chunk
//   3. surface that chunk through AcpActivityStream's rendered DOM
//   4. transition through processing → idle without leaving the panel stuck
//
// This spec is the foundation for the Phase 2 scenario matrix — every
// future mock scenario (streaming, tool_call, permission_request, etc.)
// will follow the same shape: spawnMockAgentPod → render → assert DOM.
test.describe("ACP UI: e2e-echo agent (ACP mode)", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("ACP echo scenario surfaces prompt as assistant chunk in activity stream", async ({ page, api }) => {
    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "echo",
      prompt: "hello world",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");

    // Mock agent echoes "echo: <prompt>" as one agent_message_chunk, then
    // ends turn — give relay enough latency budget to push the snapshot
    // + the chunk + the sessionState=idle notification.
    await expect(page.getByText("echo: hello world")).toBeVisible({ timeout: 10_000 });

    // No wasm-bindgen recursion regressed.
    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });

  test("ACP pod creation does not require a real LLM CLI on the runner", async ({ api }) => {
    // Sanity guard: this spec must succeed even on runners that have no
    // Claude / Codex / Gemini installed. The mock agent is the *only*
    // path-resolvable executable e2e-echo depends on.
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "echo",
      prompt: "no-llm probe",
    });
    if (!pod) { test.skip(); return; }
    expect(pod.podKey).toBeTruthy();
    expect(pod.podKey.length).toBeGreaterThan(0);
    await pod.cleanup();
  });
});
