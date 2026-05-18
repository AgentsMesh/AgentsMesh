import type { Page } from "@playwright/test";

/**
 * Page Object Model for the Add Runner Modal.
 * Based on: clients/web/src/components/ide/modals/AddRunnerModal.tsx
 */
export class AddRunnerModal {
  constructor(private page: Page) {}

  async waitForOpen(): Promise<void> {
    await this.page
      .getByRole("heading", { name: /^add runner$/i })
      .waitFor({ state: "visible" });
  }

  async waitForClosed(): Promise<void> {
    await this.page
      .getByRole("heading", { name: /^add runner$/i })
      .waitFor({ state: "hidden" });
  }

  async getToken(): Promise<string | null> {
    const code = this.page.locator("code, pre").first();
    if (await code.isVisible()) return code.textContent();
    return null;
  }

  async generateToken(): Promise<void> {
    await this.page.getByRole("button", { name: /generate/i }).click();
  }

  async close(): Promise<void> {
    await this.page.getByRole("button", { name: /^(done|cancel|close)$/i }).first().click();
  }
}
