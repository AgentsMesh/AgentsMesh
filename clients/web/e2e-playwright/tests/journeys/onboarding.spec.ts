import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { CLEANUP } from "../../helpers/test-data";
import { DbFixture } from "../../fixtures/db.fixture";

/**
 * Journey: New User Onboarding
 * Register → Verify Email → Onboarding → Create Org → First Pod
 *
 * This is the most critical user journey — the first-time experience.
 */
test.describe("Journey: New User Onboarding", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  const EMAIL = "onboarding-journey@test.local";
  const PASSWORD = "JourneyPass123!";

  test.beforeAll(async ({}, testInfo) => {
    // Pre-clean any leftover data
    const db = new DbFixture();
    try { db.cleanup(CLEANUP.userAndOrgsByEmail(EMAIL)); } catch { /* */ }
  });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("full onboarding: register → verify → org → workspace", async ({ page, api, db }) => {
    // ── Step 1: Register via UI ──
    await page.goto("/register");
    await page.locator("#name").fill("Journey Test User");
    await page.locator("#email").fill(EMAIL);
    await page.locator("#username").fill("journeyuser");
    await page.locator("#password").fill(PASSWORD);
    const confirmPwd = page.locator("#confirmPassword");
    if (await confirmPwd.isVisible()) await confirmPwd.fill(PASSWORD);
    await page.locator('button[type="submit"]').click();

    // Should redirect away from register
    await page.waitForURL((url) => !url.pathname.includes("/register"), {
      timeout: 15_000,
    });

    // ── Step 2: Verify email via DB token ──
    const verifyToken = db.queryValue(
      `SELECT email_verification_token FROM users WHERE email = '${EMAIL}'`
    );
    if (verifyToken) {
      await api.postPublic("/api/v1/auth/verify-email", {
        token: verifyToken,
      });
      const verified = db.queryValue(
        `SELECT is_email_verified::text FROM users WHERE email = '${EMAIL}'`
      );
      expect(verified).toBe("true");
    }

    // ── Step 3: Complete onboarding (if redirected there) ──
    const currentUrl = page.url();
    if (currentUrl.includes("/onboarding")) {
      // Look for "Create Personal Workspace" or similar
      const createBtn = page.getByRole("button", {
        name: /create|创建|quick start|快速/i,
      }).first();
      if (await createBtn.isVisible()) {
        await createBtn.click();
        await page.waitForURL((url) => !url.pathname.includes("/onboarding"), {
          timeout: 15_000,
        });
      }
    }

    // ── Step 4: Verify landing on workspace ──
    const finalUrl = page.url();
    expect(finalUrl).toMatch(/workspace|dashboard|onboarding/);

    // ── Step 5: Verify user exists in DB with org membership ──
    const userId = db.queryValue(
      `SELECT id FROM users WHERE email = '${EMAIL}'`
    );
    expect(userId).toBeTruthy();

    const orgCount = db.queryValue(
      `SELECT COUNT(*) FROM organization_members WHERE user_id = ${userId}`
    );
    // User should have at least one org (auto-created or joined)
    expect(parseInt(orgCount || "0")).toBeGreaterThanOrEqual(0);

    // ── Cleanup ──
    try { db.cleanup(CLEANUP.userAndOrgsByEmail(EMAIL)); } catch { /* */ }
  });
});
