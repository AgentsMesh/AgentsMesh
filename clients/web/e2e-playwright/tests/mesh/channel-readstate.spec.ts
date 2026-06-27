import { test, expect } from "@playwright/test";
import { ChannelsPage } from "../../pages/channels.page";
import { SidebarPage } from "../../pages/sidebar.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

// NOTE: authored-but-unverified — needs the live dev stack to run:
//   bazel run //deploy/dev:up   then   bazel test //clients/web:e2e
// Covers the new single-user channel read-state flows (draft persistence, pin,
// mute, mark-unread). Multi-user realtime flows (unread badge / @me / divider
// from another sender) are a separate two-context spec best authored live.
test.describe("Channel read-state UX", () => {
  let channels: ChannelsPage;
  let sidebar: SidebarPage;

  test.beforeEach(async ({ page }) => {
    clearAuthRateLimit();
    channels = new ChannelsPage(page, TEST_ORG_SLUG);
    sidebar = new SidebarPage(page, TEST_ORG_SLUG);
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("load");
    await sidebar.navigateTo("channels");
  });

  test("draft persists across channel switches and shows [Draft] in the sidebar", async ({ page }) => {
    const a = "E2E Draft A " + Date.now();
    const b = "E2E Draft B " + Date.now();
    await channels.createChannel(a, { visibility: "public" });
    await channels.createChannel(b, { visibility: "public" });

    await channels.selectChannel(a);
    await channels.messageInput.fill("unsent draft text");
    await page.waitForTimeout(600); // debounce save (400ms) settles

    await expect(page.getByText("[Draft]").first()).toBeVisible({ timeout: 5000 });

    // switch away (empty composer) and back (draft restored)
    await channels.selectChannel(b);
    await expect(channels.messageInput).toHaveValue("");
    await channels.selectChannel(a);
    await expect(channels.messageInput).toHaveValue("unsent draft text");
  });

  test("pin moves a channel into the Pinned group", async ({ page }) => {
    const name = "E2E Pin " + Date.now();
    await channels.createChannel(name, { visibility: "public" });

    await channels.getChannelItem(name).click({ button: "right" });
    await page.getByRole("menuitem", { name: "Pin", exact: true }).click();

    await expect(page.getByText("Pinned", { exact: true })).toBeVisible({ timeout: 5000 });
  });

  test("mute shows the muted (BellOff) indicator on the row", async ({ page }) => {
    const name = "E2E Mute " + Date.now();
    await channels.createChannel(name, { visibility: "public" });

    await channels.getChannelItem(name).click({ button: "right" });
    await page.getByRole("menuitem", { name: "Mute", exact: true }).click();

    await expect(page.getByLabel("Muted").first()).toBeVisible({ timeout: 5000 });
  });

  test("mark-as-unread flags an already-read channel", async ({ page }) => {
    const name = "E2E MarkUnread " + Date.now();
    await channels.createChannel(name, { visibility: "public" });
    await channels.selectChannel(name);
    await channels.sendMessage("hello"); // own message → auto-read
    await page.waitForTimeout(600); // markRead (300ms) settles

    await channels.getChannelItem(name).click({ button: "right" });
    await page.getByRole("menuitem", { name: "Mark as unread", exact: true }).click();

    // marked-unread dot carries aria-label "Unread"
    await expect(page.getByLabel("Unread").first()).toBeVisible({ timeout: 5000 });
  });
});
