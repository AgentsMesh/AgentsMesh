import { test, expect } from "../../fixtures";
import { InfraPage } from "../../pages/infra.page";

test("Infra · route opens (defaults to repositories tab)", async ({ page }) => {
  const infra = new InfraPage(page);
  await infra.goto();
  await infra.expectOnPage();
});

test("Infra · switches to runners tab", async ({ page }) => {
  const infra = new InfraPage(page);
  await infra.gotoTab("runners");
  await expect(page).toHaveURL(/tab=runners/);
});
