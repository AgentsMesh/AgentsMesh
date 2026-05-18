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
  // Worst-case CI budget: electron.launch 240s + firstWindow 90s +
  // login/redirect 90s + small overhead. Playwright's global 180s
  // (playwright.config.ts) is per-test and would trip before the
  // bumped launch+window timeouts could ever apply. Override locally
  // so other specs keep the tighter default.
  if (isCi()) setup.setTimeout(480_000);

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
    // macmini-03 cold-starts the Electron renderer in 30-60s under normal
    // load, but the shared CI box drifts to load avg 5-8 when dev residue
    // accumulates (Simulators, Chrome). 120s was tripping electron.launch
    // before firstWindow could even open. Match the firstWindow 90s bump
    // + headroom for the Electron process to spawn.
    timeout: isCi() ? 240_000 : 30_000,
  });

  try {
    // See electron.fixture.ts: macmini-03 cold-starts the renderer in
    // 30-60s; the default 30s firstWindow timeout was tripping the
    // global setup before any spec even ran.
    const page = await app.firstWindow({ timeout: isCi() ? 90_000 : 30_000 });
    page.on("pageerror", (err) => console.log(`[pageerror]`, err.message));
    await page.waitForLoadState("domcontentloaded");

    // Pin renderer locale to English before login so the captured
    // localStorage snapshot bakes `app_locale=en` in. macmini-04 (and
    // most CI hosts) default to non-English system locales; without this
    // pin, every spec's role-by-name locator matching English strings
    // (e.g. "Email" on the login form) trips 30s timeouts on the
    // translated variant. The reload makes IntlProvider re-read this on
    // mount before the login UI renders.
    await page.evaluate(() => localStorage.setItem("app_locale", "en"));
    await page.reload();
    await page.waitForLoadState("domcontentloaded");

    await page.waitForSelector("input#email", { timeout: 30_000 });
    await page.fill("input#email", TEST_USER.email);
    await page.fill("input#password", TEST_USER.password);
    await page.click('button[type="submit"]');

    await expectHashMatches(
      page,
      new RegExp(`/${TEST_ORG_SLUG}/|/onboarding|/workspace`),
      // macmini-03 cold-starts the renderer + backs the login request through
      // the full docker stack — same reason firstWindow above bumps to 90s on
      // CI. The 30s post-login redirect ceiling tripped TICKET-145's PR twice
      // before any spec ran (login succeeded; renderer just hadn't hydrated +
      // routed yet). Use the same CI bump.
      isCi() ? 90_000 : 30_000,
    );

    const snap = await captureStorage(page);
    saveStorageFile(getAuthFile(), snap);
  } finally {
    await app.close().catch(() => undefined);
  }
});
