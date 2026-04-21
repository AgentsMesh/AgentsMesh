import { test as uiTest, expect as uiExpect } from "@playwright/test";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";
import { collectConsoleErrors, assertNoWasmErrors } from "../../helpers/console-errors";

// Infra is the unified Runner + Repository landing tab. master-detail layout:
// left list → right detail pane in the main area (not the old /runners or
// /repositories list pages). On entry without ?id, the page should auto-select
// the first entry and update the URL.

uiTest.describe("Infra page — UI", () => {
  uiTest.beforeEach(async () => { clearAuthRateLimit(); });

  uiTest("root /infra defaults to ?tab=repositories", async ({ page }) => {
    const errors = collectConsoleErrors(page);
    await page.goto(`/${TEST_ORG_SLUG}/infra`);
    await page.waitForLoadState("networkidle");
    await uiExpect(page).toHaveURL(/tab=repositories/);
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
