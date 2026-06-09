// Migrated R5+: Connect-RPC only (no REST middle layer).
//
// Regression suite for GitHub issue #400:
// 1. agent_count must reflect channel_pods (not always 0)
// 2. RightRail must render joined pods via useChannelPods (not channel.pods)
// 3. Header agent_count must stay in sync with RightRail
// 4. member_count and agent_count are independent counters
//
// Pod creation uses a real dev runner — tests skip when none is available.
import { test, expect } from "../../fixtures/index";
import type { Page } from "@playwright/test";
import type { ApiFixture } from "../../fixtures/api.fixture";
import { ChannelsPage } from "../../pages/channels.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

type ConnectClient = Awaited<ReturnType<ApiFixture["connect"]>>;
type Runner = { id: bigint };
type Agent = { slug: string };
type Channel = { id: bigint; memberCount: bigint; agentCount: bigint };
type ChannelPod = { podKey: string };

const SECOND_USER = { email: "dev2@agentsmesh.local", password: "devpass123" };
const RAIL = '[data-testid="channel-right-rail"]';
const POD_ROW = '[data-testid="channel-rail-pod"]';

interface CreatedPod { podKey: string }
interface CreatedChannel { id: bigint; name: string; memberCount: number; agentCount: number }

async function createPod(cc: ConnectClient, _prompt: string): Promise<CreatedPod> {
  const { items: runners } = await cc.runner.listAvailableRunners({ orgSlug: TEST_ORG_SLUG }) as { items: Runner[] };
  expect(runners.length, "dev env must have an online runner").toBeGreaterThan(0);
  const { builtinAgents: agents } = await cc.agent.listAgents({ orgSlug: TEST_ORG_SLUG }) as { builtinAgents: Agent[] };
  expect(agents.length, "dev env must have a builtin agent").toBeGreaterThan(0);
  // Prefer `e2e-echo` — a runner-side stub agent that boots instantly without
  // an LLM. Falls back to the first builtin agent if the dev runner doesn't
  // ship it (e.g. a stripped-down image).
  const agent = agents.find((a) => a.slug === "e2e-echo") ?? agents[0];
  const resp = await cc.pod.createPod({
    orgSlug: TEST_ORG_SLUG,
    runnerId: runners[0].id,
    agentSlug: agent.slug,
  }) as { pod: { podKey: string } };
  const podKey = resp.pod?.podKey;
  expect(podKey, "createPod must return a pod_key").toBeTruthy();
  return { podKey: podKey! };
}

async function terminatePod(cc: ConnectClient, podKey: string): Promise<void> {
  await cc.pod.terminatePod({ orgSlug: TEST_ORG_SLUG, podKey }).catch(() => undefined);
}

async function createChannel(cc: ConnectClient, suffix: string): Promise<CreatedChannel> {
  const name = `E2E AgentCount ${suffix} ${Date.now()}`;
  const ch = await cc.channel.createChannel({ orgSlug: TEST_ORG_SLUG, name }) as Channel;
  return { id: ch.id, name, memberCount: Number(ch.memberCount), agentCount: Number(ch.agentCount) };
}

async function fetchChannel(cc: ConnectClient, id: bigint): Promise<{ memberCount: number; agentCount: number }> {
  const ch = await cc.channel.getChannel({ orgSlug: TEST_ORG_SLUG, id }) as Channel;
  return { memberCount: Number(ch.memberCount), agentCount: Number(ch.agentCount) };
}

async function selectInUI(page: Page, name: string): Promise<void> {
  const channels = new ChannelsPage(page, TEST_ORG_SLUG);
  await channels.goto();
  await channels.refreshButton.click().catch(() => undefined);
  await channels.selectChannel(name);
  await page.locator(RAIL).waitFor({ timeout: 15_000 });
}

async function archive(cc: ConnectClient, id: bigint): Promise<void> {
  await cc.channel.archiveChannel({ orgSlug: TEST_ORG_SLUG, id }).catch(() => undefined);
}

async function joinPod(cc: ConnectClient, id: bigint, podKey: string): Promise<void> {
  await cc.channel.joinChannelPod({ orgSlug: TEST_ORG_SLUG, id, podKey });
}

async function leavePod(cc: ConnectClient, id: bigint, podKey: string): Promise<void> {
  await cc.channel.leaveChannelPod({ orgSlug: TEST_ORG_SLUG, id, podKey }).catch(() => undefined);
}

test.describe("Channel × Pod membership (issue #400 regression)", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("W-001 fresh channel has member_count=1, agent_count=0", async ({ api }) => {
    const cc = await api.connect();
    const ch = await createChannel(cc, "fresh");
    expect(ch.memberCount).toBeGreaterThanOrEqual(1);
    expect(ch.agentCount).toBe(0);
    await archive(cc, ch.id);
  });

  test("W-002 join first pod → agent_count 0→1, RightRail shows pod row", async ({ api, page }) => {
    const cc = await api.connect();
    const pod = await createPod(cc, "agent-count-w002");
    const ch = await createChannel(cc, "first-pod");
    try {
      await joinPod(cc, ch.id, pod.podKey);

      const fresh = await fetchChannel(cc, ch.id);
      expect(fresh.agentCount).toBe(1);

      await selectInUI(page, ch.name);
      const rail = page.locator(RAIL);
      await expect(rail).toContainText("1");
      await expect(rail.locator(POD_ROW)).toHaveCount(1);
    } finally {
      await leavePod(cc, ch.id, pod.podKey);
      await terminatePod(cc, pod.podKey);
      await archive(cc, ch.id);
    }
  });

  test("W-003 join second pod → agent_count=2", async ({ api }) => {
    const cc = await api.connect();
    const p1 = await createPod(cc, "w003-a");
    const p2 = await createPod(cc, "w003-b");
    const ch = await createChannel(cc, "two-pods");
    try {
      await joinPod(cc, ch.id, p1.podKey);
      await joinPod(cc, ch.id, p2.podKey);

      const fresh = await fetchChannel(cc, ch.id);
      expect(fresh.agentCount).toBe(2);

      const { items } = await cc.channel.listChannelPods({
        orgSlug: TEST_ORG_SLUG,
        id: ch.id,
      }) as { items: ChannelPod[] };
      expect(items.length).toBe(2);
    } finally {
      for (const k of [p1.podKey, p2.podKey]) {
        await leavePod(cc, ch.id, k);
        await terminatePod(cc, k);
      }
      await archive(cc, ch.id);
    }
  });

  test("W-004 leave pod → agent_count decremented, row removed from RightRail", async ({ api, page }) => {
    const cc = await api.connect();
    const p1 = await createPod(cc, "w004-a");
    const p2 = await createPod(cc, "w004-b");
    const ch = await createChannel(cc, "leave-pod");
    try {
      await joinPod(cc, ch.id, p1.podKey);
      await joinPod(cc, ch.id, p2.podKey);

      await leavePod(cc, ch.id, p1.podKey);
      const fresh = await fetchChannel(cc, ch.id);
      expect(fresh.agentCount).toBe(1);

      await selectInUI(page, ch.name);
      await expect(page.locator(`${RAIL} ${POD_ROW}`)).toHaveCount(1);
    } finally {
      await leavePod(cc, ch.id, p2.podKey);
      for (const k of [p1.podKey, p2.podKey]) await terminatePod(cc, k);
      await archive(cc, ch.id);
    }
  });

  test("W-005 member_count and agent_count are independent counters", async ({ api }) => {
    const cc = await api.connect();
    const pod = await createPod(cc, "w005");
    const ch = await createChannel(cc, "independence");
    try {
      const initial = await fetchChannel(cc, ch.id);
      const baseMember = initial.memberCount;
      expect(initial.agentCount).toBe(0);

      // Invite user → member_count + 1, agent_count unchanged
      const { items: members } = await cc.org.listMembers({ orgSlug: TEST_ORG_SLUG }) as {
        items: { userId: bigint; user?: { email: string } }[];
      };
      const other = members?.find((m) => m.user?.email === SECOND_USER.email);
      if (other?.userId) {
        await cc.channel.inviteChannelMembers({
          orgSlug: TEST_ORG_SLUG,
          id: ch.id,
          userIds: [other.userId],
        });
        const after = await fetchChannel(cc, ch.id);
        expect(after.memberCount).toBe(baseMember + 1);
        expect(after.agentCount).toBe(0);
      }

      // Add pod → agent_count + 1, member_count unchanged
      const beforePod = await fetchChannel(cc, ch.id);
      await joinPod(cc, ch.id, pod.podKey);
      const afterPod = await fetchChannel(cc, ch.id);
      expect(afterPod.agentCount).toBe(1);
      expect(afterPod.memberCount).toBe(beforePod.memberCount);
    } finally {
      await leavePod(cc, ch.id, pod.podKey);
      await terminatePod(cc, pod.podKey);
      await archive(cc, ch.id);
    }
  });

  test("W-006 Header agent count matches RightRail count", async ({ api, page }) => {
    const cc = await api.connect();
    const pod = await createPod(cc, "w006");
    const ch = await createChannel(cc, "header-rail-sync");
    try {
      await joinPod(cc, ch.id, pod.podKey);
      await selectInUI(page, ch.name);
      await expect(page.locator(RAIL)).toContainText("1");
      // Header text matches "{n} agents" (en) or "{n} 个 Agent" (zh)
      await expect(
        page.getByText(/\b1\s*agents?\b|\b1\s*个 Agent\b/i).first()
      ).toBeVisible({ timeout: 10_000 });
    } finally {
      await leavePod(cc, ch.id, pod.podKey);
      await terminatePod(cc, pod.podKey);
      await archive(cc, ch.id);
    }
  });

  test("W-007 agent_count persists across navigation", async ({ api, page }) => {
    const cc = await api.connect();
    const pod = await createPod(cc, "w007");
    const ch = await createChannel(cc, "persist-nav");
    try {
      await joinPod(cc, ch.id, pod.podKey);

      await selectInUI(page, ch.name);
      await expect(page.locator(RAIL)).toContainText("1");

      await page.goto(`/${TEST_ORG_SLUG}/channels`);
      await page.waitForLoadState("load");
      await selectInUI(page, ch.name);

      await expect(page.locator(RAIL)).toContainText("1");
    } finally {
      await leavePod(cc, ch.id, pod.podKey);
      await terminatePod(cc, pod.podKey);
      await archive(cc, ch.id);
    }
  });

  test("W-008 switching channels does not leak pod list across channels", async ({ api, page }) => {
    const cc = await api.connect();
    const podA = await createPod(cc, "w008-a");
    const podB = await createPod(cc, "w008-b");
    const chA = await createChannel(cc, "switch-A");
    const chB = await createChannel(cc, "switch-B");
    try {
      await joinPod(cc, chA.id, podA.podKey);
      await joinPod(cc, chB.id, podB.podKey);

      await selectInUI(page, chA.name);
      await expect(page.locator(`${RAIL} ${POD_ROW}`)).toHaveCount(1);

      await selectInUI(page, chB.name);
      await expect(page.locator(`${RAIL} ${POD_ROW}`)).toHaveCount(1);

      // Back to A — must show A's pod, not B's stale data
      await selectInUI(page, chA.name);
      const rail = page.locator(RAIL);
      await expect(rail.locator(POD_ROW)).toHaveCount(1);
      await expect(rail).not.toContainText(podB.podKey);
    } finally {
      for (const [chId, key] of [[chA.id, podA.podKey], [chB.id, podB.podKey]] as const) {
        await leavePod(cc, chId, key);
        await terminatePod(cc, key);
        await archive(cc, chId);
      }
    }
  });

  test("W-009 terminated pod still counted in channel (historical membership)", async ({ api }) => {
    // Joining and lifecycle are decoupled: terminating a pod must not silently
    // change agent_count — explicit LeavePod is required.
    const cc = await api.connect();
    const pod = await createPod(cc, "w009");
    const ch = await createChannel(cc, "terminate-still-counted");
    try {
      await joinPod(cc, ch.id, pod.podKey);
      expect((await fetchChannel(cc, ch.id)).agentCount).toBe(1);

      await terminatePod(cc, pod.podKey);

      expect((await fetchChannel(cc, ch.id)).agentCount).toBe(1);
    } finally {
      await leavePod(cc, ch.id, pod.podKey);
      await archive(cc, ch.id);
    }
  });
});
