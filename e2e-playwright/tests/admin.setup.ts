import { test as setup } from "@playwright/test";
import { ADMIN_USER, getWebBaseUrl } from "../helpers/env";
import { clearAuthRateLimit } from "../helpers/redis";

const ADMIN_AUTH_FILE = ".auth/admin.json";

/**
 * Global setup: logs in as the admin user and saves browser state.
 * Tests requiring admin auth use storageState: '.auth/admin.json'.
 *
 * The first render of /login in a cold webpack dev server can exceed the
 * default 15s wait window (WASM init + chunk compilation). We retry once
 * with a longer timeout to absorb that cold-start spike.
 */
setup("authenticate as admin user", async ({ browser }) => {
  clearAuthRateLimit();
  const context = await browser.newContext();
  const page = await context.newPage();

  const attempt = async (timeoutMs: number) => {
    await page.goto(`${getWebBaseUrl()}/login`, { waitUntil: "domcontentloaded" });
    await page.locator("#email").waitFor({ state: "visible", timeout: timeoutMs });
    await page.locator("#email").fill(ADMIN_USER.email);
    await page.locator("#password").fill(ADMIN_USER.password);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes("/login"), { timeout: timeoutMs });
  };

  try {
    await attempt(15_000);
  } catch {
    // Cold-boot flake: retry with a longer window after clearing rate-limit.
    clearAuthRateLimit();
    await attempt(30_000);
  }

  await context.storageState({ path: ADMIN_AUTH_FILE });
  await context.close();
});
