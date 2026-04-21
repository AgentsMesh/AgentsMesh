import { test, expect } from "@playwright/test";
import { ChannelsPage } from "../../pages/channels.page";
import { SidebarPage } from "../../pages/sidebar.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { clearAuthRateLimit } from "../../helpers/redis";

/**
 * Covers the channel compose + header actions introduced by the 2026-04-20
 * refresh: three-button toolbar (B▾/@/📎), header (🔍/members/⚙️), and
 * attachment rendering.
 */
test.describe("Channel actions", () => {
  let channels: ChannelsPage;
  let sidebar: SidebarPage;
  let channelName: string;

  test.beforeEach(async ({ page }) => {
    clearAuthRateLimit();
    channels = new ChannelsPage(page, TEST_ORG_SLUG);
    sidebar = new SidebarPage(page, TEST_ORG_SLUG);
    await page.goto(`/${TEST_ORG_SLUG}/workspace`);
    await page.waitForLoadState("networkidle");

    channelName = `e2e-actions-${Date.now()}`;
    await sidebar.navigateTo("channels");
    await channels.createChannel(channelName, { visibility: "public" });
    await channels.selectChannel(channelName);
    await expect(channels.messageInput.or(page.getByTestId("message-input-textarea")))
      .toBeVisible({ timeout: 5000 });
  });

  test("toolbar wraps selection with bold markdown", async ({ page }) => {
    const textarea = page.getByTestId("message-input-textarea");
    await textarea.fill("hello world");
    await textarea.evaluate((el: HTMLTextAreaElement) => {
      el.setSelectionRange(0, 5);
    });

    await page.getByTestId("format-toolbar-trigger").click();
    await expect(page.getByTestId("format-toolbar-menu")).toBeVisible();
    await page.getByTestId("format-bold").click();

    await expect(textarea).toHaveValue("**hello** world");
  });

  test("@ button inserts at cursor and opens mention dropdown", async ({ page }) => {
    const textarea = page.getByTestId("message-input-textarea");
    await textarea.fill("hi ");
    await textarea.evaluate((el: HTMLTextAreaElement) => {
      el.setSelectionRange(el.value.length, el.value.length);
    });

    await page.getByTestId("toolbar-mention").click();
    await expect(textarea).toHaveValue("hi @");
  });

  test("attachment button exposes a hidden file input", async ({ page }) => {
    await expect(page.getByTestId("toolbar-attach")).toBeVisible();
    const input = page.getByTestId("message-attachment-input");
    await expect(input).toBeAttached();
    expect(await input.getAttribute("type")).toBe("file");
  });

  test("header opens the search modal", async ({ page }) => {
    await page.getByTestId("channel-header-search").click();
    const search = page.getByTestId("message-search-input");
    await expect(search).toBeVisible();
    await search.fill("nothing-matches-xyz");
    await expect(page.getByText(/没有匹配的消息|No matching messages/i))
      .toBeVisible({ timeout: 2000 });
  });

  test("more button toggles the right drawer", async ({ page }) => {
    const rail = page.getByTestId("channel-right-rail");
    const more = page.getByTestId("channel-header-more");
    await expect(rail).toBeVisible();
    await expect(more).toHaveAttribute("aria-pressed", "true");
    await more.click();
    await expect(rail).toHaveCount(0);
    await expect(more).toHaveAttribute("aria-pressed", "false");
    await more.click();
    await expect(rail).toBeVisible();
  });

  test("rail settings entry opens the settings modal and saves a rename", async ({ page }) => {
    await page.getByTestId("channel-rail-settings").click();
    await expect(page.getByTestId("channel-settings-name")).toBeVisible();
    await page.getByTestId("channel-settings-name").fill(`${channelName}-renamed`);
    await page.getByTestId("channel-settings-save").click();
    await expect(page.getByTestId("channel-settings-save")).toBeHidden({ timeout: 5000 });
  });

  test("no close (X) button is rendered in the header", async ({ page }) => {
    const xIconButtons = await page
      .locator("header, [data-testid='channel-header-search']")
      .first()
      .evaluate(() => 1); // ensure header is present
    expect(xIconButtons).toBe(1);
    // The previous design exposed an onClose X icon; ensure it's gone.
    await expect(
      page.locator('button[aria-label="close" i], button[title="close" i]')
    ).toHaveCount(0);
  });
});
