import { test, expect } from "../../fixtures/index";
import { TEST_USER, TEST_ORG_SLUG, getWebBaseUrl, getApiBaseUrl } from "../../helpers/env";

/**
 * Regression guard for the light-auth rollout:
 *   /login, /register, /forgot-password, /reset-password, /verify-email,
 *   /onboarding, /invite/*, /runners/authorize and both OAuth callbacks
 *   must NEVER load the 40MB agentsmesh-wasm bundle.
 *
 * Wasm only kicks in once the user crosses into (dashboard).
 *
 * Static defenses (ESLint no-restricted-imports +
 * scripts/check-no-wasm-in-marketing.sh) catch import-graph regressions,
 * but a dynamic `import("@/lib/wasm-core")` would slip past both. This
 * spec watches the actual network requests and is the only layer that
 * catches that.
 */

function isWasmRequest(url: string): boolean {
  // Both the .wasm asset itself and the JS chunk wrapping agentsmesh-wasm
  // are indicators that wasm boot has started.
  return url.endsWith(".wasm") || /agentsmesh[-_]wasm/.test(url);
}

test.describe("Pre-dashboard routes are wasm-zero", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  for (const path of [
    "/login",
    "/register",
    "/forgot-password",
    "/onboarding",
    "/invite/some-token",
    "/runners/authorize",
    "/auth/callback",
    "/auth/sso/callback",
  ]) {
    test(`anonymous visit to ${path} does not request wasm`, async ({ page }) => {
      const wasmRequests: string[] = [];
      page.on("request", (req) => {
        const url = req.url();
        if (isWasmRequest(url)) wasmRequests.push(url);
      });

      await page.goto(path);
      await page.waitForLoadState("networkidle");

      expect(
        wasmRequests,
        `Expected zero wasm requests on ${path}; got:\n${wasmRequests.join("\n")}`,
      ).toEqual([]);
    });
  }
});

test.describe("Dashboard still loads wasm after login", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("wasm boots when navigating into the workspace", async ({ browser }) => {
    // Skip the UI form (CI webpack-dev races against fill). Mirror the
    // lightLogin code path: hit /auth/login, seed PersistedSession, then
    // measure what gets requested on the dashboard route.
    const apiBaseUrl = getApiBaseUrl();
    const loginRes = await fetch(`${apiBaseUrl}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: TEST_USER.email, password: TEST_USER.password }),
    });
    expect(loginRes.status).toBe(200);
    const { token, refresh_token, expires_in } = await loginRes.json();
    const baseUrl = getWebBaseUrl();
    const expiresAt = Math.floor(Date.now() / 1000) + (expires_in ?? 3600);

    const context = await browser.newContext();
    await context.addInitScript(
      ({ token, refresh_token, expiresAt, baseUrl }) => {
        const u = new URL(baseUrl);
        const port = u.port ? `_${u.port}` : "";
        const raw = `${u.protocol.replace(":", "")}_${u.hostname.toLowerCase()}${port}`;
        const slug = raw.replace(/[^a-zA-Z0-9]/g, "_").slice(0, 64);
        localStorage.setItem(
          `agentsmesh-auth/${slug}/session`,
          JSON.stringify({
            access_token: token,
            refresh_token,
            expires_at: expiresAt,
            base_url: baseUrl,
            current_org_slug: null,
            schema_version: 1,
          }),
        );
      },
      { token, refresh_token, expiresAt, baseUrl },
    );
    const page = await context.newPage();

    const wasmRequests: string[] = [];
    page.on("request", (req) => {
      if (isWasmRequest(req.url())) wasmRequests.push(req.url());
    });

    await page.goto(`${baseUrl}/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");

    expect(
      wasmRequests.length,
      "dashboard layout must boot wasm on entry",
    ).toBeGreaterThan(0);
    expect(page.url()).toContain(`/${TEST_ORG_SLUG}`);

    await context.close();
  });
});
