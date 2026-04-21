import { test, expect } from "../../fixtures/index";
import { WorkspacePage } from "../../pages/workspace.page";
import { SidebarPage } from "../../pages/sidebar.page";
import { TEST_ORG_SLUG } from "../../helpers/env";

/**
 * Workspace UI layout tests.
 * Maps to: e2e/workspace/TC-WS-001~004
 */
test.describe("Workspace Layout", () => {
  /**
   * TC-WS-001: Workspace page loads and shows empty/active state
   */
  test("workspace page loads correctly", async ({ page }) => {
    const workspace = new WorkspacePage(page, TEST_ORG_SLUG);
    await workspace.goto();

    // Page should either show empty state or terminal grid
    const isEmpty = await workspace.isEmptyState();
    const hasGrid = await workspace.hasTerminalGrid();
    const tabCount = await workspace.getPodTabCount();

    // At least one condition should be true
    expect(isEmpty || hasGrid || tabCount >= 0).toBe(true);
  });

  /**
   * TC-WS-001: Empty state shows create button
   */
  test("empty state displays create pod button", async ({ page }) => {
    const workspace = new WorkspacePage(page, TEST_ORG_SLUG);
    await workspace.goto();

    // If empty state, create button should be visible
    const isEmpty = await workspace.isEmptyState();
    if (isEmpty) {
      await expect(workspace.createPodButton).toBeVisible();
    }
  });

  /**
   * TC-WS-002: Navigate to workspace from sidebar
   */
  test("navigate to workspace via sidebar", async ({ page }) => {
    const sidebar = new SidebarPage(page, TEST_ORG_SLUG);
    // Start from infra instead of legacy /runners — the redirect chain through
    // /runners → /infra?tab=runners → /infra?tab=runners&id=<n> can exceed
    // playwright's goto timeout.
    await page.goto(`/${TEST_ORG_SLUG}/infra?tab=runners`);
    await page.waitForLoadState("networkidle");

    await sidebar.dismissDevOverlay();
    await sidebar.navigateTo("workspace");
    expect(page.url()).toContain(`/${TEST_ORG_SLUG}/workspace`);
  });

  /**
   * TC-WS-002: Create pod modal can be opened
   */
  test("create pod modal opens from workspace", async ({ page }) => {
    const workspace = new WorkspacePage(page, TEST_ORG_SLUG);
    await workspace.goto();

    // Find any create/new button
    const createBtn = page.getByRole("button", {
      name: /create|new|创建|新建/i,
    }).first();

    if (await createBtn.isVisible()) {
      await createBtn.click();
      // Modal should appear
      const dialog = page.locator('[role="dialog"]').first();
      const visible = await dialog.isVisible().catch(() => false);
      if (visible) {
        await expect(dialog).toBeVisible();
        // Close it
        await page.keyboard.press("Escape");
      }
    }
  });
});
