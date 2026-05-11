import { test, expect } from "../../fixtures";
import { LoginPage } from "../../pages/login.page";
import { TEST_USER, TEST_ORG_SLUG, getApiBaseUrl } from "../../helpers/env";
import { resolve } from "node:path";
import { rmSync } from "node:fs";

// Fresh Electron profile every run. After the SSOT refactor (2026-05-10),
// server config persists at `userData/server.json` (NOT localStorage), so
// each test starts from a clean disk to get deterministic defaults.
const FRESH_USER_DATA = resolve(__dirname, "../../.auth/electron-userdata-server-settings");

test.use({
  skipAuthRestore: true,
  userDataDir: FRESH_USER_DATA,
});

test.beforeAll(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

// Per-test reset of the Electron profile. Each spec mutates persisted
// config + occasionally completes a real login (driving session blobs into
// FileStorage). Without per-test cleanup, subsequent specs land on a
// dashboard inherited from the prior session.
test.beforeEach(() => {
  rmSync(FRESH_USER_DATA, { recursive: true, force: true });
});

const GLOBAL_URL = "https://agentsmesh.ai";
const CN_URL = "https://agentsmesh.cn";

// Radio order in the dialog: 0=global, 1=cn, 2=custom.
const RADIO_GLOBAL = 0;
const RADIO_CN = 1;
const RADIO_CUSTOM = 2;

async function readSnapshot(page: import("@playwright/test").Page) {
  return page.evaluate(() => window.electronAPI.serverConfig.snapshot);
}

async function openServerSettings(page: import("@playwright/test").Page) {
  await page.getByRole("button", { name: /服务器设置|Server settings/ }).click();
  await expect(page.getByText(/服务器配置|Server configuration/)).toBeVisible();
}

test.describe("Auth · server settings", () => {
  // Default preset = Global. PR #336 retired the broken `app.agentsmesh.ai`
  // host; this test pins that the dialog never resurrects it after later
  // refactors.
  test("default preset is Global with the correct host", async ({ page }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await openServerSettings(page);

    const radios = page.locator('input[type="radio"][name="kind"]');
    await expect(radios.nth(RADIO_GLOBAL)).toBeChecked();
    await expect(radios.nth(RADIO_CN)).not.toBeChecked();
    await expect(radios.nth(RADIO_CUSTOM)).not.toBeChecked();

    await expect(page.getByText(GLOBAL_URL)).toBeVisible();
    await expect(page.getByText(CN_URL)).toBeVisible();
    await expect(page.locator("text=app.agentsmesh.ai")).toHaveCount(0);
  });

  // Switching to CN persists `kind: "cn"` to userData/server.json and
  // reloads the renderer. Main rebuilds AppState bound to the new URL —
  // so renderer's preload snapshot reflects it on next read.
  test("switching to CN preset persists kind=cn", async ({ page }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await openServerSettings(page);

    await page.locator('input[type="radio"][name="kind"]').nth(RADIO_CN).click();
    await page.getByRole("button", { name: /^连接$|^Connect$/ }).click();
    await page.waitForLoadState("domcontentloaded");
    await login.expectOnLoginPage();

    expect(await readSnapshot(page)).toEqual({
      kind: "cn",
      customLabel: "",
      customUrl: "",
    });

    // The preload snapshot is the SSOT for `electronAPI.apiUrl` — main
    // rebuilt AppState with this URL too.
    const apiUrl = await page.evaluate(() => window.electronAPI.apiUrl);
    expect(apiUrl).toBe(CN_URL);

    // Footer pill reflects the active preset.
    await expect(page.getByText(/AgentsMesh.*中国|AgentsMesh.*China/)).toBeVisible();
  });

  test("custom server: pick → save → renderer hits chosen URL on next login", async ({ page }) => {
    const login = new LoginPage(page);
    await login.expectOnLoginPage();
    await openServerSettings(page);

    const radios = page.locator('input[type="radio"][name="kind"]');
    await expect(radios.nth(RADIO_GLOBAL)).toBeChecked();

    // Custom inputs are hidden until the radio flips.
    await expect(page.locator('input[placeholder*="https://your-server"]')).not.toBeVisible();
    await radios.nth(RADIO_CUSTOM).click();
    await expect(radios.nth(RADIO_CUSTOM)).toBeChecked();
    await expect(page.locator('input[placeholder*="https://your-server"]')).toBeVisible();

    // Save the dev backend URL — same one AGENTSMESH_API_URL pointed at on
    // launch, so post-reload login still works against the dev backend.
    const apiUrl = getApiBaseUrl();
    const customLabel = "Dev backend";
    const labelInput = page.locator('input[placeholder*="例如"], input[placeholder*="e.g."]');
    const urlInput = page.locator('input[placeholder*="https://your-server"]');
    await labelInput.fill(customLabel);
    await urlInput.fill(apiUrl);

    await page.getByRole("button", { name: /^连接$|^Connect$/ }).click();
    await page.waitForLoadState("domcontentloaded");
    await login.expectOnLoginPage();

    expect(await readSnapshot(page)).toEqual({ kind: "custom", customLabel, customUrl: apiUrl });

    await expect(page.getByText(customLabel)).toBeVisible();

    // Sanity: chosen URL drives the next request; login works because
    // apiUrl == AGENTSMESH_API_URL == the dev backend.
    await login.login(TEST_USER.email, TEST_USER.password);
    await login.waitForLoginRedirect(TEST_ORG_SLUG);
  });

  test("cancel: dialog closes without persisting changes", async ({ page }) => {
    const login = new LoginPage(page);
    await login.goto();
    await login.expectOnLoginPage();
    await openServerSettings(page);

    await page.locator('input[type="radio"][name="kind"]').nth(RADIO_CUSTOM).click();
    const urlInput = page.locator('input[placeholder*="https://your-server"]');
    await expect(urlInput).toBeVisible();
    await urlInput.fill("https://abandoned.example.com");
    await page.getByRole("button", { name: /^取消$|^Cancel$/ }).click();

    // No persistence — snapshot still default Global. Without this guard
    // a "draft survives cancel" regression would write the abandoned URL
    // through the IPC layer to disk.
    const cfg = await readSnapshot(page);
    expect(cfg.kind).toBe("global");
    expect(cfg.customUrl).toBe("");
  });
});
