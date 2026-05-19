import { test, expect } from "../../fixtures/index";
import { CLEANUP, uniqueEmail, HASH_DEVPASS123 } from "../../helpers/test-data";
import { clearAuthRateLimit } from "../../helpers/redis";

/**
 * Full forgot/reset password flow against the light-auth REST endpoints.
 * Mirrors what a user would experience: request reset email, click the
 * link (we read the token from the DB instead), set a new password, log
 * back in with the new password.
 *
 * Specifically guards against:
 *   - lightForgotPassword/lightResetPassword payload shape regressions
 *   - the reset-password page not picking up `?token=` correctly
 *   - the new password not actually being applied
 */
test.describe("Forgot/reset password (light-auth)", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("forgot → reset via UI → login with new password", async ({ page, db, api }) => {
    const email = uniqueEmail("reset-flow");
    // Username must be unique in DB even if email is. Derive from the
    // unique-email prefix and strip any chars the username regex rejects
    // ([a-zA-Z0-9_-]+).
    const username = email.split("@")[0].replace(/[^a-zA-Z0-9_-]/g, "").slice(0, 50);
    const oldPassword = "devpass123";
    const newPassword = "NewPass456!";

    // Seed user directly so we know exactly what email we're targeting and
    // can clean up deterministically.
    db.setup(`
      INSERT INTO users (email, username, password_hash, name, is_email_verified, created_at, updated_at)
      VALUES ('${email}', '${username}', '${HASH_DEVPASS123}', 'Reset Flow', true, NOW(), NOW())
    `);

    try {
      const forgotRes = await api.postPublic("/api/v1/auth/forgot-password", { email });
      // Backend masks success/failure to prevent email enumeration; either way it's 200.
      expect(forgotRes.status).toBe(200);

      const token = db.queryValue(`SELECT password_reset_token FROM users WHERE email = '${email}'`);
      expect(token, "reset token should be persisted to users.password_reset_token").toBeTruthy();

      await page.goto(`/reset-password?token=${token}`);
      await page.locator("#password").fill(newPassword);
      await page.locator("#confirmPassword").fill(newPassword);
      await page.locator('button[type="submit"]').click();

      // Reset page schedules a router.push("/login") after a 2s success delay.
      await page.waitForURL((url) => url.pathname.includes("/login"), { timeout: 10_000 });

      // Old password rejected, new password accepted — both via API to keep this
      // test focused on credential change rather than UI re-runs.
      const failed = await api.postPublic("/api/v1/auth/login", { email, password: oldPassword });
      expect(failed.status, "old password must no longer work").toBe(401);

      const ok = await api.postPublic("/api/v1/auth/login", { email, password: newPassword });
      expect(ok.status, "new password must authenticate").toBe(200);
    } finally {
      db.cleanup(CLEANUP.userByEmail(email));
    }
  });

  test("reset-password page with missing token shows the error state", async ({ page }) => {
    await page.goto("/reset-password");
    await expect(page.getByText(/invalid reset link|reset link is invalid or missing/i).first()).toBeVisible();
  });
});
