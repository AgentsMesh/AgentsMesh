import { test, expect } from "../../fixtures/index";
import { TEST_USER, TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

/**
 * OAuth / SSO callback pages are entirely light-auth driven: they pluck
 * `token` + `refresh_token` off the URL query (backend hands them over
 * via a 307 redirect after exchanging the provider code), persist them
 * into localStorage in the Rust-compatible PersistedSession shape, then
 * let resolvePostLoginUrlLight pick the destination.
 *
 * We can't drive a real provider in E2E, but we can mint a real token
 * pair via the password login endpoint and feed it back through the
 * callback URL — that exercises the same code path the GitHub / Google
 * / SAML / OIDC flows take after provider exchange.
 */
test.describe("OAuth / SSO callback (light-auth)", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  for (const callbackPath of ["/auth/callback", "/auth/sso/callback"]) {
    test(`${callbackPath} with real token pair lands on dashboard`, async ({ page, api }) => {
      const tokens = await api.login(TEST_USER.email, TEST_USER.password);
      const url = `${callbackPath}?token=${encodeURIComponent(tokens.token)}&refresh_token=${encodeURIComponent(tokens.refresh_token)}`;

      await page.goto(url);
      await page.waitForURL(
        (u) => u.pathname.startsWith(`/${TEST_ORG_SLUG}`) || u.pathname === "/onboarding",
        { timeout: 15_000 },
      );

      // PersistedSession must be written in the Rust-compatible shape so
      // dashboard bootstrap can read it back.
      const sessionBlob = await page.evaluate(() => {
        const entry = Object.entries(localStorage).find(([k]) =>
          k.startsWith("agentsmesh-auth/") && k.endsWith("/session"));
        return entry ? entry[1] : null;
      });
      expect(sessionBlob, "PersistedSession blob must exist after callback").toBeTruthy();
      const parsed = JSON.parse(sessionBlob!);
      expect(parsed.schema_version).toBe(1);
      expect(parsed.access_token).toBe(tokens.token);
      expect(parsed.refresh_token).toBe(tokens.refresh_token);
      expect(parsed.expires_at).toBeGreaterThan(Math.floor(Date.now() / 1000));
    });
  }

  test("callback with error= shows error UI without writing a session", async ({ page }) => {
    await page.goto("/auth/callback?error=access_denied");
    await expect(page.getByText(/sign in failed|cancelled the authorization request/i).first()).toBeVisible();

    const sessionBlob = await page.evaluate(() => {
      const entry = Object.entries(localStorage).find(([k]) =>
        k.startsWith("agentsmesh-auth/") && k.endsWith("/session"));
      return entry ? entry[1] : null;
    });
    expect(sessionBlob, "error callback must not write a session").toBeNull();
  });

  test("callback with no token at all shows missing-token error", async ({ page }) => {
    await page.goto("/auth/callback");
    await expect(page.getByText(/authentication token is missing|sign in failed/i).first()).toBeVisible();
  });
});
