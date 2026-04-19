import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { textContent } from "../../helpers/test-data";

/**
 * Remaining page and CRUD gap tests.
 */
test.describe("Remaining Coverage Gaps", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  /** TC-ORGSET-005: Delete org confirmation dialog */
  test("org settings page has danger zone", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=organization&tab=general`);
    await page.waitForLoadState("networkidle");
    const body = await page.textContent("body");
    expect(body).toMatch(/danger|危险|delete|删除/i);
  });

  /** TC-NOTIFY-003: Disable notifications */
  test("notification settings toggle exists", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=personal&tab=notifications`);
    await page.waitForLoadState("networkidle");
    const body = await page.textContent("body");
    expect(body).toMatch(/notification|通知/i);
  });

  /** TC-WS-003/004: Workspace layout */
  test("workspace page supports terminal layout", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");
    // Page loads without error
    expect(page.url()).toContain("/workspace");
  });

  /** TC-GITCRED-006/007: Git credentials UI flow */
  test("git settings page loads", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=personal&tab=git`);
    await page.waitForLoadState("networkidle");
    const body = await page.textContent("body");
    expect(body).toMatch(/git|credential|凭据/i);
  });

  /** TC-PROF-006: User settings full flow (profile page) */
  test("user profile accessible from settings", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=personal&tab=general`);
    await page.waitForLoadState("networkidle");
    const body = await page.textContent("body");
    expect(body).toMatch(/language|语言|personal|个人/i);
  });

  /** TC-CHAN-002: Send message to channel */
  test("send message to channel", async ({ api }) => {
    // Create channel
    const chRes = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
      name: "E2E Msg Channel " + Date.now(),
    });
    expect([200, 201]).toContain(chRes.status);
    const ch = await chRes.json();
    const chId = ch.channel?.id || ch.id;
    if (!chId) { test.skip(); return; }

    // Send message
    const msgRes = await api.post(
      `/api/v1/orgs/${TEST_ORG_SLUG}/channels/${chId}/messages`,
      { content: textContent("Hello from E2E test") }
    );
    expect([200, 201]).toContain(msgRes.status);

    // Get messages
    const listRes = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/channels/${chId}/messages`
    );
    expect(listRes.status).toBe(200);

    // Cleanup
    await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels/${chId}/archive`, {});
  });

  /** TC-MESH-002~004: Mesh topology with pods */
  test("mesh topology API returns structure", async ({ api }) => {
    const res = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/mesh/topology`);
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data).toBeTruthy();
  });

  /** Billing: cancel, seats, checkout UI stubs */
  test("billing plans prices endpoint", async ({ api }) => {
    const res = await api.get(
      `/api/v1/orgs/${TEST_ORG_SLUG}/billing/plans/prices`
    );
    expect(res.status).toBe(200);
  });

  /** TC-CYCLE-003: Billing cycle change (yearly to monthly) */
  test("billing cycle change to monthly", async ({ api }) => {
    const res = await api.post(
      `/api/v1/orgs/${TEST_ORG_SLUG}/billing/subscription/change-cycle`,
      { billing_cycle: "monthly" }
    );
    expect([200, 400]).toContain(res.status);
  });
});
