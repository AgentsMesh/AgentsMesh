import { test, expect } from "../../fixtures";
import { gotoHash } from "../../helpers/nav";

/**
 * Desktop counterpart of clients/web/e2e-playwright/tests/settings/git-providers-active.spec.ts.
 *
 * The renderer reuses the web GitSettingsContent + GitProviderCard components,
 * and Desktop ships the same wasm-core artifact. So the regression must be
 * verified on the Electron build too: a freshly created provider should not
 * render the "已禁用 / Disabled" badge, and toggling is_active in the
 * EditProviderDialog must persist to the backend.
 */
test.describe("Desktop · Settings → Git Providers · is_active flow", () => {
  let providerId: number | undefined;

  test.afterEach(async ({ api }) => {
    if (providerId) {
      await api.delete(`/api/v1/users/repository-providers/${providerId}`);
      providerId = undefined;
    }
  });

  test("created provider renders without disabled badge", async ({ api, page }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: `Desktop E2E Active ${Date.now()}`,
      base_url: "https://api.github.com",
      bot_token: "ghp_desktop_active",
    });
    const created = (await createRes.json()).provider;
    providerId = created.id;
    expect(created.is_active).toBe(true);

    await gotoHash(page, "/settings/git");
    const card = page.locator(`[data-testid="git-provider-card"][data-provider-id="${providerId}"]`);
    await expect(card).toBeVisible({ timeout: 10_000 });
    await expect(card).toHaveAttribute("data-is-active", "true");
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toHaveCount(0);
  });

  test("toggling is_active off via dialog persists across reload", async ({ api, page }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: `Desktop E2E Toggle ${Date.now()}`,
      base_url: "https://api.github.com",
      bot_token: "ghp_desktop_toggle",
    });
    providerId = (await createRes.json()).provider.id;

    await gotoHash(page, "/settings/git");
    const card = page.locator(`[data-testid="git-provider-card"][data-provider-id="${providerId}"]`);
    await expect(card).toBeVisible({ timeout: 10_000 });

    await card.locator('[data-testid="git-provider-edit-button"]').click();
    const toggle = page.locator('[data-testid="edit-provider-active-toggle"]');
    const toggleLabel = page.locator('[data-testid="edit-provider-active-toggle-label"]');
    await expect(toggle).toBeChecked();
    await toggleLabel.click();
    await expect(toggle).not.toBeChecked();
    await page.locator('[data-testid="edit-provider-save-button"]').click();
    await expect(page.locator('[data-testid="edit-provider-dialog"]')).toBeHidden();

    await expect(card).toHaveAttribute("data-is-active", "false");
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toBeVisible();

    const verifyRes = await api.get(`/api/v1/users/repository-providers/${providerId}`);
    expect((await verifyRes.json()).provider.is_active).toBe(false);

    await gotoHash(page, "/workspace");
    await gotoHash(page, "/settings/git");
    await expect(card.locator('[data-testid="git-provider-disabled-badge"]')).toBeVisible();
  });
});
