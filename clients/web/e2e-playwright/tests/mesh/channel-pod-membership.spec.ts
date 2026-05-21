import { test, expect } from "../../fixtures/index";
import type { Page } from "@playwright/test";
import type { ApiFixture } from "../../fixtures/api.fixture";
import { ChannelsPage } from "../../pages/channels.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// Regression suite for GitHub issue #400:
// 1. agent_count must reflect channel_pods (not always 0)
// 2. RightRail must render joined pods via useChannelPods (not channel.pods)
// 3. Header agent_count must stay in sync with RightRail
// 4. member_count and agent_count are independent counters
//
// Pod creation uses a real dev runner — tests skip when none is available.

const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;
const PODS = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;
const SECOND_USER = { email: "dev2@agentsmesh.local", password: "devpass123" };
const RAIL = '[data-testid="channel-right-rail"]';

interface CreatedPod { podKey: string }
interface CreatedChannel { id: number; name: string; member_count: number; agent_count: number }

async function createPod(api: ApiFixture, prompt: string): Promise<CreatedPod | null> {
  const runners = (await (await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/runners/available`)).json()).runners;
  if (!runners?.length) return null;
  const agents = (await (await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/agents`)).json()).builtin_agents;
  if (!agents?.length) return null;
  // Prefer `e2e-echo` — a runner-side stub agent that boots instantly without
  // an LLM. Falls back to the first builtin agent if the dev runner doesn't
  // ship it (e.g. a stripped-down image).
  const agent = agents.find((a: { slug: string }) => a.slug === "e2e-echo") ?? agents[0];
  const res = await api.post(PODS, { runner_id: runners[0].id, agent_slug: agent.slug, prompt });
  if (![200, 201].includes(res.status)) return null;
  const data = await res.json();
  const podKey = data.pod_key ?? data.pod?.pod_key;
  return podKey ? { podKey } : null;
}

async function terminatePod(api: ApiFixture, podKey: string): Promise<void> {
  await api.post(`${PODS}/${podKey}/terminate`, {}).catch(() => undefined);
}

async function createChannel(api: ApiFixture, suffix: string): Promise<CreatedChannel> {
  const name = `E2E AgentCount ${suffix} ${Date.now()}`;
  const res = await api.post(CHANNELS, { name });
  expect(res.status).toBe(201);
  const ch = (await res.json()).channel;
  return { id: ch.id, name, member_count: ch.member_count, agent_count: ch.agent_count };
}

async function fetchChannel(api: ApiFixture, id: number) {
  const res = await api.get(`${CHANNELS}/${id}`);
  expect(res.status).toBe(200);
  return (await res.json()).channel as { member_count: number; agent_count: number };
}

async function selectInUI(page: Page, name: string): Promise<void> {
  const channels = new ChannelsPage(page, TEST_ORG_SLUG);
  await channels.goto();
  await channels.refreshButton.click().catch(() => undefined);
  await channels.selectChannel(name);
  await page.locator(RAIL).waitFor({ timeout: 15_000 });
}

async function archive(api: ApiFixture, id: number): Promise<void> {
  await api.post(`${CHANNELS}/${id}/archive`, {}).catch(() => undefined);
}

test.describe("Channel × Pod membership (issue #400 regression)", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("W-001 fresh channel has member_count=1, agent_count=0", async ({ api }) => {
    const ch = await createChannel(api, "fresh");
    expect(ch.member_count).toBeGreaterThanOrEqual(1);
    expect(ch.agent_count).toBe(0);
    await archive(api, ch.id);
  });

  test("W-002 join first pod → agent_count 0→1, RightRail shows pod row", async ({ api, page }) => {
    const pod = await createPod(api, "agent-count-w002");
    if (!pod) { test.skip(); return; }
    const ch = await createChannel(api, "first-pod");
    try {
      const joinRes = await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: pod.podKey });
      expect(joinRes.status).toBe(200);
      expect((await joinRes.json()).channel?.agent_count).toBe(1);

      const fresh = await fetchChannel(api, ch.id);
      expect(fresh.agent_count).toBe(1);

      await selectInUI(page, ch.name);
      const rail = page.locator(RAIL);
      await expect(rail).toContainText("1");
      await expect(rail.locator("ul li")).toHaveCount(1);
    } finally {
      await api.delete(`${CHANNELS}/${ch.id}/pods/${pod.podKey}`).catch(() => undefined);
      await terminatePod(api, pod.podKey);
      await archive(api, ch.id);
    }
  });

  test("W-003 join second pod → agent_count=2", async ({ api }) => {
    const p1 = await createPod(api, "w003-a");
    const p2 = await createPod(api, "w003-b");
    if (!p1 || !p2) { test.skip(); return; }
    const ch = await createChannel(api, "two-pods");
    try {
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: p1.podKey });
      const r2 = await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: p2.podKey });
      expect(r2.status).toBe(200);
      expect((await r2.json()).channel?.agent_count).toBe(2);

      const podsRes = await api.get(`${CHANNELS}/${ch.id}/pods`);
      expect((await podsRes.json()).pods.length).toBe(2);
    } finally {
      for (const k of [p1.podKey, p2.podKey]) {
        await api.delete(`${CHANNELS}/${ch.id}/pods/${k}`).catch(() => undefined);
        await terminatePod(api, k);
      }
      await archive(api, ch.id);
    }
  });

  test("W-004 leave pod → agent_count decremented, row removed from RightRail", async ({ api, page }) => {
    const p1 = await createPod(api, "w004-a");
    const p2 = await createPod(api, "w004-b");
    if (!p1 || !p2) { test.skip(); return; }
    const ch = await createChannel(api, "leave-pod");
    try {
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: p1.podKey });
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: p2.podKey });

      const leaveRes = await api.delete(`${CHANNELS}/${ch.id}/pods/${p1.podKey}`);
      expect(leaveRes.status).toBe(200);
      expect((await leaveRes.json()).channel?.agent_count).toBe(1);

      await selectInUI(page, ch.name);
      await expect(page.locator(`${RAIL} ul li`)).toHaveCount(1);
    } finally {
      await api.delete(`${CHANNELS}/${ch.id}/pods/${p2.podKey}`).catch(() => undefined);
      for (const k of [p1.podKey, p2.podKey]) await terminatePod(api, k);
      await archive(api, ch.id);
    }
  });

  test("W-005 member_count and agent_count are independent counters", async ({ api }) => {
    const pod = await createPod(api, "w005");
    if (!pod) { test.skip(); return; }
    const ch = await createChannel(api, "independence");
    try {
      const initial = await fetchChannel(api, ch.id);
      const baseMember = initial.member_count;
      expect(initial.agent_count).toBe(0);

      // Invite user → member_count + 1, agent_count unchanged
      const members = (await (await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/members`)).json()).members;
      const other = members?.find((m: { user?: { email: string } }) => m.user?.email === SECOND_USER.email);
      if (other?.user_id) {
        await api.post(`${CHANNELS}/${ch.id}/members`, { user_ids: [other.user_id] });
        const after = await fetchChannel(api, ch.id);
        expect(after.member_count).toBe(baseMember + 1);
        expect(after.agent_count).toBe(0);
      }

      // Add pod → agent_count + 1, member_count unchanged
      const beforePod = await fetchChannel(api, ch.id);
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: pod.podKey });
      const afterPod = await fetchChannel(api, ch.id);
      expect(afterPod.agent_count).toBe(1);
      expect(afterPod.member_count).toBe(beforePod.member_count);
    } finally {
      await api.delete(`${CHANNELS}/${ch.id}/pods/${pod.podKey}`).catch(() => undefined);
      await terminatePod(api, pod.podKey);
      await archive(api, ch.id);
    }
  });

  test("W-006 Header agent count matches RightRail count", async ({ api, page }) => {
    const pod = await createPod(api, "w006");
    if (!pod) { test.skip(); return; }
    const ch = await createChannel(api, "header-rail-sync");
    try {
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: pod.podKey });
      await selectInUI(page, ch.name);
      await expect(page.locator(RAIL)).toContainText("1");
      // Header text matches "{n} agents" (en) or "{n} 个 Agent" (zh)
      await expect(
        page.getByText(/\b1\s*agents?\b|\b1\s*个 Agent\b/i).first()
      ).toBeVisible({ timeout: 10_000 });
    } finally {
      await api.delete(`${CHANNELS}/${ch.id}/pods/${pod.podKey}`).catch(() => undefined);
      await terminatePod(api, pod.podKey);
      await archive(api, ch.id);
    }
  });

  test("W-007 agent_count persists across navigation", async ({ api, page }) => {
    const pod = await createPod(api, "w007");
    if (!pod) { test.skip(); return; }
    const ch = await createChannel(api, "persist-nav");
    try {
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: pod.podKey });

      await selectInUI(page, ch.name);
      await expect(page.locator(RAIL)).toContainText("1");

      await page.goto(`/${TEST_ORG_SLUG}/channels`);
      await page.waitForLoadState("networkidle");
      await selectInUI(page, ch.name);

      await expect(page.locator(RAIL)).toContainText("1");
    } finally {
      await api.delete(`${CHANNELS}/${ch.id}/pods/${pod.podKey}`).catch(() => undefined);
      await terminatePod(api, pod.podKey);
      await archive(api, ch.id);
    }
  });

  test("W-008 switching channels does not leak pod list across channels", async ({ api, page }) => {
    const podA = await createPod(api, "w008-a");
    const podB = await createPod(api, "w008-b");
    if (!podA || !podB) { test.skip(); return; }
    const chA = await createChannel(api, "switch-A");
    const chB = await createChannel(api, "switch-B");
    try {
      await api.post(`${CHANNELS}/${chA.id}/pods`, { pod_key: podA.podKey });
      await api.post(`${CHANNELS}/${chB.id}/pods`, { pod_key: podB.podKey });

      await selectInUI(page, chA.name);
      await expect(page.locator(`${RAIL} ul li`)).toHaveCount(1);

      await selectInUI(page, chB.name);
      await expect(page.locator(`${RAIL} ul li`)).toHaveCount(1);

      // Back to A — must show A's pod, not B's stale data
      await selectInUI(page, chA.name);
      const rail = page.locator(RAIL);
      await expect(rail.locator("ul li")).toHaveCount(1);
      await expect(rail).not.toContainText(podB.podKey);
    } finally {
      for (const [chId, key] of [[chA.id, podA.podKey], [chB.id, podB.podKey]] as const) {
        await api.delete(`${CHANNELS}/${chId}/pods/${key}`).catch(() => undefined);
        await terminatePod(api, key);
        await archive(api, chId);
      }
    }
  });

  test("W-009 terminated pod still counted in channel (historical membership)", async ({ api }) => {
    // Joining and lifecycle are decoupled: terminating a pod must not silently
    // change agent_count — explicit LeavePod is required.
    const pod = await createPod(api, "w009");
    if (!pod) { test.skip(); return; }
    const ch = await createChannel(api, "terminate-still-counted");
    try {
      await api.post(`${CHANNELS}/${ch.id}/pods`, { pod_key: pod.podKey });
      expect((await fetchChannel(api, ch.id)).agent_count).toBe(1);

      await terminatePod(api, pod.podKey);

      expect((await fetchChannel(api, ch.id)).agent_count).toBe(1);
    } finally {
      await api.delete(`${CHANNELS}/${ch.id}/pods/${pod.podKey}`).catch(() => undefined);
      await archive(api, ch.id);
    }
  });
});
