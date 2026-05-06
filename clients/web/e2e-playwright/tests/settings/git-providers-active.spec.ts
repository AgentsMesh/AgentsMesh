import { test, expect } from "../../fixtures/index";
import { TEST_ORG_SLUG } from "../../helpers/env";

/**
 * Regression coverage for the wasm-core field-stripping bug
 * (Rust types missing is_active / has_identity / has_bot_token / has_client_id):
 * every provider used to render the "已禁用 / Disabled" badge regardless
 * of DB state, and the EditProviderDialog toggle silently no-op'd.
 */
test.describe("Settings → Git Providers · is_active flow", () => {
  let providerId: number | undefined;

  test.afterEach(async ({ api }) => {
    if (providerId) {
      await api.delete(`/api/v1/users/repository-providers/${providerId}`);
      providerId = undefined;
    }
  });

  test("freshly created provider does NOT show disabled badge", async ({ api, page }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: `E2E UI Active ${Date.now()}`,
      base_url: "https://api.github.com",
      bot_token: "ghp_e2e_ui_active",
    });
    const created = (await createRes.json()).provider;
    providerId = created.id;
    expect(created.is_active).toBe(true);

    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=personal&tab=git`);
    await page.waitForLoadState("networkidle");

    const card = page.locator(`[data-testid="git-provider-card"][data-provider-id="${providerId}"]`);
    await expect(card).toBeVisible({ timeout: 10_000 });
    await expect(card).toHaveAttribute("data-is-active", "true");

    const disabledBadge = card.locator('[data-testid="git-provider-disabled-badge"]');
    await expect(disabledBadge).toHaveCount(0);
  });

  test("toggling is_active off in EditDialog persists and shows disabled badge", async ({ api, page }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: `E2E UI Toggle ${Date.now()}`,
      base_url: "https://api.github.com",
      bot_token: "ghp_e2e_ui_toggle",
    });
    providerId = (await createRes.json()).provider.id;

    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=personal&tab=git`);
    await page.waitForLoadState("networkidle");

    const card = page.locator(`[data-testid="git-provider-card"][data-provider-id="${providerId}"]`);
    await expect(card).toBeVisible();
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toHaveCount(0);

    await card.locator('[data-testid="git-provider-edit-button"]').click();
    const dialog = page.locator('[data-testid="edit-provider-dialog"]');
    await expect(dialog).toBeVisible();

    const toggle = dialog.locator('[data-testid="edit-provider-active-toggle"]');
    const toggleLabel = dialog.locator('[data-testid="edit-provider-active-toggle-label"]');
    await expect(toggle).toBeChecked();
    await toggleLabel.click();
    await expect(toggle).not.toBeChecked();

    await dialog.locator('[data-testid="edit-provider-save-button"]').click();
    await expect(dialog).toBeHidden();

    await expect(card).toHaveAttribute("data-is-active", "false");
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toBeVisible();

    const verifyRes = await api.get(`/api/v1/users/repository-providers/${providerId}`);
    expect((await verifyRes.json()).provider.is_active).toBe(false);
  });

  test("re-enabling a disabled provider clears the disabled badge", async ({ api, page }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: `E2E UI ReEnable ${Date.now()}`,
      base_url: "https://api.github.com",
      bot_token: "ghp_e2e_reenable",
    });
    providerId = (await createRes.json()).provider.id;
    await api.put(`/api/v1/users/repository-providers/${providerId}`, { is_active: false });

    await page.goto(`/${TEST_ORG_SLUG}/settings?scope=personal&tab=git`);
    await page.waitForLoadState("networkidle");

    const card = page.locator(`[data-testid="git-provider-card"][data-provider-id="${providerId}"]`);
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toBeVisible();

    await card.locator('[data-testid="git-provider-edit-button"]').click();
    const toggle = page.locator('[data-testid="edit-provider-active-toggle"]');
    const toggleLabel = page.locator('[data-testid="edit-provider-active-toggle-label"]');
    await expect(toggle).not.toBeChecked();
    await toggleLabel.click();
    await expect(toggle).toBeChecked();
    await page.locator('[data-testid="edit-provider-save-button"]').click();
    await expect(page.locator('[data-testid="edit-provider-dialog"]')).toBeHidden();

    await expect(card).toHaveAttribute("data-is-active", "true");
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toHaveCount(0);

    const verifyRes = await api.get(`/api/v1/users/repository-providers/${providerId}`);
    expect((await verifyRes.json()).provider.is_active).toBe(true);
  });
});
