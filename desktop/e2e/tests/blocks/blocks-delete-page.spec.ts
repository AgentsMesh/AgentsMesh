import { test, expect } from "../../fixtures";
import { BlocksPage } from "../../pages/blocks.page";

// Regression & contract: the sidebar PAGES list must expose a context-menu
// Delete action. Previously no menu at all (blank right-click); after
// wiring ContextMenu, the user should be able to right-click a page → Delete
// → confirm, and the item disappears. Backend dispatches `deleteBlockOp`
// which Rust then broadcasts via WS, so the deletion survives reload.
test("Blocks sidebar · right-click Delete removes a page", async ({ page }) => {
  const blocks = new BlocksPage(page);
  await blocks.goto();
  await blocks.expectOnPage();

  // Let sidebar hydrate from Rust SSOT.
  await expect(page.getByText(/loading workspace/i)).toHaveCount(0, { timeout: 10_000 });

  // Create a fresh page via the "+" button so the test is self-contained.
  const beforeCount = await page.locator('[data-testid^="blocks-sidebar-page-"]').count();
  const addBtn = page.getByRole("button", { name: /add page|新建页面|create page/i }).first();
  await addBtn.click();
  await expect.poll(
    () => page.locator('[data-testid^="blocks-sidebar-page-"]').count(),
    { timeout: 10_000 },
  ).toBe(beforeCount + 1);

  // Right-click the newest leaf to open the context menu.
  const items = page.locator('[data-testid^="blocks-sidebar-page-"]');
  const target = items.nth(await items.count() - 1);
  const testId = await target.getAttribute("data-testid");
  expect(testId).toBeTruthy();
  await target.click({ button: "right" });

  const deleteItem = page.locator(`[data-testid="${testId}-delete"]`);
  await expect(deleteItem).toBeVisible({ timeout: 3000 });
  await deleteItem.click();

  // Confirm in the AlertDialog.
  const confirm = page.locator('[data-testid="blocks-sidebar-delete-confirm"]');
  await expect(confirm).toBeVisible();
  await confirm.click();

  // Target entry disappears from the sidebar.
  await expect(page.locator(`[data-testid="${testId}"]`)).toHaveCount(0, { timeout: 10_000 });
});
