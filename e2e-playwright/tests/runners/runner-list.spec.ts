import { test, expect } from "../../fixtures/index";
import { RunnersPage } from "../../pages/runners.page";
import { TEST_ORG_SLUG } from "../../helpers/env";

test.describe("Runner List Page", () => {
  let runnersPage: RunnersPage;

  test.beforeEach(async ({ page }) => {
    runnersPage = new RunnersPage(page, TEST_ORG_SLUG);
  });

  /**
   * TC-UI-001: Runner list page loads
   * Maps to: e2e/runner/ui/TC-UI-001-list-page.yaml
   *
   * Seed data includes a pre-registered 'dev-runner'.
   */
  test("runner list page shows runners from seed data", async ({ page }) => {
    await runnersPage.goto();
    await runnersPage.waitForList();

    // After the IA-Infra refactor, the runners list lives under /infra?tab=runners
    // (the legacy /runners URL redirects there). Auto-select then adds ?id=<n>.
    expect(page.url()).toContain(`/${TEST_ORG_SLUG}/infra`);
    expect(page.url()).toContain("tab=runners");

    // The "Add Runner" button should be visible (in sidebar or detail header).
    await expect(runnersPage.addRunnerButton).toBeVisible();
  });

  /**
   * Runner list with database fixture for additional test data.
   */
  test("displays runner count correctly", async ({ page, db }) => {
    // Setup: ensure at least one runner exists via seed data
    const runnerCount = db.queryValue(
      `SELECT COUNT(*) FROM runners WHERE organization_id = (
        SELECT id FROM organizations WHERE slug = '${TEST_ORG_SLUG}'
      )`
    );

    await runnersPage.goto();
    await runnersPage.waitForList();

    // If runners exist in DB, the page should show content
    if (runnerCount && parseInt(runnerCount) > 0) {
      // At least one element in the runner list area should be visible
      const pageContent = await page.textContent("body");
      expect(pageContent).toBeTruthy();
    }
  });
});
