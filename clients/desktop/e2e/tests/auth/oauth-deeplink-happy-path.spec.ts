import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG } from "../../helpers/env";
import { invokeIpc } from "../../helpers/ipc";
import { resolve } from "node:path";
import { rmSync } from "node:fs";

// Fresh profile so the OAuth callback runs against a clean session — any
// bleed-through from a prior login would confuse the assertion ("did we
// reach dashboard because OAuth worked, or because the cached session
// was already there?").
const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-oauth-happy");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

test.describe("Auth · OAuth deep link · happy path", () => {
  // The negative-path specs (oauth-deep-link, oauth-callback-no-placeholder)
  // both feed bogus tokens and expect failure cleanup. They could not catch
  // the v0.31.x bug where `setAuth` only wrote the renderer cache and never
  // pushed the token to the main-process Rust AuthManager — both "bogus
  // token + bug" and "valid token + bug" produce the same auth_expired
  // observable. This spec drives the deep-link with a REAL token obtained
  // via authLogin and asserts the post-OAuth state is dashboard, not error.
  test("real-token deep link completes login without auth_expired", async ({
    page,
    electronApp,
  }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();

    // 1. Get a real token by driving the main-process login directly. Using
    //    IPC instead of the UI keeps this spec orthogonal to LoginPage's
    //    SSO-discovery + form-submit choreography (separate test surface).
    const loginJson = await invokeIpc<string>(
      page,
      "authLogin",
      TEST_USER.email,
      TEST_USER.password,
    );
    const session = JSON.parse(loginJson) as { token: string; refresh_token: string };
    expect(session.token, "authLogin must return a JWT").toBeTruthy();
    expect(session.refresh_token, "authLogin must return a refresh token").toBeTruthy();

    // 2. Clear the main-process session so the next OAuth callback can't
    //    succeed by accident — userGetMe must specifically depend on the
    //    token that setAuth → authApplySession just installed, NOT on any
    //    state left behind by step 1.
    await invokeIpc(page, "authClearSession");

    // Sanity: after clear, userGetMe MUST fail with auth_expired. If this
    // fails it means the test environment is broken (token cached
    // somewhere unexpected) and the next assertion would be meaningless.
    const meAfterClearError = await page.evaluate(async () => {
      const api = (window as unknown as { electronAPI: { invoke: (c: string) => Promise<unknown> } }).electronAPI;
      try {
        await api.invoke("userGetMe");
        return null;
      } catch (e) {
        return e instanceof Error ? e.message : String(e);
      }
    });
    expect(meAfterClearError, "userGetMe must fail after authClearSession").toContain("auth_expired");

    // 3. Inject the deep-link as main process would after open-url /
    //    second-instance. Renderer subscribes to `oauth:callback` in
    //    main.tsx and navigates to /auth/callback?token=...
    await electronApp.evaluate(({ BrowserWindow }, payload) => {
      const win = BrowserWindow.getAllWindows()[0];
      if (!win) throw new Error("no BrowserWindow");
      win.webContents.send("oauth:callback", payload);
    }, `agentsmesh://oauth/callback?token=${session.token}&refresh_token=${session.refresh_token}`);

    // 4. The OAuthCallbackPage runs setAuth → userGetMe → setOrganizations →
    //    redirect. Wait for it to leave the callback route — either toward
    //    workspace (org exists) or onboarding (no org). Failure shape: hash
    //    stays on /auth/callback while an error banner says "Sign in failed".
    await page.waitForFunction(
      (slug) => {
        const h = window.location.hash;
        return h.includes(`/${slug}/`) || h.includes("/workspace") || h.includes("/onboarding");
      },
      TEST_ORG_SLUG,
      { timeout: 20_000 },
    );

    const hash = await page.evaluate(() => window.location.hash);
    expect(hash, "should leave /auth/callback after successful login").not.toContain("/auth/callback");
    expect(hash, "should not bounce back to /login").not.toContain("/login");

    // 5. Direct assertion on the bug: userGetMe must succeed now that
    //    setAuth completed. If apply_session forgets to fan out
    //    authApplySession IPC, this fails the same way the production
    //    "Sign in failed · auth_expired" UI does. IPC userGetMe returns
    //    a serialized User directly (Rust ApiClient unwraps the backend's
    //    `{user: ...}` envelope in get_resource), so JSON.parse yields
    //    the bare user object.
    const meJson = await invokeIpc<string>(page, "userGetMe");
    const me = JSON.parse(meJson) as { email: string };
    expect(me.email).toBe(TEST_USER.email);
  });
});
