import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG, getApiBaseUrl } from "../../helpers/env";
import { resolve } from "node:path";
import { rmSync } from "node:fs";

// Fresh Electron profile every run — the spec writes to localStorage
// (the server config persists there), and we want a known starting
// point ("kind: cloud") on every invocation. Sharing userData with
// other auth specs would race the persisted choice across runs.
const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-server-settings");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

// End-to-end proof that picking "自定义服务器" and entering a URL in
// the Server Settings dialog actually drives the desktop client at
// the chosen origin afterwards. Without this spec a regression that
// breaks the `kind === "custom"` branch in env.ts would silently make
// every install hit the cloud default — invisible until a self-host
// user complains they "can't log in".
//
// The custom URL we save IS the dev backend (== AGENTSMESH_API_URL the
// fixture launches Electron with), so login works after the reload —
// proving the new origin is what the renderer picks up.

test.describe("Auth · server settings", () => {
  test("custom server: pick → save → reload → login goes to chosen URL", async ({ page }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();

    // Bottom-left of the hero pane carries the dialog trigger. Clicked
    // by visible text to avoid coupling to a specific aria-label that
    // could change with i18n.
    const settingsButton = page.getByRole("button", { name: /服务器设置|Server settings/ });
    await expect(settingsButton).toBeVisible();
    await settingsButton.click();

    // The dialog title reads "服务器配置" / "Server configuration" once
    // the modal mounts. Wait on it before driving inputs to avoid
    // racing the Radix-portaled DialogContent transition.
    await expect(page.getByText(/服务器配置|Server configuration/)).toBeVisible();

    // Cloud is the built-in default. Verify the radio that bears the
    // cloud label is checked, then flip to custom.
    const cloudRadio = page.locator('input[type="radio"][name="kind"]').first();
    const customRadio = page.locator('input[type="radio"][name="kind"]').nth(1);
    await expect(cloudRadio).toBeChecked();
    await expect(customRadio).not.toBeChecked();

    // Custom inputs are hidden until the radio flips — the load-bearing
    // UX guarantee from the previous review round. Verify they only
    // appear after selection.
    await expect(page.locator('input[placeholder*="https://your-server"]')).not.toBeVisible();
    await customRadio.click();
    await expect(customRadio).toBeChecked();
    await expect(page.locator('input[placeholder*="https://your-server"]')).toBeVisible();

    // Save the dev backend URL under a custom label. This same URL is
    // what AGENTSMESH_API_URL pointed at on launch, so post-reload
    // login still works — the difference being it's now resolved via
    // the user-config code path in env.ts instead of the preload
    // bridge default.
    const apiUrl = getApiBaseUrl();
    const customLabel = "Dev backend";
    const labelInput = page.locator('input[placeholder*="例如"], input[placeholder*="e.g."]');
    const urlInput = page.locator('input[placeholder*="https://your-server"]');
    await labelInput.fill(customLabel);
    await urlInput.fill(apiUrl);

    // Connect persists the config and reloads the window. Race the
    // reload against a brief delay so the next assertion runs against
    // the freshly-mounted renderer, not the about-to-unmount one.
    await page.getByRole("button", { name: /^连接$|^Connect$/ }).click();
    await page.waitForLoadState("domcontentloaded");
    await login.expectOnLoginPage();

    // localStorage carries the user-saved config under v2 key.
    const cfg = await page.evaluate(() => {
      const raw = window.localStorage.getItem("agentsmesh.server_config_v2");
      return raw ? JSON.parse(raw) : null;
    });
    expect(cfg).toEqual({ kind: "custom", customLabel, customUrl: apiUrl });

    // The bottom-left button label updates to reflect the active server.
    // `getByText` keeps this resilient to button-structure changes
    // (the label sits in a child span, so role+name regex was racy).
    await expect(page.getByText(customLabel)).toBeVisible();

    // Sanity: the chosen URL drives the next request. Logging in works
    // because apiUrl == AGENTSMESH_API_URL == the dev backend the
    // testkit started — and the only way login would also work if
    // env.ts ignored the user choice would be coincidence (it'd hit
    // app.agentsmesh.ai). Combined with the localStorage assertion
    // above, this proves the custom URL is what the renderer uses.
    await login.login(TEST_USER.email, TEST_USER.password);
    await login.waitForLoginRedirect(TEST_ORG_SLUG);
  });

  test("cancel: dialog closes without persisting changes", async ({ page }) => {
    const login = new LoginPage(page);
    // The previous test logged in, so the shared Electron page may
    // already be on the dashboard route. Force navigation back to
    // /login so the AuthShell + Server Settings entry are reachable.
    await login.goto();
    await login.expectOnLoginPage();

    await page.getByRole("button", { name: /服务器设置|Server settings/ }).click();
    await expect(page.getByText(/服务器配置|Server configuration/)).toBeVisible();

    // Flip to custom, type something, then cancel — the saved config
    // should still report `kind: cloud` (the initial default). Without
    // this guard a "draft survives cancel" regression would persist
    // half-typed URLs to localStorage and surprise the user on the
    // next launch.
    await page.locator('input[type="radio"][name="kind"]').nth(1).click();
    const urlInput = page.locator('input[placeholder*="https://your-server"]');
    await expect(urlInput).toBeVisible();
    await urlInput.fill("https://abandoned.example.com");
    await page.getByRole("button", { name: /^取消$|^Cancel$/ }).click();

    const cfg = await page.evaluate(() => {
      const raw = window.localStorage.getItem("agentsmesh.server_config_v2");
      return raw ? JSON.parse(raw) : null;
    });
    // cfg may be null (never saved) or the default cloud snapshot —
    // both are acceptable; the regression is when it carries the
    // abandoned custom URL.
    expect(cfg?.customUrl ?? "").not.toContain("abandoned.example.com");
  });
});
