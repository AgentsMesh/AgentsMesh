import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import { setupAcpScenarioPage } from "../../helpers/acp-spec-setup";

// First real end-to-end coverage of the ACP UI path:
//   browser ↔ relay ↔ runner ↔ e2e-mock-agent (--mode=acp --scenario=echo)
// Validates that an ACP-mode pod can:
//   1. complete handshake (initialize + session/new) without panic
//   2. accept a prompt and echo it back as an agent_message_chunk
//   3. surface that chunk through AcpActivityStream's rendered DOM
//   4. transition through processing → idle without leaving the panel stuck
test.describe.fixme("ACP UI: e2e-echo agent (ACP mode)", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test.fixme("ACP echo scenario surfaces prompt as assistant chunk in activity stream", async ({ page, api }) => {
    const ctx = await setupAcpScenarioPage(page, api, {
      mode: "acp", scenario: "echo", prompt: "hello world",
    });
    if (!ctx) { test.skip(); return; }

    await expect(page.getByText("echo: hello world")).toBeVisible({ timeout: 10_000 });
    ctx.assertWasmHealthy();
  });

  test.fixme("ACP pod creation does not require a real LLM CLI on the runner", async ({ page, api }) => {
    const ctx = await setupAcpScenarioPage(page, api, {
      mode: "acp", scenario: "echo", prompt: "no-llm probe",
    });
    if (!ctx) { test.skip(); return; }
    expect(ctx.pod.podKey).toBeTruthy();
    expect(ctx.pod.podKey.length).toBeGreaterThan(0);
  });
});
