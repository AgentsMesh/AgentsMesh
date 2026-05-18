import { test, expect } from "../../fixtures";
import { InfraPage } from "../../pages/infra.page";

// Empty-state CTA → modal coverage. Forces an empty Runner/Repository list by
// overriding the IPC handlers in the main process, so the empty-state branch
// always renders regardless of dev seed data. Guards the regression PR #379
// fixed on web and the parallel desktop fix: empty-state buttons used to push
// URLs no one consumed, so clicking them did nothing.
//
// Note: desktop runs inside the IDE shell which also has sidebar buttons named
// "Add Runner" / "Import Repository". We scope the CTA locator to the
// EmptyState container (anchored on the "No runners yet" / "No repositories yet"
// heading) so we click the empty-state primary action, not the sidebar one.
test.describe("Infra empty state — CTA opens modal", () => {
  test.beforeEach(async ({ electronApp, page }) => {
    await electronApp.evaluate(({ ipcMain }) => {
      ipcMain.removeHandler("runnerFetchRunners");
      ipcMain.handle("runnerFetchRunners", async () =>
        JSON.stringify({ runners: [] }),
      );
      ipcMain.removeHandler("repositoryList");
      ipcMain.handle("repositoryList", async () =>
        JSON.stringify({ repositories: [] }),
      );
    });
    // Force the renderer to remount so React hooks' initial fetch picks up
    // the new IPC handlers — without reload, the page launched by the fixture
    // may have already fired fetches against the original handlers.
    await page.reload();
  });

  test("Add Runner empty-state button opens the Add Runner modal", async ({ page }) => {
    const infra = new InfraPage(page);
    await infra.gotoTab("runners");
    await page
      .getByRole("heading", { name: /no runners yet/i })
      .locator("..")
      .getByRole("button", { name: /^add runner$/i })
      .click();
    await expect(
      page.getByRole("heading", { name: /^add runner$/i }),
    ).toBeVisible();
  });

  test("Add Runner modal closes via cancel", async ({ page }) => {
    const infra = new InfraPage(page);
    await infra.gotoTab("runners");
    await page
      .getByRole("heading", { name: /no runners yet/i })
      .locator("..")
      .getByRole("button", { name: /^add runner$/i })
      .click();
    const heading = page.getByRole("heading", { name: /^add runner$/i });
    await heading.waitFor({ state: "visible" });
    await page.getByRole("button", { name: /^cancel$/i }).click();
    await expect(heading).toBeHidden();
  });

  test("Import Repository empty-state button opens the Import modal", async ({ page }) => {
    const infra = new InfraPage(page);
    await infra.gotoTab("repositories");
    await page
      .getByRole("heading", { name: /no repositories yet/i })
      .locator("..")
      .getByRole("button", { name: /^import repository$/i })
      .click();
    await expect(
      page.getByRole("heading", { name: /^import repository$/i }),
    ).toBeVisible();
  });

  test("Import Repository modal closes via cancel", async ({ page }) => {
    const infra = new InfraPage(page);
    await infra.gotoTab("repositories");
    await page
      .getByRole("heading", { name: /no repositories yet/i })
      .locator("..")
      .getByRole("button", { name: /^import repository$/i })
      .click();
    const heading = page.getByRole("heading", { name: /^import repository$/i });
    await heading.waitFor({ state: "visible" });
    await page.getByRole("button", { name: /^cancel$/i }).click();
    await expect(heading).toBeHidden();
  });
});
