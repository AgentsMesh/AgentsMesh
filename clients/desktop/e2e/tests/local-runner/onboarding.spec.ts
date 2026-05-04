import { test, expect } from "../../fixtures";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";

// FIXME(local-runner-e2e): runs against the still-being-provisioned
// macmini-03 self-hosted runner; the registration flow needs the real
// runner CLI to be on PATH so install_binary stub → register → service
// start can transition the card to "running". Skip until that runner is
// in CI rotation and the stub coverage matches the renderer flow.
test.fixme("Local-runner onboarding · install → register → start → registered badge", async ({ page }) => {
  await gotoHash(page, `/${TEST_ORG_SLUG}/workspace`);

  const card = page.locator('text=Register this Mac as a Runner').first();
  await expect(card).toBeVisible({ timeout: 15_000 });

  const button = page.getByRole("button", { name: /Register/i }).first();
  await expect(button).toBeEnabled();
  await button.click();

  const success = page.locator('text=This Mac is registered as a Runner').first();
  await expect(success).toBeVisible({ timeout: 15_000 });
});
