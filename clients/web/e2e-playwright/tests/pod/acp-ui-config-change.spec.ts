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
// See acp-ui-echo.spec.ts header — same r6 fix applies.
test.describe("ACP UI: control plane round-trip", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("clicking a mode in the selector updates the rendered label after server ack", async ({ page, api }) => {
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "config_change_plan",
      prompt: "ready",
    });

    await page.goto(workspaceUrlForPod(pod.podKey));
    // Use "load" (matches setupAcpScenarioPage in helpers/acp-spec-setup.ts):
    // Connect-RPC EventsService keeps connections open indefinitely so
    // "networkidle" times out under r6.
    await page.waitForLoadState("load");
    // Wait for the initial acknowledgment chunk so we know wasm session
    // is wired and Selector is mounted.
    await expect(page.getByText("Ready for mode switches", { exact: false })).toBeVisible({ timeout: 15_000 });

    // Open the selector — DropdownMenuTrigger button carries the active
    // mode's i18n description as `title`. Empty/unknown initial seed maps
    // to t("unknown.desc") = "Mode not yet reported by runner".
    await page.locator('button[title*="Mode" i], button[title*="Approve" i], button[title*="Auto-approve" i]').first().click();

    // Click "Default" mode entry in the dropdown.
    await page.getByText("Default", { exact: true }).first().click();

    // After the round-trip, the trigger should display "Default" (matches
    // the MODES[2].label for value="default").
    await expect(
      page.locator('button:has-text("Default")').first()
    ).toBeVisible({ timeout: 10_000 });
  });

  test("loopal-advertised modes render in the selector dropdown (capability path)", async ({ page, api }) => {
    // Mock advertises agentsmeshExtensions.permissionModes (loopal's 3 modes);
    // exercises the full advertise → parse → snapshot → selector render path.
    const pod = await createMockAgentPod(api, {
      mode: "acp",
      scenario: "permission_modes_loopal",
      prompt: "ready",
    });

    await page.goto(workspaceUrlForPod(pod.podKey));
    await page.waitForLoadState("load");
    await expect(page.getByText("Ready for mode switches", { exact: false })).toBeVisible({ timeout: 15_000 });

    await page.locator('button[title*="Mode" i], button[title*="Approve" i], button[title*="Auto-approve" i]').first().click();

    // loopal's advertised modes render (en labels); the Claude-only set does not.
    await expect(page.getByText("Ask Risky", { exact: true })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText("Ask Writes", { exact: true })).toBeVisible();
    await expect(page.getByText("Accept Edits", { exact: true })).toHaveCount(0);
  });
});
