import { test, expect } from "../../fixtures/index";
import { SettingsNavPage } from "../../pages/settings/settings-nav.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

/**
 * Console error validation for ALL dashboard pages that use WASM API calls.
 * Ensures no JSON deserialization errors, missing fields, or fetch failures
 * appear in the browser console when navigating to each page.
 */
test.describe("All Pages Console Error Validation", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  function consoleErrorCollector(page: import("@playwright/test").Page) {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") errors.push(msg.text());
    });
    return errors;
  }

  function assertNoWasmErrors(errors: string[]) {
    const critical = errors.filter(
      (e) =>
        (e.includes("missing field") ||
        e.includes("is not valid JSON") ||
        e.includes("Failed to fetch board") ||
        e.includes("Failed to fetch topology") ||
        e.includes("Failed to load runner") ||
        e.includes("Failed to load repository") ||
        e.includes("Failed to load ticket")) &&
        !e.includes("Failed to load resource")
    );
    expect(critical).toHaveLength(0);
  }

  // ── Main Dashboard Pages ──

  test("workspace page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("tickets list page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/tickets`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("mesh page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/mesh`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("loops list page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/loops`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("runners list page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/runners`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("repositories list page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/repositories`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("channels page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/channels`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("blocks page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/blocks`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("infra page (repositories tab): no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=repositories`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  test("infra page (runners tab): no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=runners`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });

  // ── Personal Settings Tabs ──

  test("settings/personal/general: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "general");
    assertNoWasmErrors(errors);
  });

  test("settings/personal/git: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "git");
    assertNoWasmErrors(errors);
  });

  test("settings/personal/agents: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "agents");
    assertNoWasmErrors(errors);
  });

  test("settings/personal/agents/claude-code: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "agents/claude-code");
    assertNoWasmErrors(errors);
  });

  test("settings/personal/notifications: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("personal", "notifications");
    assertNoWasmErrors(errors);
  });

  // ── Organization Settings Tabs ──

  test("settings/organization/general: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "general");
    assertNoWasmErrors(errors);
  });

  test("settings/organization/members: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "members");
    assertNoWasmErrors(errors);
  });

  test("settings/organization/extensions: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "extensions");
    assertNoWasmErrors(errors);
  });

  test("settings/organization/api-keys: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "api-keys");
    assertNoWasmErrors(errors);
  });

  test("settings/organization/usage: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "usage");
    assertNoWasmErrors(errors);
  });

  test("settings/organization/billing: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "billing");
    assertNoWasmErrors(errors);
  });

  test("settings/organization/runners: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    const nav = new SettingsNavPage(page, TEST_ORG_SLUG);
    await nav.goto("organization", "runners");
    assertNoWasmErrors(errors);
  });

  // ── Support ──

  test("support page: no console errors", async ({ page }) => {
    const errors = consoleErrorCollector(page);
    await page.goto(`/${TEST_ORG_SLUG}/support`);
    await page.waitForLoadState("networkidle");
    assertNoWasmErrors(errors);
  });
});
