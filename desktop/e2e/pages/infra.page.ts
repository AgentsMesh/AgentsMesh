import type { Page } from "@playwright/test";
import { TEST_ORG_SLUG } from "../helpers/env";
import { gotoHash, expectHashMatches } from "../helpers/nav";

// Route: #/:org/infra?tab=repositories|runners&id=<n>.
// Master-detail: selecting a row updates the query string; the detail pane renders in the main area.
export class InfraPage {
  constructor(private page: Page) {}

  async goto(): Promise<void> {
    await gotoHash(this.page, `/${TEST_ORG_SLUG}/infra`);
  }

  async gotoTab(tab: "repositories" | "runners"): Promise<void> {
    await gotoHash(this.page, `/${TEST_ORG_SLUG}/infra?tab=${tab}`);
  }

  async gotoWithSelection(tab: "repositories" | "runners", id: number | string): Promise<void> {
    await gotoHash(this.page, `/${TEST_ORG_SLUG}/infra?tab=${tab}&id=${id}`);
  }

  async expectOnPage(): Promise<void> {
    await expectHashMatches(this.page, /\/infra/);
  }
}
