import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG, getApiBaseUrl, getElectronMainPath, isCi } from "../../helpers/env";
import { resolve, join } from "node:path";
import { rmSync, existsSync, readdirSync } from "node:fs";
import { _electron as electron } from "@playwright/test";

const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-server-switch");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test.describe("Auth · server switch clears session", () => {
  test("changing active backend restarts Electron into /login", async ({
    page,
    electronApp,
  }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await login.login(TEST_USER.email, TEST_USER.password);
    await login.waitForLoginRedirect(TEST_ORG_SLUG);

    const userData = await electronApp.evaluate(({ app }) => app.getPath("userData")) as string;
    const namespaceRoot = join(userData, "agentsmesh", "agentsmesh-auth");
    expect(existsSync(namespaceRoot)).toBe(true);
    const namespacesBefore = readdirSync(namespaceRoot);
    expect(namespacesBefore.length).toBeGreaterThan(0);
    const previousNamespace = namespacesBefore[0];

    await electronApp.close().catch(() => undefined);

    // Relaunch with a DIFFERENT base_url. Same userDataDir → the previous
    // namespace's session file is still on disk, but the new manager
    // reads its own (empty) namespace and falls through to anonymous.
    const ciArgs = isCi() && process.platform === "linux"
      ? ["--no-sandbox", "--disable-dev-shm-usage"]
      : [];
    const app2 = await electron.launch({
      args: [getElectronMainPath(), `--user-data-dir=${FRESH_USER_DATA}`, ...ciArgs],
      env: {
        ...process.env,
        AGENTSMESH_API_URL: "https://other-host.invalid",
        NODE_ENV: "test",
        ELECTRON_DISABLE_SECURITY_WARNINGS: "true",
      },
      timeout: isCi() ? 120_000 : 30_000,
    });
    try {
      const page2 = await app2.firstWindow({ timeout: isCi() ? 90_000 : 30_000 });
      await page2.waitForLoadState("domcontentloaded");
      await page2.waitForTimeout(2000); // bootstrap settles

      const hash = await page2.evaluate(() => window.location.hash);
      expect(hash).toContain("/login");

      // Previous server's namespace file untouched (we only clean OUR namespace).
      expect(existsSync(join(namespaceRoot, previousNamespace, "session.json"))).toBe(true);
    } finally {
      await app2.close().catch(() => undefined);
    }
    // Restart unused getApiBaseUrl import — import kept for forward compat.
    void getApiBaseUrl;
  });
});
