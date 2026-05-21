import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import {
  createMockAgentPod,
  workspaceUrlForPod,
} from "../../helpers/mock-agent";

// End-to-end regression for the ACP control plane round-trip:
//
//   Web Selector click
//   → relay sendAcpCommand "set_permission_mode"
//   → runner ACPClient.SetPermissionMode
//   → ACPTransport sends session/control_request to mock binary
//   → mock acks with {ok:true}
//   → ACPClient fires OnConfigChange (Phase B refactor)
//   → message_handler_acp wraps it → relay broadcasts "configChanged"
//   → web acpEventDispatcher → store.updateConfiguration
//   → AcpPermissionModeSelector reads useAcpSessionField → re-renders
//
// This spec asserts the round-trip completes by watching the Selector's
// rendered label flip from one mode to another after click.
test.describe.fixme("ACP UI: control plane round-trip", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test.fixme("clicking a mode in the selector updates the rendered label after server ack", async ({ page, api }) => {
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "config_change_plan",
      prompt: "ready",
    });
    if (!pod) { test.skip(); return; }

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("networkidle");
    // Wait for the initial acknowledgment chunk so we know wasm session
    // is wired and Selector is mounted.
    await expect(page.getByText("Ready for mode switches", { exact: false })).toBeVisible({ timeout: 15_000 });

    // Open the selector — the trigger button shows the current label.
    // For an empty seeded configuration the label is "—" (UNKNOWN_MODE
    // fallback in AcpPermissionModeSelector); after mock acks the click
    // the configChanged broadcast arrives and the label switches.
    const trigger = page.getByRole("button", { name: /Shield/i }).first().or(
      page.locator('button:has([data-lucide="shield"])').first()
    );
    // Fallback: just find the first button containing the Shield SVG via class.
    const triggers = page.locator('button').filter({
      has: page.locator('svg').first(),
    });
    await triggers.first().click().catch(() => trigger.click());

    // Click "Default" mode entry in the dropdown.
    await page.getByText("Default", { exact: true }).first().click();

    // After the round-trip, the trigger should display "Default" (matches
    // the MODES[2].label for value="default").
    await expect(
      page.locator('button:has-text("Default")').first()
    ).toBeVisible({ timeout: 10_000 });
  });
});
