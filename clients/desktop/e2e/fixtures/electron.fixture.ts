import { test as base, _electron as electron, type ElectronApplication, type Page } from "@playwright/test";
import {
  getApiBaseUrl,
  getElectronMainPath,
  getAuthFile,
  getUserDataDir,
  isCi,
} from "../helpers/env";
import { invokeIpc } from "../helpers/ipc";
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
    // firstWindow() defaults to playwright's 30s timeout, but the
    // macmini-03 self-hosted runner cold-starts the Electron renderer
    // in 30-60s under load (Bazel cache restore + electron-vite bundle
    // resolution + Rust core init via NAPI). Match the launch timeout
    // window — repeated CI failures on main were all this one assertion.
    const page = await electronApp.firstWindow({ timeout: isCi() ? 90_000 : 30_000 });
    await page.waitForLoadState("domcontentloaded");

    const snap = skipAuthRestore ? null : loadStorageFile(authFile);
    if (snap) await restoreStorage(page, snap);

    // Pin renderer locale to English AFTER any restoreStorage call so the
    // snapshot can't override it. macmini-04 (and most CI hosts) default
    // to non-English system locales that IntlProvider picks up via
    // navigator.language — turning role-by-name locators matching English
    // strings (e.g. /^add runner$/i) into 30s misses on translated
    // variants (e.g. "添加 Runner"). The reload below makes IntlProvider
    // re-read localStorage on mount and stick to English for the session.
    await page.evaluate(() => localStorage.setItem("app_locale", "en"));

    await page.reload().catch(() => undefined);
    await page.waitForLoadState("domcontentloaded");

    if (snap) {
      // Restore main-process Rust auth state from the disk-persisted
      // session global.setup wrote on login. Rust core intentionally
      // does not auto-bootstrap (see node-bridge/lib.rs:61); without
      // this call ApiClient.org_path() returns non-org URLs and every
      // org-scoped IPC (channel/runner/autopilot/...) 404s with
      // ResourceNotFound { resource: "resource", id: null }. Failures
      // here surface immediately rather than producing mysterious IPC
      // errors deep inside individual tests.
      await invokeIpc(page, "authBootstrap");
    }

    await use(page);
  },
});

export { expect } from "@playwright/test";
