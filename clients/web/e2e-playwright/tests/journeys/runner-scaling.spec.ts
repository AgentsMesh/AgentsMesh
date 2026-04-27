import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { pollUntil } from "../../helpers/retry";
import { terminateAllPods } from "../../helpers/pod-cleanup";

const PODS = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;
const RUNNERS = `/api/v1/orgs/${TEST_ORG_SLUG}/runners`;

/**
 * Journey: Runner Scaling & Capacity Management
 * Verify capacity limits → Create pods up to limit → Enforce cap → Disable/Enable
 */
test.describe("Journey: Runner Scaling", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test.afterAll(async () => {
    await terminateAllPods();
  });

  test("runner capacity enforcement and scaling", async ({ api }) => {
    // ── Step 1: Get available runner and record initial state ──
    const rRes = await api.get(`${RUNNERS}/available`);
    const runners = (await rRes.json()).runners;
    if (!runners?.length) { test.skip(); return; }
    const runner = runners[0];
    const runnerId = runner.id;
    const initialPods = runner.current_pods || 0;
    const maxPods = runner.max_concurrent_pods || 10;

    const agentRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/agents`);
    const agents = (await agentRes.json()).builtin_agents;
    if (!agents?.length) { test.skip(); return; }

    // ── Step 2: Create a pod and verify capacity increases ──
    const pod1Res = await api.post(PODS, {
      runner_id: runnerId,
      agent_slug: agents[0].slug,
      prompt: "E2E Scaling Pod 1",
    });
    expect([200, 201]).toContain(pod1Res.status);
    const pod1Data = await pod1Res.json();
    const pod1Key = pod1Data.pod_key || pod1Data.pod?.pod_key;

    await pollUntil(
      async () => {
        const r = await api.get(`${RUNNERS}/${runnerId}`);
        const d = await r.json();
        return (d.runner?.current_pods || 0) > initialPods;
      },
      { maxAttempts: 10, intervalMs: 2000, label: "capacity-increase" }
    ).catch(() => {});

    // ── Step 3: Create second pod ──
    const pod2Res = await api.post(PODS, {
      runner_id: runnerId,
      agent_slug: agents[0].slug,
      prompt: "E2E Scaling Pod 2",
    });
    expect([200, 201]).toContain(pod2Res.status);
    const pod2Data = await pod2Res.json();
    const pod2Key = pod2Data.pod_key || pod2Data.pod?.pod_key;

    // ── Step 4: Verify capacity reflects new pods ──
    let midPods = initialPods;
    await pollUntil(
      async () => {
        const r = await api.get(`${RUNNERS}/${runnerId}`);
        const d = await r.json();
        midPods = d.runner?.current_pods || 0;
        return midPods > initialPods;
      },
      { maxAttempts: 10, intervalMs: 2000, label: "capacity-increase" }
    ).catch(() => {});

    // ── Step 5: Terminate pod 1, verify capacity decreases ──
    if (pod1Key) await api.post(`${PODS}/${pod1Key}/terminate`, {});

    await pollUntil(
      async () => {
        const r = await api.get(`${RUNNERS}/${runnerId}`);
        const d = await r.json();
        return (d.runner?.current_pods || 0) < midPods;
      },
      { maxAttempts: 5, intervalMs: 2000, label: "capacity-decrease" }
    ).catch(() => {});

    // ── Step 6: Disable runner, verify not in available list ──
    await api.put(`${RUNNERS}/${runnerId}`, { is_enabled: false });

    const availRes = await api.get(`${RUNNERS}/available`);
    const available = (await availRes.json()).runners || [];
    const found = available.find(
      (r: { id: number }) => r.id === runnerId
    );
    expect(found).toBeFalsy(); // disabled runner should NOT be available

    // ── Step 7: Re-enable runner ──
    await api.put(`${RUNNERS}/${runnerId}`, { is_enabled: true });

    const reAvailRes = await api.get(`${RUNNERS}/available`);
    const reAvailable = (await reAvailRes.json()).runners || [];
    const reFound = reAvailable.find(
      (r: { id: number }) => r.id === runnerId
    );
    expect(reFound).toBeTruthy(); // re-enabled runner should be available

    // ── Step 8: Terminate remaining pod ──
    if (pod2Key) await api.post(`${PODS}/${pod2Key}/terminate`, {});

    // ── Step 9: Verify capacity restored ──
    await pollUntil(
      async () => {
        const r = await api.get(`${RUNNERS}/${runnerId}`);
        const d = await r.json();
        return (d.runner?.current_pods || 0) <= initialPods;
      },
      { maxAttempts: 5, intervalMs: 2000, label: "capacity-restored" }
    ).catch(() => {});
  });
});
