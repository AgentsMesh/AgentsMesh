import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG, getApiBaseUrl } from "../../helpers/env";
import { resolve } from "node:path";
import { rmSync } from "node:fs";

// Fresh Electron profile every run — the spec writes to localStorage
// (the server config persists there), and we want a known starting
// point on every invocation. Sharing userData with other auth specs
// would race the persisted choice across runs.
const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-server-settings");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

const STORAGE_KEY = "agentsmesh.server_config_v2";
const GLOBAL_URL = "https://agentsmesh.ai";
const CN_URL = "https://agentsmesh.cn";

// Radio order in the dialog: 0=global, 1=cn, 2=custom. Encoded as
// constants so a re-order in the modal flips one place, not seven.
const RADIO_GLOBAL = 0;
const RADIO_CN = 1;
const RADIO_CUSTOM = 2;

async function readConfig(page: import("@playwright/test").Page) {
  return page.evaluate((key) => {
    const raw = window.localStorage.getItem(key);
    return raw ? JSON.parse(raw) : null;
  }, STORAGE_KEY);
}

async function seedConfig(page: import("@playwright/test").Page, value: unknown) {
  await page.evaluate(
    ([key, raw]) => window.localStorage.setItem(key as string, raw as string),
    [STORAGE_KEY, JSON.stringify(value)],
  );
}

async function openServerSettings(page: import("@playwright/test").Page) {
  await page.getByRole("button", { name: /服务器设置|Server settings/ }).click();
  await expect(page.getByText(/服务器配置|Server configuration/)).toBeVisible();
}

test.describe("Auth · server settings", () => {
  // The default preset must be **Global** (https://agentsmesh.ai), not
  // the long-broken Cloud (https://app.agentsmesh.ai) which never
  // resolved in production. This test guards that PR #336 stays the
  // SSOT — if anyone re-introduces the bad host the dialog will show
  // it inline as the selected preset's URL.
  test("default preset is Global with the correct host", async ({ page }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await openServerSettings(page);

    const radios = page.locator('input[type="radio"][name="kind"]');
    await expect(radios.nth(RADIO_GLOBAL)).toBeChecked();
    await expect(radios.nth(RADIO_CN)).not.toBeChecked();
    await expect(radios.nth(RADIO_CUSTOM)).not.toBeChecked();

    // The Global preset row must surface its actual URL — anyone
    // reintroducing app.agentsmesh.ai will trip this assertion.
    await expect(page.getByText(GLOBAL_URL)).toBeVisible();
    await expect(page.getByText(CN_URL)).toBeVisible();
    await expect(page.locator("text=app.agentsmesh.ai")).toHaveCount(0);
  });

  // Switching to CN must persist `kind: "cn"` and reload the window —
  // env.ts then resolves the active URL through the cn preset path.
  // We don't try to log in (cn backend isn't reachable from CI), the
  // localStorage assertion + footer label change is sufficient.
  test("switching to CN preset persists kind=cn", async ({ page }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await openServerSettings(page);

    await page.locator('input[type="radio"][name="kind"]').nth(RADIO_CN).click();
    await page.getByRole("button", { name: /^连接$|^Connect$/ }).click();
    await page.waitForLoadState("domcontentloaded");
    await login.expectOnLoginPage();

    expect(await readConfig(page)).toEqual({
      kind: "cn",
      customLabel: "",
      customUrl: "",
    });

    // Footer pill reflects the active preset.
    await expect(page.getByText(/AgentsMesh.*中国|AgentsMesh.*China/)).toBeVisible();
  });

  // v0.30.x shipped `kind: "cloud"` pointing at app.agentsmesh.ai.
  // After this PR, the dialog must read it as `global` so the user
  // doesn't end up on a dead host. The localStorage record is left
  // in v2 form (we don't eagerly rewrite); the migration is at the
  // read site (normaliseKind in server-config.ts).
  test("legacy kind=cloud migrates to Global on dialog open", async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.expectOnLoginPage();

    // Seed the legacy shape that v0.30.x desktops persisted.
    await seedConfig(page, { kind: "cloud", customLabel: "", customUrl: "" });
    await page.reload();
    await login.expectOnLoginPage();

    await openServerSettings(page);

    // Global radio is checked even though the saved value says "cloud".
    const radios = page.locator('input[type="radio"][name="kind"]');
    await expect(radios.nth(RADIO_GLOBAL)).toBeChecked();
  });

  test("custom server: pick → save → reload → login goes to chosen URL", async ({ page }) => {
    const login = new LoginPage(page);
    // Reset to a clean slate — earlier tests may have persisted a
    // preset choice on the shared profile.
    await login.goto();
    await page.evaluate((key) => window.localStorage.removeItem(key), STORAGE_KEY);
    await page.reload();
    await login.expectOnLoginPage();

    await openServerSettings(page);

    const radios = page.locator('input[type="radio"][name="kind"]');
    await expect(radios.nth(RADIO_GLOBAL)).toBeChecked();

    // Custom inputs are hidden until the radio flips — the load-bearing
    // UX guarantee. Verify they only appear after selection.
    await expect(page.locator('input[placeholder*="https://your-server"]')).not.toBeVisible();
    await radios.nth(RADIO_CUSTOM).click();
    await expect(radios.nth(RADIO_CUSTOM)).toBeChecked();
    await expect(page.locator('input[placeholder*="https://your-server"]')).toBeVisible();

    // Save the dev backend URL under a custom label. This same URL is
    // what AGENTSMESH_API_URL pointed at on launch, so post-reload
    // login still works — proving the user choice is what env.ts
    // resolves to instead of the preload bridge default.
    const apiUrl = getApiBaseUrl();
    const customLabel = "Dev backend";
    const labelInput = page.locator('input[placeholder*="例如"], input[placeholder*="e.g."]');
    const urlInput = page.locator('input[placeholder*="https://your-server"]');
    await labelInput.fill(customLabel);
    await urlInput.fill(apiUrl);

    await page.getByRole("button", { name: /^连接$|^Connect$/ }).click();
    await page.waitForLoadState("domcontentloaded");
    await login.expectOnLoginPage();

    expect(await readConfig(page)).toEqual({ kind: "custom", customLabel, customUrl: apiUrl });

    await expect(page.getByText(customLabel)).toBeVisible();

    // Sanity: the chosen URL drives the next request. Logging in works
    // because apiUrl == AGENTSMESH_API_URL == the dev backend.
    await login.login(TEST_USER.email, TEST_USER.password);
    await login.waitForLoginRedirect(TEST_ORG_SLUG);
  });

  test("cancel: dialog closes without persisting changes", async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.expectOnLoginPage();
    await openServerSettings(page);

    // Flip to custom, type something, then cancel — the saved config
    // should still report the prior value (or null). Without this guard
    // a "draft survives cancel" regression would persist half-typed
    // URLs and surprise the user on next launch.
    await page.locator('input[type="radio"][name="kind"]').nth(RADIO_CUSTOM).click();
    const urlInput = page.locator('input[placeholder*="https://your-server"]');
    await expect(urlInput).toBeVisible();
    await urlInput.fill("https://abandoned.example.com");
    await page.getByRole("button", { name: /^取消$|^Cancel$/ }).click();

    const cfg = await readConfig(page);
    // cfg may be null (never saved), the seeded custom value from a
    // prior test, or a preset — the regression is when it carries the
    // abandoned URL.
    expect(cfg?.customUrl ?? "").not.toContain("abandoned.example.com");
  });
});
