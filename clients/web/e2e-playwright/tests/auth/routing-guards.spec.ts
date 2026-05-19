import { test, expect, type Page } from "../../fixtures/index";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG, getWebBaseUrl } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// Routing guards exercised here mirror the three real-world flows:
//   E1: ?redirect= survives the login round-trip (deep-link restore).
//   E2: useRedirectIfAuthenticated on (auth) pages — already-signed-in
//       users never see the login/register/forgot-password forms.
//   E3: useRequireLightAuth on protected (auth) pages — anonymous users
//       get bounced to /login?redirect=<here>.
//
// The (auth) layout loads wasm, but the guard logic itself is light-auth
// driven (reads PersistedSession out of localStorage) so we can seed a
// session via page.addInitScript without booting wasm at all for the E2
// "already authenticated" cases.

// Mirrors lib/light-session.ts::urlSlug (cross-language SSOT pin to Rust
// crates/auth/src/state.rs::url_slug). Must stay byte-equal — bootstrap
// rejects mismatched slugs and clears the session.
async function seedLightSession(
  page: Page,
  tokens: { token: string; refresh_token: string },
): Promise<void> {
  const baseUrl = getWebBaseUrl();
  const expiresAt = Math.floor(Date.now() / 1000) + 3600;
  await page.addInitScript(
    ({ tokens, baseUrl, expiresAt }) => {
      const blob = {
        access_token: tokens.token,
        refresh_token: tokens.refresh_token,
        expires_at: expiresAt,
        base_url: baseUrl,
        current_org_slug: null,
        schema_version: 1,
      };
      const u = new URL(baseUrl);
      const port = u.port ? `_${u.port}` : "";
      const raw = `${u.protocol.replace(":", "")}_${u.hostname.toLowerCase()}${port}`;
      const slug = raw.replace(/[^a-zA-Z0-9]/g, "_").slice(0, 64);
      window.localStorage.setItem(
        `agentsmesh-auth/${slug}/session`,
        JSON.stringify(blob),
      );
    },
    { tokens, baseUrl, expiresAt },
  );
}

test.describe("E1: ?redirect= preserves deep link through login", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("login with ?redirect= preserves deep link path", async ({ page }) => {
    const target = `/${TEST_ORG_SLUG}/workspace`;
    await page.goto(`/login?redirect=${encodeURIComponent(target)}`);

    const loginPage = new LoginPage(page);
    await loginPage.emailInput.waitFor({ state: "visible" });
    await loginPage.login(TEST_USER.email, TEST_USER.password);

    await page.waitForURL((url) => url.pathname !== "/login", { timeout: 15_000 });
    expect(page.url()).toContain(target);
  });
});

test.describe("E2: authenticated user is redirected away from (auth) pages", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  for (const path of ["/login", "/register", "/forgot-password"]) {
    test(`authenticated user visiting ${path} is redirected to dashboard`, async ({
      page,
      api,
    }) => {
      const tokens = await api.login(TEST_USER.email, TEST_USER.password);
      await seedLightSession(page, tokens);

      await page.goto(path);
      await page.waitForURL((url) => url.pathname !== path, { timeout: 15_000 });
      expect(page.url()).toContain(`/${TEST_ORG_SLUG}`);
    });
  }
});

test.describe("E3: anonymous user is bounced to /login with ?redirect=", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("unauthenticated /onboarding redirects to /login with ?redirect=/onboarding", async ({
    page,
  }) => {
    await page.goto("/onboarding");
    await page.waitForURL((url) => url.pathname === "/login", { timeout: 15_000 });
    const redirectParam = new URL(page.url()).searchParams.get("redirect");
    expect(redirectParam).toBe("/onboarding");
  });

  test("unauthenticated /runners/authorize?key= shows sign-in prompt (no redirect)", async ({
    page,
  }) => {
    // /runners/authorize never auto-redirects — its AuthForm renders an
    // explicit "Sign in to authorize" link for anonymous visitors so the
    // runner daemon flow stays on this URL until the user clicks through.
    await page.goto("/runners/authorize?key=test-key");
    expect(page.url()).toContain("/runners/authorize");
    expect(page.url()).not.toContain("/login");
    await expect(
      page.getByRole("link", { name: /sign in/i }).first(),
    ).toBeVisible();
  });
});
