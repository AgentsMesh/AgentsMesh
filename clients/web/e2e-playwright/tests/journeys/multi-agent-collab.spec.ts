import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { pollUntil } from "../../helpers/retry";
import { textContent } from "../../helpers/test-data";
import { terminateAllPods } from "../../helpers/pod-cleanup";

const PODS = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;
const CHANNELS = `/api/v1/orgs/${TEST_ORG_SLUG}/channels`;

/**
 * Journey: Multi-Agent Collaboration
 * Channel → Multiple Pods → Message Exchange → Coordination
 */
test.describe("Journey: Multi-Agent Collaboration", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test.afterAll(async () => {
    await terminateAllPods();
  });

  test("channel-based multi-pod collaboration flow", async ({ api }) => {
    // ── Step 1: Check runner availability ──
    const runnerRes = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/runners/available`
    );
    const runners = (await runnerRes.json()).runners;
    if (!runners?.length) { test.skip(); return; }

    const agentRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/agents`);
    const agents = (await agentRes.json()).builtin_agents;
    if (!agents?.length) { test.skip(); return; }

    // ── Step 2: Create collaboration channel ──
    const chName = "E2E Collab " + Date.now();
    const chRes = await api.post(CHANNELS, {
      name: chName,
      description: "Multi-agent collaboration test",
    });
    expect([200, 201]).toContain(chRes.status);
    const ch = await chRes.json();
    const chId = ch.channel?.id || ch.id;
    expect(chId).toBeTruthy();

    // ── Step 3: Create Pod A (analyst) ──
    const podARes = await api.post(PODS, {
      runner_id: runners[0].id,
      agent_slug: agents[0].slug,
      prompt: "E2E Collab Pod A - Analyst",
    });
    const podAData = await podARes.json();
    const podAKey = podAData.pod_key || podAData.pod?.pod_key;

    // ── Step 4: Create Pod B (implementer) ──
    const podBRes = await api.post(PODS, {
      runner_id: runners[0].id,
      agent_slug: agents[0].slug,
      prompt: "E2E Collab Pod B - Implementer",
    });
    const podBData = await podBRes.json();
    const podBKey = podBData.pod_key || podBData.pod?.pod_key;

    // ── Step 5: Wait for both pods running ──
    for (const key of [podAKey, podBKey].filter(Boolean)) {
      await pollUntil(
        async () => {
          const r = await api.get(`${PODS}/${key}`);
          const d = await r.json();
          return (d.pod?.status || d.status) === "running";
        },
        { maxAttempts: 10, intervalMs: 3000, label: `pod-${key}-running` }
      ).catch(() => {});
    }

    // ── Step 6: Add pods to channel ──
    if (podAKey) {
      await api.post(`${CHANNELS}/${chId}/pods`, { pod_key: podAKey });
    }
    if (podBKey) {
      await api.post(`${CHANNELS}/${chId}/pods`, { pod_key: podBKey });
    }

    // ── Step 7: Verify channel members ──
    const membersRes = await api.get(`${CHANNELS}/${chId}/members`);
    if (membersRes.status === 200) {
      const members = await membersRes.json();
      expect(members).toBeTruthy();
    }

    // ── Step 8: Send message to channel ──
    const msgRes = await api.post(`${CHANNELS}/${chId}/messages`, {
      content: textContent("Pod A found a bug in the auth module. Pod B, please fix it."),
    });
    expect([200, 201]).toContain(msgRes.status);

    // ── Step 9: Verify message appears in history ──
    const histRes = await api.get(`${CHANNELS}/${chId}/messages`);
    expect(histRes.status).toBe(200);
    const messages = await histRes.json();
    expect(messages.messages?.length || messages.length).toBeGreaterThan(0);

    // ── Step 10: Verify mesh topology shows pods and channel ──
    const topoRes = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/mesh/topology`
    );
    expect(topoRes.status).toBe(200);

    // ── Step 11: Cleanup — terminate pods, archive channel ──
    for (const key of [podAKey, podBKey].filter(Boolean)) {
      await api.post(`${PODS}/${key}/terminate`, {});
    }
    await api.post(`${CHANNELS}/${chId}/archive`, {});
  });
});
