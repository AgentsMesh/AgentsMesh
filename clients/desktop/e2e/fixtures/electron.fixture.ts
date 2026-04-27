import { test as base, _electron as electron, type ElectronApplication, type Page } from "@playwright/test";
import {
  getApiBaseUrl,
  getElectronMainPath,
  getAuthFile,
  getUserDataDir,
  isCi,
} from "../helpers/env";
import { loadStorageFile, restoreStorage } from "../helpers/storage-state";

export interface ElectronFixtures {
  electronApp: ElectronApplication;
  /** The Electron main window as a Playwright Page. */
  page: Page;
  /** Path of the saved auth storage (may not exist in setup project). */
  authFile: string;
  /** Skip applying saved localStorage (use for fresh-login tests). */
  skipAuthRestore: boolean;
  /** Electron userData directory — isolated for tests. */
  userDataDir: string;
}

/**
 * Launch Electron with AGENTSMESH_API_URL + NODE_ENV=test + isolated userData.
 * If a saved storage snapshot exists (from global.setup.ts), inject it into localStorage.
 */
export const test = base.extend<ElectronFixtures>({
  authFile: async ({}, use) => {
    await use(getAuthFile());
  },

  userDataDir: async ({}, use) => {
    await use(getUserDataDir());
  },

  skipAuthRestore: [false, { option: true }],

  electronApp: async ({ userDataDir }, use) => {
    // Linux CI Electron needs `--no-sandbox` (no suid-sandbox helper)
    // and `--disable-dev-shm-usage` (tiny /dev/shm tmpfs on the runner).
    const ciArgs = isCi() && process.platform === "linux"
      ? ["--no-sandbox", "--disable-dev-shm-usage"]
      : [];
    const app = await electron.launch({
      args: [getElectronMainPath(), `--user-data-dir=${userDataDir}`, ...ciArgs],
      env: {
        ...process.env,
        AGENTSMESH_API_URL: getApiBaseUrl(),
        NODE_ENV: "test",
        ELECTRON_DISABLE_SECURITY_WARNINGS: "true",
      },
      timeout: isCi() ? 120_000 : 30_000,
    });
    await use(app);
    await app.close().catch(() => undefined);
  },

  page: async ({ electronApp, authFile, skipAuthRestore }, use) => {
    const page = await electronApp.firstWindow();
    await page.waitForLoadState("domcontentloaded");

    if (!skipAuthRestore) {
      const snap = loadStorageFile(authFile);
      if (snap) {
        await restoreStorage(page, snap);
        await page.reload().catch(() => undefined);
        await page.waitForLoadState("domcontentloaded");
      }
    }

    await use(page);
  },
});

export { expect } from "@playwright/test";
