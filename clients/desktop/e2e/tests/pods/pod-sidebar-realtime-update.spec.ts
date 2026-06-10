// Validate the desktop sidebar updates without manual reload when a pod
// is created through the IPC bridge.
//
// Pre-bridge: pod-realtime.spec.ts:90-104 explicitly documents this
// limitation — "Desktop's electron-adapter ships a NoopEventsManager:
// realtime pod:created events are not delivered to the renderer until a
// main-process Connect ServerStream bridge lands. The seed podCreatePod
// bypasses the renderer entirely (direct IPC → Connect → DB), so there
// is no event to dispatch... Reload the page so WorkspaceSidebarContent
// mounts fresh."
//
// Post-bridge: the realtime event flows main → renderer, and the sidebar
// store's handlePodEvent("pod:created") pulls the single new entity via
// fetchPod (IPC → app_pod_insert_created → upsert + tick bump). The pod
// lands in the SSOT cache and the client-side filter renders it — no full
// list refetch, no reload.
import { test, expect } from "../../fixtures";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";
import { invokeIpc } from "../../helpers/ipc";

test.describe("Desktop sidebar · realtime pod refresh (no reload)", () => {
  test("creating a pod via IPC updates the sidebar within the realtime window", async ({ page }) => {
    await gotoHash(page, `/${TEST_ORG_SLUG}/workspace`);
    await page.waitForTimeout(2_000); // bridge connect settle

    // Prerequisites
    const runners = await invokeIpc<string>(page, "runnerFetchRunners");
    const runnerList = JSON.parse(runners) as { runners?: { id: number; status: string }[] };
    const onlineRunner = (runnerList.runners ?? []).find((r) => r.status === "online");
    expect(onlineRunner, "dev env must have an online runner").toBeTruthy();

    const agentsJson = await invokeIpc<string>(page, "agentListAgents");
    const agents = JSON.parse(agentsJson) as { builtin_agents?: { slug: string }[] };
    const agent = agents.builtin_agents?.[0];
    expect(agent, "dev env must have a builtin agent").toBeTruthy();

    // Capture sidebar state pre-create.
    const sidebarSelector = '[data-testid="pod-list-item"]';
    const beforeCount = await page.locator(sidebarSelector).count();

    const created = await invokeIpc<string>(page, "podCreatePod", JSON.stringify({
      agent_slug: agent!.slug, runner_id: onlineRunner!.id, cols: 80, rows: 24,
    }));
    const { pod } = JSON.parse(created) as { pod: { pod_key: string } };

    try {
      // Sidebar should grow without manual reload. fetchPod resolves the new
      // entity into the SSOT cache; we wait for the event → upsert → render.
      const newPodSelector = `${sidebarSelector}[data-pod-key="${pod.pod_key}"]`;
      await expect(page.locator(newPodSelector)).toBeVisible({ timeout: 8_000 });
      const afterCount = await page.locator(sidebarSelector).count();
      expect(afterCount).toBeGreaterThan(beforeCount);
    } finally {
      await invokeIpc(page, "podTerminatePod", pod.pod_key).catch(() => undefined);
    }
  });
});
