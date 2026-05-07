import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { resolve } from "node:path";
import { rmSync } from "node:fs";

// Fresh profile so the GitHub button click never lands on a dashboard
// (would happen if a session restore from a previous test bled in).
const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-oauth-deeplink");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test.describe("Auth · OAuth deep link", () => {
  // The desktop GitHub button must hand the system browser a URL whose
  // `redirect=` parameter is the `agentsmesh://oauth/callback` scheme,
  // not a web URL. v0.30.x desktop shipped with the broken redirect
  // and the OAuth flow could never complete because the token landed
  // in the web app instead of the desktop app. This spec replaces
  // the `shellOpen` IPC handler with a spy and asserts the URL
  // structure when the user clicks GitHub.
  test("GitHub button opens system browser with deep-link redirect", async ({
    page, electronApp,
  }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();

    // Replace the production handler with one that records the URL.
    // The handler in main/index.ts shells out to `shell.openExternal`;
    // recording the arg is enough to prove the renderer asked for the
    // right URL — the OS-level `open` is verified separately.
    await electronApp.evaluate(({ ipcMain }) => {
      ipcMain.removeHandler("shellOpen");
      ipcMain.handle("shellOpen", (_e, url: string) => {
        (globalThis as Record<string, unknown>).__lastShellOpen = url;
      });
    });

    await page.getByRole("button", { name: /^GitHub$/ }).click();

    // Allow the dynamic `import("@/shims/electron-shell")` + IPC round-trip.
    await page.waitForFunction(
      () => Boolean((globalThis as Record<string, unknown>).__lastShellOpen),
      undefined,
      { timeout: 5000 },
    ).catch(() => undefined);

    const url = await electronApp.evaluate(() =>
      (globalThis as Record<string, unknown>).__lastShellOpen as string | undefined,
    );

    expect(url, "GitHub button should invoke shellOpen with an OAuth URL").toBeTruthy();
    expect(url).toContain("/api/v1/auth/oauth/github");
    // The redirect query param is URL-encoded, so we look for either
    // the encoded or decoded form to keep the spec resilient to small
    // formatting changes.
    expect(url).toMatch(/redirect=agentsmesh(%3A%2F%2F|:\/\/)oauth(%2F|\/)callback/);
  });

  // Once GitHub authorises, the backend 302's back to
  // `agentsmesh://oauth/callback?token=...&refresh_token=...`. macOS
  // (`open-url` event) and Windows (`second-instance` argv) both
  // funnel into the same `oauth:callback` IPC channel that the
  // renderer subscribes to in main.tsx. This spec emits that exact
  // event from the main process and asserts the renderer reroutes
  // to the in-app callback page with token/refresh_token preserved.
  test("oauth:callback IPC event navigates renderer to /auth/callback", async ({
    page, electronApp,
  }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();

    const TOKEN = "fake-token-abc-123";
    const REFRESH = "fake-refresh-xyz-789";

    // Emit the deep-link callback as the main process would after
    // open-url / second-instance triggers `webContents.send`.
    await electronApp.evaluate(({ BrowserWindow }, payload) => {
      const win = BrowserWindow.getAllWindows()[0];
      if (!win) throw new Error("no BrowserWindow");
      win.webContents.send("oauth:callback", payload);
    }, `agentsmesh://oauth/callback?token=${TOKEN}&refresh_token=${REFRESH}`);

    // hashRouter encodes the route in the URL hash. We don't care
    // whether the page can complete login (it can't — the token is
    // bogus, getMe will 401), only that the deep-link reaches the
    // existing /auth/callback page with token + refresh_token in the
    // query string. Reusing OAuthCallbackPage end-to-end means a
    // working real-token flow is one redirect away.
    await page.waitForFunction(
      () => window.location.hash.includes("/auth/callback"),
      undefined,
      { timeout: 5000 },
    );

    const hash = await page.evaluate(() => window.location.hash);
    expect(hash).toContain(`token=${TOKEN}`);
    expect(hash).toContain(`refresh_token=${REFRESH}`);
  });
});
