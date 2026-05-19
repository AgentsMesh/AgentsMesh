import { test, expect } from "../../fixtures/index";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG } from "../../helpers/env";

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

  test("wasm boots when navigating into the workspace", async ({ page }) => {
    const wasmRequests: string[] = [];
    page.on("request", (req) => {
      if (isWasmRequest(req.url())) wasmRequests.push(req.url());
    });

    const loginPage = new LoginPage(page);
    await loginPage.goto();
    expect(wasmRequests, "login itself must not pull wasm").toEqual([]);

    await loginPage.login(TEST_USER.email, TEST_USER.password);
    await page.waitForURL((url) => !url.pathname.includes("/login"), {
      timeout: 15_000,
    });
    // Give the dashboard layout a moment to lazy-import the wasm chunk.
    await page.waitForLoadState("networkidle");

    expect(
      wasmRequests.length,
      "dashboard layout must boot wasm on entry",
    ).toBeGreaterThan(0);
    expect(page.url()).toContain(`/${TEST_ORG_SLUG}`);
  });
});
