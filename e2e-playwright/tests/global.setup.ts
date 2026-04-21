import { test as setup } from "@playwright/test";
import { TEST_USER, getWebBaseUrl } from "../helpers/env";
import { clearAuthRateLimit } from "../helpers/redis";
import { terminateAllPods } from "../helpers/pod-cleanup";

const AUTH_FILE = ".auth/user.json";

/**
 * Global setup: clean leftover pods, clear rate limits, authenticate.
 */
setup("authenticate as test user", async ({ browser }) => {
  const cleaned = await terminateAllPods();
  if (cleaned > 0) console.log(`[setup] Terminated ${cleaned} leftover pods`);
  clearAuthRateLimit();

  const context = await browser.newContext();
  const page = await context.newPage();

  const attempt = async (timeoutMs: number) => {
    await page.goto(`${getWebBaseUrl()}/login`, { waitUntil: "domcontentloaded" });
    await page.locator("#email").waitFor({ state: "visible", timeout: timeoutMs });
    await page.locator("#email").fill(TEST_USER.email);
    await page.locator("#password").fill(TEST_USER.password);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes("/login"), { timeout: timeoutMs });
  };

  try {
    await attempt(15_000);
  } catch {
    clearAuthRateLimit();
    await attempt(30_000);
  }

  await context.storageState({ path: AUTH_FILE });
  await context.close();
});
