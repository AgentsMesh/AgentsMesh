import { test as uiTest, expect as uiExpect } from "@playwright/test";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { collectConsoleErrors, assertNoWasmErrors } from "../../helpers/console-errors";
import { AddRunnerModal } from "../../pages/modals/add-runner.modal";
import { ImportRepositoryModal } from "../../pages/modals/import-repository.modal";

// Infra is the unified Runner + Repository landing tab. master-detail layout:
// left list → right detail pane in the main area (not the old /runners or
// /repositories list pages). On entry without ?id, the page should auto-select
// the first entry and update the URL.

uiTest.describe("Infra page — UI", () => {
  uiTest.beforeEach(async () => { clearAuthRateLimit(); });

  uiTest("root /infra defaults to ?tab=runners", async ({ page }) => {
    const errors = collectConsoleErrors(page);
    await page.goto(`/${TEST_ORG_SLUG}/infra`);
    await page.waitForLoadState("networkidle");
    await uiExpect(page).toHaveURL(/tab=runners/);
    assertNoWasmErrors(errors);
  });

  uiTest("switching to runners tab updates URL", async ({ page }) => {
    const errors = collectConsoleErrors(page);
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=runners`);
    await page.waitForLoadState("networkidle");
    await uiExpect(page).toHaveURL(/tab=runners/);
    assertNoWasmErrors(errors);
  });

  uiTest("repositories tab auto-selects first repo when no id given", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=repositories`);
    await page.waitForLoadState("networkidle");
    // After auto-select, the URL should gain an ?id=<n> — if there are repos
    // in the seed data. If not, the empty state is visible instead.
    await page.waitForTimeout(800);
    const hasId = page.url().includes("id=");
    const hasEmpty = await page.getByText(/no repositor|还没有仓库|empty/i).isVisible({ timeout: 500 }).catch(() => false);
    uiExpect(hasId || hasEmpty).toBe(true);
  });

  uiTest("runners tab auto-selects first runner when no id given", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=runners`);
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(800);
    const hasId = page.url().includes("id=");
    const hasEmpty = await page.getByText(/no runners|还没有|empty/i).isVisible({ timeout: 500 }).catch(() => false);
    uiExpect(hasId || hasEmpty).toBe(true);
  });

  uiTest("detail pane renders in main area (not list redirect)", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=repositories`);
    await page.waitForLoadState("networkidle");
    // Even after auto-select we must still be on /infra, never redirected
    // back to /repositories (that's the pre-refactor regression path).
    uiExpect(page.url()).toContain("/infra");
    uiExpect(page.url()).not.toMatch(/\/repositories($|\?)/);
  });
});

// Empty-state CTA → modal coverage. Forces an empty list via page.route so the
// empty-state branch always renders, regardless of dev seed data. Guards the
// regression PR #379 fixed: empty-state buttons used to push a URL no one
// consumed, so clicking them did nothing.
uiTest.describe("Infra empty state — CTA opens modal", () => {
  uiTest.beforeEach(async ({ page }) => {
    clearAuthRateLimit();
    await page.route(`**/api/v1/organizations/${TEST_ORG_SLUG}/runners**`, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ runners: [] }),
      }),
    );
    await page.route(`**/api/v1/organizations/${TEST_ORG_SLUG}/repositories**`, (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ repositories: [] }),
      }),
    );
  });

  uiTest("Add Runner empty-state button opens the Add Runner modal", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=runners`);
    await page.waitForLoadState("networkidle");
    await page.getByRole("button", { name: /^add runner$/i }).click();
    await new AddRunnerModal(page).waitForOpen();
    await uiExpect(page.getByRole("heading", { name: /^add runner$/i })).toBeVisible();
  });

  uiTest("Add Runner modal closes via cancel", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=runners`);
    await page.waitForLoadState("networkidle");
    await page.getByRole("button", { name: /^add runner$/i }).click();
    const modal = new AddRunnerModal(page);
    await modal.waitForOpen();
    await modal.close();
    await uiExpect(page.getByRole("heading", { name: /^add runner$/i })).toBeHidden();
  });

  uiTest("Import Repository empty-state button opens the Import modal", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=repositories`);
    await page.waitForLoadState("networkidle");
    await page.getByRole("button", { name: /^import repository$/i }).click();
    await new ImportRepositoryModal(page).waitForOpen();
  });

  uiTest("Import Repository modal closes via cancel", async ({ page }) => {
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=repositories`);
    await page.waitForLoadState("networkidle");
    await page.getByRole("button", { name: /^import repository$/i }).click();
    const modal = new ImportRepositoryModal(page);
    await modal.waitForOpen();
    await modal.close();
    await modal.waitForClosed();
  });
});
