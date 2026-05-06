import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { terminateAllPods } from "../../helpers/pod-cleanup";
import {
  collectConsoleErrors,
  collectPageErrors,
  assertNoWasmRecursiveBorrow,
} from "../../helpers/console-errors";

const PODS_BASE = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;

/**
 * Regression: opening a pod terminal triggered
 *   "recursive use of an object detected which would lead to unsafe aliasing in rust"
 * because every ACP-aware component synchronously called wasm `get_session_json`
 * during React render, racing the `&mut self` mutators driven by relay events.
 *
 * The fix moved wasm reads inside store mutators (SSOT cache); React render
 * only touches the cache. This spec asserts the page never surfaces that
 * exact wasm-bindgen borrow-check error.
 */
test.describe("ACP terminal: no wasm recursive borrow", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });
  test.afterEach(async () => { await terminateAllPods(); });

  test("workspace page load does not trigger wasm recursive borrow", async ({ page }) => {
    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");
    // Let any deferred ACP/relay subscriptions settle.
    await page.waitForTimeout(1500);

    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });

  test("creating a pod and rendering its terminal does not trigger wasm recursive borrow", async ({ page, api }) => {
    const runnersRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/runners/available`);
    const runners = (await runnersRes.json()).runners;
    if (!runners?.length) { test.skip(); return; }

    const agentsRes = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/agents`);
    const agents = (await agentsRes.json()).builtin_agents;
    if (!agents?.length) { test.skip(); return; }

    const consoleErrors = collectConsoleErrors(page);
    const pageErrors = collectPageErrors(page);

    const createRes = await api.post(PODS_BASE, {
      runner_id: runners[0].id,
      agent_slug: agents[0].slug,
      prompt: "E2E ACP recursive-borrow regression",
    });
    const data = await createRes.json();
    const podKey = data.pod_key || data.pod?.pod_key;
    expect(podKey, "pod creation must return a pod_key").toBeTruthy();

    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");

    // Give the relay time to push an ACP snapshot + at least one event so
    // the multi-hook AgentPanel render path actually exercises the cache.
    await page.waitForTimeout(4000);

    assertNoWasmRecursiveBorrow(consoleErrors);
    assertNoWasmRecursiveBorrow(pageErrors);
  });
});
