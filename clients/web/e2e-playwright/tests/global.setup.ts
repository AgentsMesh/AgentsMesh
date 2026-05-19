import { test as setup } from "@playwright/test";
import { TEST_USER, getWebBaseUrl, getApiBaseUrl } from "../helpers/env";
import { clearAuthRateLimit } from "../helpers/redis";
import { terminateAllPods } from "../helpers/pod-cleanup";

const AUTH_FILE = ".auth/user.json";

setup("authenticate as test user", async ({ browser }) => {
  const cleaned = await terminateAllPods();
  if (cleaned > 0) console.log(`[setup] Terminated ${cleaned} leftover pods`);
  clearAuthRateLimit();

  // Skip the fragile UI form path — webpack dev's main-app.js takes a while
  // to hydrate under CI Docker, so locator.fill races against React
  // re-rendering the form. Mirror what the lightLogin code path does:
  // POST /auth/login, then inject the PersistedSession blob (Rust SSOT shape
  // defined in clients/core/crates/auth/src/state.rs) into localStorage via
  // addInitScript so the dashboard's wasm bootstrap can hydrate.
  const apiBaseUrl = getApiBaseUrl();
  const loginRes = await fetch(`${apiBaseUrl}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: TEST_USER.email, password: TEST_USER.password }),
  });
  if (!loginRes.ok) throw new Error(`login failed: ${loginRes.status}`);
  const { token, refresh_token, expires_in } = await loginRes.json();
  const baseUrl = getWebBaseUrl();
  const expiresAt = Math.floor(Date.now() / 1000) + (expires_in ?? 3600);

  const context = await browser.newContext();
  await context.addInitScript(
    ({ token, refresh_token, expiresAt, baseUrl }) => {
      // Mirrors lib/light-session.ts::urlSlug — must stay in sync with
      // Rust SSOT clients/core/crates/auth/src/state.rs::url_slug.
      const u = new URL(baseUrl);
      const port = u.port ? `_${u.port}` : "";
      const raw = `${u.protocol.replace(":", "")}_${u.hostname.toLowerCase()}${port}`;
      const slug = raw.replace(/[^a-zA-Z0-9]/g, "_").slice(0, 64);
      const blob = {
        access_token: token,
        refresh_token,
        expires_at: expiresAt,
        base_url: baseUrl,
        current_org_slug: null,
        schema_version: 1,
      };
      localStorage.setItem(`agentsmesh-auth/${slug}/session`, JSON.stringify(blob));
    },
    { token, refresh_token, expiresAt, baseUrl },
  );

  // Visit any URL to commit the addInitScript write into the page's localStorage,
  // then snapshot the storage state for downstream tests.
  const page = await context.newPage();
  await page.goto(baseUrl, { waitUntil: "domcontentloaded" });
  await context.storageState({ path: AUTH_FILE });
  await context.close();
});
