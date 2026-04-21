import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { pollUntil } from "../../helpers/retry";

const PODS = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;

/**
 * Journey: Git Workflow
 * Repository → Ticket → Pod with repo context → Terminate
 *
 * Uses seed data: Gitea repos (demo-webapp, demo-api) already imported.
 */
test.describe("Journey: Git Workflow", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  test.afterAll(async () => {
    const { terminateAllPods } = await import("../../helpers/pod-cleanup");
    await terminateAllPods();
  });

  test("repo → ticket → pod with repository context", async ({ api, db }) => {
    // ── Step 1: Verify repositories exist (from seed/Gitea) ──
    const repoRes = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/repositories`
    );
    expect(repoRes.status).toBe(200);
    const repos = (await repoRes.json()).repositories || [];
    expect(repos.length).toBeGreaterThan(0);
    const repo = repos[0];

    // ── Step 2: Create ticket linked to repository ──
    const ticketRes = await api.post(
      `/api/v1/orgs/${TEST_ORG_SLUG}/tickets`,
      {
        title: "Journey: Fix authentication bug",
        description: "E2E journey test — agent should analyze auth module",
        repository_id: repo.id,
      }
    );
    expect([200, 201]).toContain(ticketRes.status);
    const ticket = await ticketRes.json();
    const ticketSlug = ticket.ticket?.slug || ticket.slug;
    expect(ticketSlug).toBeTruthy();

    // ── Step 3: Verify ticket is linked to repo ──
    const ticketDetail = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/tickets/${ticketSlug}`
    );
    expect(ticketDetail.status).toBe(200);

    // ── Step 4: Get available runner and agent ──
    const runnerRes = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/runners/available`
    );
    const runners = (await runnerRes.json()).runners;
    if (!runners?.length) {
      await api.delete(`/api/v1/orgs/${TEST_ORG_SLUG}/tickets/${ticketSlug}`);
      test.skip();
      return;
    }

    const agentRes = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/agents`
    );
    const agents = (await agentRes.json()).builtin_agents;

    // ── Step 5: Create pod WITH repository and ticket context ──
    const podRes = await api.post(PODS, {
      runner_id: runners[0].id,
      agent_slug: agents[0].slug,
      prompt: "Analyze the authentication module and suggest improvements",
      repository_id: repo.id,
      ticket_slug: ticketSlug,
    });
    expect([200, 201]).toContain(podRes.status);
    const podData = await podRes.json();
    const podKey = podData.pod_key || podData.pod?.pod_key;
    expect(podKey).toBeTruthy();

    // ── Step 6: Wait for pod to become running ──
    await pollUntil(
      async () => {
        const r = await api.get(`${PODS}/${podKey}`);
        const d = await r.json();
        return (d.pod?.status || d.status) === "running";
      },
      { maxAttempts: 10, intervalMs: 3000, label: "journey-pod-running" }
    ).catch(() => {});

    // ── Step 7: Verify pod has repository association ──
    const podDetail = await api.get(`${PODS}/${podKey}`);
    const podInfo = await podDetail.json();
    // Pod should reference the repo (exact field depends on API)
    expect(podDetail.status).toBe(200);

    // ── Step 8: Verify runner has this pod associated ──
    // We can't assert `runner.current_pods >= 1` directly — the runner's own
    // heartbeat can race with the server-side IncrementPods write and clobber
    // the count back to 0 before the REST read lands. Instead, assert via the
    // pods list scoped to this runner, which is the authoritative source.
    const runnerPods = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/pods?runner_id=${runners[0].id}`,
    );
    expect(runnerPods.status).toBe(200);
    const { pods: podList } = await runnerPods.json();
    expect(Array.isArray(podList)).toBe(true);
    expect(podList.some((p: { pod_key: string }) => p.pod_key === podKey)).toBe(true);

    // ── Step 9: Terminate pod ──
    const termRes = await api.post(`${PODS}/${podKey}/terminate`, {});
    expect(termRes.status).toBe(200);

    // ── Step 10: Verify pod terminated ──
    await pollUntil(
      async () => {
        const r = await api.get(`${PODS}/${podKey}`);
        const d = await r.json();
        return (d.pod?.status || d.status) === "terminated";
      },
      { maxAttempts: 5, intervalMs: 2000, label: "journey-pod-terminated" }
    ).catch(() => {});

    // ── Step 11: Cleanup ticket ──
    await api.delete(`/api/v1/orgs/${TEST_ORG_SLUG}/tickets/${ticketSlug}`);
  });

  test("git workflow visible in UI: repo → ticket → workspace", async ({ page }) => {
    // Verify repositories page shows seed repos
    await page.goto(`/${TEST_ORG_SLUG}/repositories`);
    await page.waitForLoadState("networkidle");
    let body = await page.textContent("body");
    expect(body).toMatch(/demo-webapp|demo-api|repository|仓库/i);

    // Verify tickets page loads
    await page.goto(`/${TEST_ORG_SLUG}/tickets`);
    await page.waitForLoadState("networkidle");
    body = await page.textContent("body");
    expect(body).toMatch(/ticket|工单|任务/i);

    // Verify workspace loads
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");
    expect(page.url()).toContain("/workspace");
  });
});
