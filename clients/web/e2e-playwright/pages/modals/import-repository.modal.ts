import type { Page } from "@playwright/test";

/**
 * Page Object Model for the Import Repository Modal.
 * Based on: clients/web/src/components/ide/modals/ImportRepositoryModal/
 */
export class ImportRepositoryModal {
  constructor(private page: Page) {}

  async waitForOpen(): Promise<void> {
    await this.page
      .getByRole("heading", { name: /^import repository$/i })
      .waitFor({ state: "visible" });
  }

  async waitForClosed(): Promise<void> {
    await this.page
      .getByRole("heading", { name: /^import repository$/i })
      .waitFor({ state: "hidden" });
  }

  async close(): Promise<void> {
    await this.page.getByRole("button", { name: /^cancel$/i }).click();
  }
}
