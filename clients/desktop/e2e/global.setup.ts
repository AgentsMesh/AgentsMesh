import { test as setup, _electron as electron } from "@playwright/test";
import { rmSync } from "node:fs";
import {
  getApiBaseUrl,
  getAuthFile,
  getElectronMainPath,
  getUserDataDir,
  TEST_USER,
  TEST_ORG_SLUG,
  isCi,
} from "./helpers/env";
import { expectHashMatches } from "./helpers/nav";
import { captureStorage, saveStorageFile } from "./helpers/storage-state";

setup("authenticate as test user (Electron)", async () => {
  // Reset userData so login always starts fresh — avoids leaking dev-profile session.
  const userDataDir = getUserDataDir();
  rmSync(userDataDir, { recursive: true, force: true });

  // Linux CI Electron needs `--no-sandbox` (the suid-sandbox helper
  // isn't installed) and `--disable-dev-shm-usage` (CI runners ship
  // a tiny tmpfs for /dev/shm — Chromium runs out of GPU shared
  // memory and aborts before BrowserWindow opens).
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

  try {
    const page = await app.firstWindow();
    page.on("pageerror", (err) => console.log(`[pageerror]`, err.message));
    await page.waitForLoadState("domcontentloaded");

    await page.waitForSelector("input#email", { timeout: 30_000 });
    await page.fill("input#email", TEST_USER.email);
    await page.fill("input#password", TEST_USER.password);
    await page.click('button[type="submit"]');

    await expectHashMatches(
      page,
      new RegExp(`/${TEST_ORG_SLUG}/|/onboarding|/workspace`),
      30_000
    );

    const snap = await captureStorage(page);
    saveStorageFile(getAuthFile(), snap);
  } finally {
    await app.close().catch(() => undefined);
  }
});
