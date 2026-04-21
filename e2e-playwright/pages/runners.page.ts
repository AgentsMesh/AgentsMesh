import type { Locator, Page } from "@playwright/test";

/**
 * Page Object Model for the Runner list + detail.
 *
 * After the IA-Infra refactor, Runner management lives under /infra?tab=runners
 * in a master-detail layout (list sidebar + detail pane in `main`). The legacy
 * /runners URL still redirects to the Infra tab. The "Add Runner" button moved
 * into the runner list sidebar (or empty-state CTA).
 */
export class RunnersPage {
  readonly addRunnerButton: Locator;
  readonly runnerTable: Locator;

  constructor(
    private page: Page,
    private orgSlug: string
  ) {
    this.addRunnerButton = page.getByRole("button", {
      name: /add runner|添加 runner|添加.*Runner/i,
    }).first();
    this.runnerTable = page.locator("table");
  }

  async goto(): Promise<void> {
    await this.page.goto(`/${this.orgSlug}/infra?tab=runners`);
    await this.page.waitForLoadState("networkidle");
  }

  async waitForList(): Promise<void> {
    // After navigation, either a detail pane renders (selectedId auto-set) or
    // the empty state CTA shows. Wait for any indicator that the list query
    // resolved.
    await Promise.race([
      this.page.waitForURL(/tab=runners/, { timeout: 15_000 }).catch(() => null),
      this.page.waitForTimeout(3000),
    ]);
  }

  getRunnerByNodeId(nodeId: string): Locator {
    return this.page.locator("table tbody tr").filter({ hasText: nodeId });
  }

  async getRunnerCount(): Promise<number> {
    return this.page.locator("table tbody tr").count();
  }
}

