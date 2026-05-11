import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG, getApiBaseUrl, getElectronMainPath, isCi } from "../../helpers/env";
import { resolve } from "node:path";
import { rmSync } from "node:fs";
import { _electron as electron } from "@playwright/test";

// Verifies R3+R4: auto-restore must NOT bring back a logged-out account.
// In v0.31 logout cleared the in-memory token but left the storage file
// in place; on next launch `restore_session()` rehydrated it and the user
// landed back on the dashboard with a "ghost" session. The new bootstrap
// protocol cleans storage as part of logout AND treats missing / cleared
// session as Anonymous, so a restart must drop the user on /login.

const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-logout-restart");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test.describe("Auth · logout survives restart", () => {
  test("after explicit logout, restarting Electron lands on /login", async ({
    page,
    electronApp,
  }) => {
    // Step 1: log in fresh (skipAuthRestore disabled the snapshot path).
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await login.login(TEST_USER.email, TEST_USER.password);
    await login.waitForLoginRedirect(TEST_ORG_SLUG);

    // Step 2: trigger logout via IPC (the same path the SignOut button hits).
    await electronApp.evaluate(({ ipcMain }) => {
      // ipcMain.handle is registered by main/index.ts using napi method names
      // (snake → camel). authLogout invokes the Rust side which clears the
      // namespaced session file.
      const handler = (ipcMain as unknown as {
        _events?: Record<string, unknown>;
      })._events?.["authLogout"];
      // Just calling invoke from the renderer is the more faithful path —
      // skip the ipcMain peek and let the renderer's button drive it.
    });
    await page.evaluate(async () => {
      const api = (window as unknown as { electronAPI?: { invoke?: (ch: string, ...args: unknown[]) => Promise<unknown> } }).electronAPI;
      if (api?.invoke) await api.invoke("authLogout");
    });

    // Step 3: close Electron entirely.
    await electronApp.close().catch(() => undefined);

    // Step 4: relaunch with the SAME userDataDir. If logout was incomplete,
    // bootstrap would re-hydrate the token and routing would skip /login.
    const ciArgs = isCi() && process.platform === "linux"
      ? ["--no-sandbox", "--disable-dev-shm-usage"]
      : [];
    const app2 = await electron.launch({
      args: [getElectronMainPath(), `--user-data-dir=${FRESH_USER_DATA}`, ...ciArgs],
      env: {
        ...process.env,
        AGENTSMESH_API_URL: getApiBaseUrl(),
        NODE_ENV: "test",
        ELECTRON_DISABLE_SECURITY_WARNINGS: "true",
      },
      timeout: isCi() ? 120_000 : 30_000,
    });
    try {
      const page2 = await app2.firstWindow({ timeout: isCi() ? 90_000 : 30_000 });
      await page2.waitForLoadState("domcontentloaded");

      // Bootstrap is async; give it a beat to settle before asserting route.
      await page2.waitForFunction(
        () => Boolean((window as unknown as { __auth_hydrated?: boolean }).__auth_hydrated)
          || window.location.hash.includes("/login"),
        undefined,
        { timeout: 15_000 },
      ).catch(() => undefined);

      const hash = await page2.evaluate(() => window.location.hash);
      expect(hash).toContain("/login");
    } finally {
      await app2.close().catch(() => undefined);
    }
  });
});
