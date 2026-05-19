import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";

/**
 * Light-auth coverage for /runners/authorize — the page that the runner
 * daemon opens (Tailscale-style) for a human to authorize a pending
 * registration. The page is anonymous-accessible: it always polls
 * auth-status, then renders either the authenticated form, the sign-in
 * prompt, or an error screen depending on what comes back.
 *
 * We verify the three failure surfaces the page renders before the user
 * even touches it, because the happy path (authorize) is already
 * exercised by the runner-tokens API tests.
 */
test.describe("Runners authorize page (light-auth)", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async () => { clearAuthRateLimit(); });

  test("missing key shows the error screen with a sign-in CTA", async ({ page }) => {
    await page.goto("/runners/authorize");
    // ErrorScreen renders a "Sign In" button linking back to /login;
    // matches both English copy and the i18n fallback.
    await expect(page.getByRole("link", { name: /sign in|login|登录/i })).toBeVisible();
  });

  test("invalid key cannot load auth-status — surfaces the error screen", async ({ page }) => {
    await page.goto("/runners/authorize?key=this-key-does-not-exist");
    await expect(page.getByRole("link", { name: /sign in|login|登录/i })).toBeVisible();
  });

  test("anonymous visit with a real pending key shows the sign-in prompt", async ({ page, api }) => {
    // Anonymous endpoint — the runner daemon hits this with no JWT, so we
    // intentionally use postPublic (no Authorization header).
    const res = await api.postPublic("/api/v1/runners/grpc/auth-url", {
      machine_key: "test-machine-key-" + Date.now(),
      node_id: "test-node-" + Date.now(),
      labels: { source: "e2e-light-auth" },
    });
    expect(res.status).toBe(200);
    const { auth_key } = await res.json();
    expect(auth_key).toBeTruthy();

    await page.goto(`/runners/authorize?key=${auth_key}`);
    // Anonymous AuthForm branch renders a "Sign in to authorize" button.
    // The exact i18n key is runners.authorize.signInToAuthorize.
    await expect(page.getByRole("link", { name: /sign in/i }).first()).toBeVisible();
  });
});
