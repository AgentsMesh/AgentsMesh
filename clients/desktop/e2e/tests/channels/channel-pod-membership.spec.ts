import { test, expect } from "../../fixtures";
import { invokeIpc } from "../../helpers/ipc";
import { gotoHash } from "../../helpers/nav";
import { TEST_ORG_SLUG } from "../../helpers/env";

// Desktop counterpart of clients/web/e2e-playwright/tests/mesh/channel-pod-membership.spec.ts.
// Exercises the same regression (issue #400) through the Electron bridge:
//   - Channel created via IPC (channel_create_channel)
//   - Pod joined via IPC (channel_join_channel)
//   - RightRail rendered by the same React tree the web app uses
// Pod creation still goes through REST because the desktop bridge does
// not expose a pod_create IPC.

const PODS = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;
const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;
const RAIL = '[data-testid="channel-right-rail"]';

interface CreatedPod { podKey: string }

async function createPodViaApi(api: import("../../../../web/e2e-playwright/fixtures/api.fixture").ApiFixture, prompt: string): Promise<CreatedPod | null> {
  const runners = (await (await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/runners/available`)).json()).runners;
  if (!runners?.length) return null;
  const agents = (await (await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/agents`)).json()).builtin_agents;
  if (!agents?.length) return null;
  const agent = agents.find((a: { slug: string }) => a.slug === "e2e-echo") ?? agents[0];
  const res = await api.post(PODS, { runner_id: runners[0].id, agent_slug: agent.slug, prompt });
  if (![200, 201].includes(res.status)) return null;
  const data = await res.json();
  const podKey = data.pod_key ?? data.pod?.pod_key;
  return podKey ? { podKey } : null;
}

test.describe("Channel × Pod membership (Electron IPC, issue #400)", () => {
  test("D-001 channel_join_channel → RightRail renders pod row", async ({ page, api }) => {
    const pod = await createPodViaApi(api, "desktop-d001");
    if (!pod) { test.skip(); return; }

    // Create channel via IPC (exercises the bridge end-to-end).
    const name = `e2e-desktop-d001-${Date.now()}`;
    const createJson = await invokeIpc<string>(page, "channelCreateChannel",
      JSON.stringify({ name, visibility: "public" }));
    const channel = JSON.parse(createJson) as { id: number; agent_count: number };
    expect(channel.id).toBeGreaterThan(0);
    expect(channel.agent_count).toBe(0);

    try {
      // Join via IPC. Returns updated Channel JSON with agent_count refreshed.
      const joinedJson = await invokeIpc<string>(page, "channelJoinChannel", channel.id, pod.podKey);
      const joined = JSON.parse(joinedJson) as { agent_count?: number };
      expect(joined.agent_count).toBe(1);

      // Verify Rust core's pods_by_channel cache is populated (set by join_channel).
      const cachedJson = await invokeIpc<string>(page, "channelChannelPodsJson", channel.id);
      const cached = JSON.parse(cachedJson) as Array<{ pod_key: string }>;
      expect(cached.length, "pods_by_channel cache after channelJoinChannel").toBe(1);
      expect(cached[0]?.pod_key).toBe(pod.podKey);

      // Navigate and verify UI surfaces the new pod row + count.
      await gotoHash(page, `/${TEST_ORG_SLUG}/channels/${channel.id}`);
      const rail = page.locator(RAIL);
      await rail.waitFor({ timeout: 15_000 });
      await expect(rail).toContainText("1");
      await expect(rail.locator("ul li")).toHaveCount(1);
    } finally {
      await api.delete(`${CHANNELS}/${channel.id}/pods/${pod.podKey}`).catch(() => undefined);
      await api.post(`${PODS}/${pod.podKey}/terminate`, {}).catch(() => undefined);
      await api.post(`${CHANNELS}/${channel.id}/archive`, {}).catch(() => undefined);
    }
  });

  test("D-002 channel_leave_channel → pod row removed from RightRail", async ({ page, api }) => {
    const p1 = await createPodViaApi(api, "desktop-d002-a");
    const p2 = await createPodViaApi(api, "desktop-d002-b");
    if (!p1 || !p2) { test.skip(); return; }

    const name = `e2e-desktop-d002-${Date.now()}`;
    const createJson = await invokeIpc<string>(page, "channelCreateChannel",
      JSON.stringify({ name, visibility: "public" }));
    const channel = JSON.parse(createJson) as { id: number };

    try {
      await invokeIpc<string>(page, "channelJoinChannel", channel.id, p1.podKey);
      await invokeIpc<string>(page, "channelJoinChannel", channel.id, p2.podKey);

      await gotoHash(page, `/${TEST_ORG_SLUG}/channels/${channel.id}`);
      const rail = page.locator(RAIL);
      await rail.waitFor({ timeout: 15_000 });
      await expect(rail.locator("ul li")).toHaveCount(2);

      // Leave one pod via IPC — RightRail must drop to 1 row.
      const leftJson = await invokeIpc<string>(page, "channelLeaveChannel", channel.id, p1.podKey);
      const left = JSON.parse(leftJson) as { agent_count?: number };
      expect(left.agent_count).toBe(1);

      // Verify the renderer-side pod cache reflects the leave. This catches a
      // common failure mode: backend dropped the pod but ChannelLocalState's
      // _podsByChannel still mirrors the pre-leave list (no notify).
      const afterLeaveCache = await page.evaluate(async ({ id }) => {
        const svc = (window as unknown as { electronAPI: { invoke: (m: string, ...a: unknown[]) => Promise<unknown> } }).electronAPI;
        // Trigger an explicit refresh through ElectronChannelService.get_channel_pods,
        // which mirrors the backend response into _podsByChannel.
        await svc.invoke("channelGetChannelPods", id);
        return svc.invoke("channelChannelPodsJson", id) as Promise<string>;
      }, { id: channel.id });
      const cachedAfterLeave = JSON.parse(afterLeaveCache) as Array<{ pod_key: string }>;
      expect(cachedAfterLeave.length, "channel_pods_json after leave").toBe(1);

      // Re-navigate (different route then back) to force RightRail unmount/remount.
      await gotoHash(page, `/${TEST_ORG_SLUG}/workspace`);
      await gotoHash(page, `/${TEST_ORG_SLUG}/channels/${channel.id}`);
      await rail.waitFor({ timeout: 15_000 });
      await expect(rail.locator("ul li")).toHaveCount(1);
      await expect(rail).not.toContainText(p1.podKey);
    } finally {
      for (const k of [p1.podKey, p2.podKey]) {
        await api.delete(`${CHANNELS}/${channel.id}/pods/${k}`).catch(() => undefined);
        await api.post(`${PODS}/${k}/terminate`, {}).catch(() => undefined);
      }
      await api.post(`${CHANNELS}/${channel.id}/archive`, {}).catch(() => undefined);
    }
  });
});
