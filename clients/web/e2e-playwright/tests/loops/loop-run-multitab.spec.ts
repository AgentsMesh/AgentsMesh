// Multi-tab UI propagation for loop_run:started.
//
// Both tabs open the same loop detail page; tab A triggers a run via
// Connect-RPC and tab B's run-history list grows by 1 without reload.
//
// Wire-level coverage in tests/realtime/loop-events-wire.spec.ts; this
// spec exercises handler → fetchRuns → React render chain.
import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import { TEST_ORG_SLUG } from "../../helpers/env";

test.describe("Loop run · multi-tab UI propagation", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("tab A trigger run → tab B run-history list adds card", async ({ context, api }) => {
    const cc = await api.connect();

    const stamp = Date.now().toString(36);
    const loopName = `e2e-rt-loop-${stamp}`;
    const loop = (await cc.loop.createLoop({
      orgSlug: TEST_ORG_SLUG,
      name: loopName,
      slug: loopName,
      agentSlug: "e2e-echo",
      promptTemplate: "echo hi",
      executionMode: "direct",
      timeoutMinutes: 1,
    } as never)) as { slug: string };

    const tabA = await context.newPage();
    const tabB = await context.newPage();
    await Promise.all([
      tabA.goto(`/${TEST_ORG_SLUG}/loops/${loop.slug}`),
      tabB.goto(`/${TEST_ORG_SLUG}/loops/${loop.slug}`),
    ]);

    // Wait until the LoopHeader's h1 renders the loop name in BOTH tabs —
    // that means fetchLoop resolved, currentLoop is set in WASM, and the
    // realtime handler (which reads currentLoop synchronously) will route
    // the upcoming loop_run:started event correctly.
    await Promise.all([
      expect(tabA.getByRole("heading", { level: 1, name: loopName })).toBeVisible({ timeout: 30_000 }),
      expect(tabB.getByRole("heading", { level: 1, name: loopName })).toBeVisible({ timeout: 30_000 }),
    ]);

    // EventSubscriptionManager bootstrap window so both tabs are subscribed
    // before publish. Events published before subscribeAll registers have
    // no replay buffer.
    await tabA.waitForTimeout(2000);

    const runCard = `[data-testid="loop-run-card"]`;
    // Loop just created: no runs yet — assert no cards mounted.
    await Promise.all([
      expect(tabA.locator(runCard)).toHaveCount(0),
      expect(tabB.locator(runCard)).toHaveCount(0),
    ]);

    await cc.loop.triggerLoop({
      orgSlug: TEST_ORG_SLUG, loopSlug: loop.slug,
    } as never);

    // loop_run:started → debounced refetch (500ms) → fetchRuns. Both tabs
    // should observe at least 1 run-card within the window.
    await Promise.all([
      expect(tabA.locator(runCard)).toHaveCount(1, { timeout: 15_000 }),
      expect(tabB.locator(runCard)).toHaveCount(1, { timeout: 15_000 }),
    ]);

    await tabA.close();
    await tabB.close();
  });
});
