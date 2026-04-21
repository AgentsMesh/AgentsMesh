import type { Page } from "@playwright/test";
import { TEST_ORG_SLUG } from "../helpers/env";
import { gotoHash, expectHashMatches } from "../helpers/nav";

// Route: #/:org/blocks (block store document tree).
export class BlocksPage {
  constructor(private page: Page) {}

  async goto(): Promise<void> {
    await gotoHash(this.page, `/${TEST_ORG_SLUG}/blocks`);
  }

  async gotoWithWorkspace(wsId: string): Promise<void> {
    await gotoHash(this.page, `/${TEST_ORG_SLUG}/blocks?ws=${encodeURIComponent(wsId)}`);
  }

  async expectOnPage(): Promise<void> {
    await expectHashMatches(this.page, /\/blocks/);
  }
}
