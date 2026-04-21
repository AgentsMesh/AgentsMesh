import { test } from "../../fixtures";
import { BlocksPage } from "../../pages/blocks.page";

test("Blocks · route opens", async ({ page }) => {
  const blocks = new BlocksPage(page);
  await blocks.goto();
  await blocks.expectOnPage();
});
